// Package lock provides deployment locking to prevent concurrent pipeline runs.
// Lock files are stored at .pilum/deploy.lock and contain metadata about
// the lock holder for debugging and stale lock detection.
package lock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
)

const (
	lockDir  = ".pilum"
	lockFile = "deploy.lock"

	// staleDuration is the maximum age of a lock before it's considered stale.
	staleDuration = 1 * time.Hour
)

// Info contains metadata about who holds the deployment lock.
type Info struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
	Services  []string  `json:"services"`
	Command   string    `json:"command"`
}

// FilePath returns the lock file path under the given project root.
func FilePath(projectRoot string) string {
	return filepath.Join(projectRoot, lockDir, lockFile)
}

// Acquire attempts to create a deployment lock. If force is true, any existing
// lock is removed first. Returns an error with lock holder details if another
// active deployment is running.
func Acquire(projectRoot, command string, serviceNames []string, force bool) error {
	fp := FilePath(projectRoot)

	if force {
		_ = os.Remove(fp)
	}

	// Ensure .pilum directory exists
	dir := filepath.Join(projectRoot, lockDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.Wrap(err, "creating %s directory", lockDir)
	}

	// Attempt atomic file creation
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if !os.IsExist(err) {
			return errors.Wrap(err, "creating lock file")
		}

		// Lock file exists — check if it's stale
		existing, readErr := readLock(fp)
		if readErr != nil {
			// Can't read it — remove and retry
			_ = os.Remove(fp)
			return Acquire(projectRoot, command, serviceNames, false)
		}

		if isStale(existing) {
			_ = os.Remove(fp)
			return Acquire(projectRoot, command, serviceNames, false)
		}

		return errors.New(
			"deployment locked by PID %d on %s (started %s, command: %s, services: %v). Use --force to override",
			existing.PID, existing.Hostname,
			existing.Timestamp.Format(time.RFC3339),
			existing.Command, existing.Services,
		)
	}
	defer f.Close()

	hostname, _ := os.Hostname()
	info := Info{
		PID:       os.Getpid(),
		Hostname:  hostname,
		Timestamp: time.Now(),
		Services:  serviceNames,
		Command:   command,
	}

	data, err := json.Marshal(info)
	if err != nil {
		// Clean up the file we created
		_ = os.Remove(fp)
		return errors.Wrap(err, "marshaling lock info")
	}

	if _, err := f.Write(data); err != nil {
		_ = os.Remove(fp)
		return errors.Wrap(err, "writing lock file")
	}

	return nil
}

// Release removes the deployment lock. It is safe to call even if no lock exists.
func Release(projectRoot string) {
	_ = os.Remove(FilePath(projectRoot))
}

// readLock reads and parses the lock file.
func readLock(fp string) (Info, error) {
	data, err := os.ReadFile(fp)
	if err != nil {
		return Info{}, err
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return Info{}, err
	}
	return info, nil
}

// isStale returns true if the lock is older than staleDuration or the PID is dead.
func isStale(info Info) bool {
	// Check age
	if time.Since(info.Timestamp) > staleDuration {
		return true
	}

	// Check if process is still alive
	if info.PID > 0 {
		proc, err := os.FindProcess(info.PID)
		if err != nil {
			return true
		}
		// Signal 0 checks if process exists without actually sending a signal
		err = proc.Signal(syscall.Signal(0))
		if err != nil {
			return true
		}
	}

	return false
}
