package workerqueue

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
)

// CommandWorker executes commands with configurable execution context.
func CommandWorker(taskInfo *TaskInfo) (bool, error) {
	if taskInfo.Debug {
		output.Debugf("Executing command for %s", taskInfo.ServiceName)
		output.Debugf("Command: %v", taskInfo.Command)
		output.Debugf("Working directory: %s", taskInfo.Cwd)
		output.Debugf("Execution mode: %s", taskInfo.ExecutionMode)
		output.Debugf("Timeout: %d", taskInfo.Timeout)
		output.Debugf("Environment variables: %v", taskInfo.EnvVars)
	}

	var lastError string

	for attempt := 0; attempt <= taskInfo.Retries; attempt++ {
		var cmd *exec.Cmd
		var workingDir string

		// Determine working directory
		switch taskInfo.ExecutionMode {
		case "root":
			var err error
			workingDir, err = os.Getwd()
			if err != nil {
				output.Debugf("Error getting current working directory: %v", err)
				return false, nil
			}
		case "service_dir":
			workingDir = taskInfo.Cwd
		default:
			output.Debugf("Invalid execution mode: %s", taskInfo.ExecutionMode)
			return false, nil
		}

		// Prepare command
		switch v := taskInfo.Command.(type) {
		case string:
			cmd = exec.Command("sh", "-c", v)
		case []string:
			if len(v) < 1 {
				output.Debugf("Empty command array for %s", taskInfo.ServiceName)
				return false, nil
			}
			cmd = exec.Command(v[0], v[1:]...) //nolint:gosec // Command comes from trusted recipe config
		case []any:
			// Handle YAML parsed arrays ([]interface{})
			if len(v) < 1 {
				output.Debugf("Empty command array for %s", taskInfo.ServiceName)
				return false, nil
			}
			args := make([]string, len(v))
			for i, arg := range v {
				args[i] = fmt.Sprintf("%v", arg)
			}
			cmd = exec.Command(args[0], args[1:]...) //nolint:gosec // Command comes from trusted recipe config
		default:
			output.Debugf("Invalid command type for %s: %T", taskInfo.ServiceName, taskInfo.Command)
			return false, nil
		}

		// Set working directory
		cmd.Dir = workingDir

		// Prepare environment variables
		cmd.Env = os.Environ()
		for key, value := range taskInfo.EnvVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		// Set up output pipes
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return false, errors.Wrap(err, "error creating stdout pipe for %s", taskInfo.ServiceName)
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return false, errors.Wrap(err, "error creating stderr pipe for %s", taskInfo.ServiceName)
		}

		// Start command
		if err := cmd.Start(); err != nil {
			if attempt < taskInfo.Retries {
				retryDelay := ExponentialBackoffWithJitter(attempt, 1.0, 60.0)
				time.Sleep(time.Duration(retryDelay * float64(time.Second)))

				continue
			}

			return false, errors.Wrap(err, "error starting command for %s", taskInfo.ServiceName)
		}

		// Always capture stderr into a buffer.
		// In verbose mode, also stream it to the terminal via TeeReader.
		var stderrBuf bytes.Buffer
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			if output.IsVerbose() {
				tee := io.TeeReader(stderr, &stderrBuf)
				streamOutput(tee, taskInfo.ServiceName, true)
			} else {
				_, _ = io.Copy(&stderrBuf, stderr)
			}
		}()

		// Drain stdout (stream in verbose mode, discard otherwise)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if output.IsVerbose() {
				streamOutput(stdout, taskInfo.ServiceName, false)
			} else {
				_, _ = io.Copy(io.Discard, stdout)
			}
		}()

		// Set up timeout context
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(taskInfo.Timeout)*time.Second)
		defer cancel() //nolint:revive // defer in loop is intentional for cleanup on each iteration

		// Monitor command execution
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		// Wait for command to complete or timeout
		select {
		case <-ctx.Done():
			err := TerminateProcessTree(cmd.Process.Pid)
			if err != nil {
				return false, errors.Wrap(err, "error terminating process tree for %s", taskInfo.ServiceName)
			}

			return false, errors.New("command timed out")
		case err := <-done:
			// Wait for output goroutines to finish draining pipes
			wg.Wait()

			// Command completed
			if err == nil {
				return true, nil
			}

			// Capture stderr for error reporting
			lastError = strings.TrimSpace(stderrBuf.String())

			output.Debugf("Command failed for %s", taskInfo.ServiceName)
			output.Debugf("Error output: %s", lastError)

			// Retry if not the last attempt
			if attempt < taskInfo.Retries {
				retryDelay := ExponentialBackoffWithJitter(attempt, 1.0, 60.0)
				output.Debugf("Retrying for %s in %.2f seconds...", taskInfo.ServiceName, retryDelay)
				time.Sleep(time.Duration(retryDelay * float64(time.Second)))
			}
		}
	}

	// All retries exhausted — return the error with stderr output
	if lastError != "" {
		return false, errors.New("command failed for %s:\n%s", taskInfo.ServiceName, lastError)
	}

	return false, errors.New("command failed for %s", taskInfo.ServiceName)
}

// streamOutput reads from a pipe and prints lines with service name prefix.
func streamOutput(r io.Reader, serviceName string, isStderr bool) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if isStderr {
			output.VerboseStderr(serviceName, line)
		} else {
			output.VerboseStdout(serviceName, line)
		}
	}
}
