package workerqueue

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
)

// CaptureWorker executes a command and returns its stdout as a string.
// Unlike CommandWorker, it captures output instead of streaming it, and does not retry.
func CaptureWorker(taskInfo *TaskInfo) (string, error) {
	if taskInfo.Debug {
		output.Debugf("Capturing output for %s", taskInfo.ServiceName)
		output.Debugf("Command: %v", taskInfo.Command)
	}

	var cmd *exec.Cmd
	var workingDir string

	// Determine working directory
	switch taskInfo.ExecutionMode {
	case "root":
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return "", errors.Wrap(err, "error getting current working directory")
		}
	case "service_dir":
		workingDir = taskInfo.Cwd
	default:
		return "", errors.New("invalid execution mode: %s", taskInfo.ExecutionMode)
	}

	// Prepare command
	switch v := taskInfo.Command.(type) {
	case string:
		cmd = exec.Command("sh", "-c", v)
	case []string:
		if len(v) < 1 {
			return "", errors.New("empty command array for %s", taskInfo.ServiceName)
		}
		cmd = exec.Command(v[0], v[1:]...) //nolint:gosec // Command comes from trusted recipe config
	case []any:
		if len(v) < 1 {
			return "", errors.New("empty command array for %s", taskInfo.ServiceName)
		}
		args := make([]string, len(v))
		for i, arg := range v {
			args[i] = fmt.Sprintf("%v", arg)
		}
		cmd = exec.Command(args[0], args[1:]...) //nolint:gosec // Command comes from trusted recipe config
	default:
		return "", errors.New("invalid command type for %s: %T", taskInfo.ServiceName, taskInfo.Command)
	}

	cmd.Dir = workingDir

	// Prepare environment variables
	cmd.Env = os.Environ()
	for key, value := range taskInfo.EnvVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return "", errors.Wrap(err, "error starting command for %s", taskInfo.ServiceName)
	}

	// Set up timeout context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(taskInfo.Timeout)*time.Second)
	defer cancel()

	// Monitor command execution
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = TerminateProcessTree(cmd.Process.Pid)
		return "", errors.New("command timed out for %s", taskInfo.ServiceName)
	case err := <-done:
		if err != nil {
			return "", errors.New("command failed for %s: %s", taskInfo.ServiceName, stderr.String())
		}
		return stdout.String(), nil
	}
}
