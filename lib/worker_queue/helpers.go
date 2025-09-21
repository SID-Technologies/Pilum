package workerqueue

import (
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sid-technologies/centurion/lib/errors"
)

// TerminateProcessTree terminates a process and all its child processes.
func TerminateProcessTree(pid int) error {
	var err error
	// Find child processes
	findCmd := exec.Command("pgrep", "-P", fmt.Sprintf("%d", pid))
	output, err := findCmd.Output()
	if err != nil {
		return errors.Wrap(err, "error finding child processes")
	}

	// Parse child PIDs
	childPids := []int{}
	if len(output) > 0 {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if line == "" {
				continue
			}
			var childPid int
			_, err := fmt.Sscanf(line, "%d", &childPid)
			if err == nil {
				childPids = append(childPids, childPid)
			}
		}
	}

	// Terminate children first
	for _, childPid := range childPids {
		err = syscall.Kill(childPid, syscall.SIGTERM)
		if err != nil {
			return errors.Wrap(err, "error terminating child process %d", childPid)
		}
	}

	// Terminate parent
	err = syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		return errors.Wrap(err, "error terminating process %d", pid)
	}

	// Give time to terminate gracefully
	time.Sleep(2 * time.Second)

	// Force kill if still running
	for _, childPid := range childPids {
		err = syscall.Kill(childPid, syscall.SIGKILL)
		if err != nil {
			return errors.Wrap(err, "error force killing child process %d", childPid)
		}
	}
	err = syscall.Kill(pid, syscall.SIGKILL)
	if err != nil {
		return errors.Wrap(err, "error force killing process %d", pid)
	}

	return nil
}

// ExponentialBackoffWithJitter calculates exponential backoff with jitter.
func ExponentialBackoffWithJitter(attempt int, baseDelay float64, maxDelay float64) float64 {
	// Exponential backoff calculation
	delay := math.Min(maxDelay, baseDelay*math.Pow(2, float64(attempt)))

	// Add jitter (random variation between 0.5x and 1.5x of calculated delay)
	//nolint: gosec // It's a random number, not a secret
	jitteredDelay := delay * (0.5 + rand.Float64())

	return jitteredDelay
}
