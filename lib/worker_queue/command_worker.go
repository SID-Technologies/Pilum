package workerqueue

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/sid-technologies/centurion/lib/errors"
)

// CommandWorker executes commands with configurable execution context.
func CommandWorker(taskInfo *TaskInfo) (bool, error) {
	if taskInfo.Debug {
		log.Printf("Executing command for %s\n", taskInfo.ServiceName)
		log.Printf("Command: %v\n", taskInfo.Command)
		log.Printf("Working directory: %s\n", taskInfo.Cwd)
		log.Printf("Execution mode: %s\n", taskInfo.ExecutionMode)
		log.Printf("Timeout: %d\n", taskInfo.Timeout)
		log.Printf("Environment variables: %v\n", taskInfo.EnvVars)
		log.Println()
	}

	for attempt := 0; attempt <= taskInfo.Retries; attempt++ {
		var cmd *exec.Cmd
		var workingDir string

		// Determine working directory
		switch taskInfo.ExecutionMode {
		case "root":
			var err error
			workingDir, err = os.Getwd()
			if err != nil {
				log.Printf("Error getting current working directory: %v\n", err)
				return false, nil
			}
		case "service_dir":
			workingDir = taskInfo.Cwd
		default:
			log.Printf("Invalid execution mode: %s\n", taskInfo.ExecutionMode)
			return false, nil
		}

		// Prepare command
		switch v := taskInfo.Command.(type) {
		case string:
			cmd = exec.Command("sh", "-c", v)
		case []string:
			if len(v) < 1 {
				log.Printf("Empty command array for %s\n", taskInfo.ServiceName)
				return false, nil
			}
			cmd = exec.Command(v[0], v[1:]...)
		default:
			log.Printf("Invalid command type for %s\n", taskInfo.ServiceName)
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
		_, err := cmd.StdoutPipe()
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

		// Set up timeout context
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(taskInfo.Timeout)*time.Second)
		defer cancel()

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
			// Command completed
			if err == nil {
				return true, nil
			}

			// Read error output
			stderrBytes := make([]byte, 1024)
			n, _ := stderr.Read(stderrBytes)
			errorOutput := string(stderrBytes[:n])

			log.Println("--------------------")
			log.Printf("Command failed for %s\n", taskInfo.ServiceName)
			log.Printf("Error output: %s\n", errorOutput)

			// Retry if not the last attempt
			if attempt < taskInfo.Retries {
				retryDelay := ExponentialBackoffWithJitter(attempt, 1.0, 60.0)
				log.Printf("Retrying for %s in %.2f seconds...\n", taskInfo.ServiceName, retryDelay)
				log.Println("--------------------")
				time.Sleep(time.Duration(retryDelay * float64(time.Second)))
			}
		}
	}

	return false, nil
}
