package workers

import (
	"math"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
)

// TerminateProcessTree terminates a process and its direct children, sending
// SIGTERM and then SIGKILL. A process that has already exited is not an
// error, and neither is a missing pgrep: the parent is still killed.
func TerminateProcessTree(pid int) error {
	childPids := childProcesses(pid)

	// Terminate children first
	for _, childPid := range childPids {
		err := signalProcess(childPid, syscall.SIGTERM)
		if err != nil {
			return errors.Wrap(err, "error terminating child process %d", childPid)
		}
	}

	// Terminate parent
	err := signalProcess(pid, syscall.SIGTERM)
	if err != nil {
		return errors.Wrap(err, "error terminating process %d", pid)
	}

	// Give time to terminate gracefully
	time.Sleep(2 * time.Second)

	// Force kill if still running
	for _, childPid := range childPids {
		err = signalProcess(childPid, syscall.SIGKILL)
		if err != nil {
			return errors.Wrap(err, "error force killing child process %d", childPid)
		}
	}
	err = signalProcess(pid, syscall.SIGKILL)
	if err != nil {
		return errors.Wrap(err, "error force killing process %d", pid)
	}

	return nil
}

// childProcesses lists pid's direct children. pgrep exits 1 when there are
// none, and some images (golang, distroless) don't ship it, so any failure
// means no children.
func childProcesses(pid int) []int {
	output, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output() //nolint:gosec // pgrep with controlled PID argument
	if err != nil {
		return nil
	}

	childPids := []int{}
	for _, line := range strings.Fields(string(output)) {
		childPid, err := strconv.Atoi(line)
		if err == nil {
			childPids = append(childPids, childPid)
		}
	}
	return childPids
}

// signalProcess sends sig to pid, treating a process that is already gone as success.
func signalProcess(pid int, sig syscall.Signal) error {
	err := syscall.Kill(pid, sig)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

// ExponentialBackoffWithJitter calculates exponential backoff with jitter.
func ExponentialBackoffWithJitter(attempt int, baseDelay float64, maxDelay float64) float64 {
	// Exponential backoff calculation
	delay := math.Min(maxDelay, baseDelay*math.Pow(2, float64(attempt)))

	// Add jitter (random variation between 0.5x and 1.5x of calculated delay)
	//nolint:gosec // It's a random number, not a secret
	jitteredDelay := delay * (0.5 + rand.Float64())

	return jitteredDelay
}
