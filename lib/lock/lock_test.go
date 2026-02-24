package lock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAcquireAndRelease(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	err := Acquire(tmpDir, "deploy", []string{"api", "worker"}, false)
	require.NoError(t, err)

	// Verify lock file exists and is valid
	fp := FilePath(tmpDir)
	data, err := os.ReadFile(fp)
	require.NoError(t, err)

	var info Info
	require.NoError(t, json.Unmarshal(data, &info))
	require.Equal(t, os.Getpid(), info.PID)
	require.Equal(t, "deploy", info.Command)
	require.Equal(t, []string{"api", "worker"}, info.Services)
	require.NotZero(t, info.Hostname)
	require.False(t, info.Timestamp.IsZero())

	// Release
	Release(tmpDir)

	_, err = os.Stat(fp)
	require.True(t, os.IsNotExist(err))
}

func TestAcquireFailsWhenLocked(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	err := Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	// Second acquire should fail
	err = Acquire(tmpDir, "build", []string{"worker"}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "deployment locked")
	require.Contains(t, err.Error(), "deploy")

	Release(tmpDir)
}

func TestAcquireForceOverridesExistingLock(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	err := Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	// Force acquire should succeed
	err = Acquire(tmpDir, "build", []string{"worker"}, true)
	require.NoError(t, err)

	// Verify new lock info
	fp := FilePath(tmpDir)
	data, err := os.ReadFile(fp)
	require.NoError(t, err)

	var info Info
	require.NoError(t, json.Unmarshal(data, &info))
	require.Equal(t, "build", info.Command)
	require.Equal(t, []string{"worker"}, info.Services)

	Release(tmpDir)
}

func TestStaleLockCleanedByTime(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a lock that's older than staleDuration
	dir := filepath.Join(tmpDir, lockDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	info := Info{
		PID:       os.Getpid(), // Use our own PID (alive) to isolate time-based test
		Hostname:  "old-host",
		Timestamp: time.Now().Add(-2 * time.Hour), // 2 hours ago
		Services:  []string{"old-svc"},
		Command:   "old-deploy",
	}
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(FilePath(tmpDir), data, 0o644))

	// Should succeed because old lock is stale
	err = Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	Release(tmpDir)
}

func TestStaleLockCleanedByDeadPID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a lock with a PID that almost certainly doesn't exist
	// Use current hostname so the PID liveness check is triggered
	hostname, _ := os.Hostname()
	dir := filepath.Join(tmpDir, lockDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	info := Info{
		PID:       999999999,  // Very unlikely to be a real process
		Hostname:  hostname,   // Same host — PID liveness will be checked
		Timestamp: time.Now(), // Recent, so time-based check won't trigger
		Services:  []string{"dead-svc"},
		Command:   "dead-deploy",
	}
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(FilePath(tmpDir), data, 0o644))

	// Should succeed because the PID is dead (and hostname matches)
	err = Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	Release(tmpDir)
}

func TestLockFromDifferentHostNotCleanedByPID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a lock from a different host with a dead PID
	dir := filepath.Join(tmpDir, lockDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	info := Info{
		PID:       999999999,
		Hostname:  "other-host", // Different host — PID check should be skipped
		Timestamp: time.Now(),   // Recent, so time-based check won't trigger
		Services:  []string{"svc"},
		Command:   "deploy",
	}
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(FilePath(tmpDir), data, 0o644))

	// Should fail because we can't verify PID on a different host
	// and the lock isn't old enough for time-based cleanup
	err = Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "deployment locked")

	Release(tmpDir)
}

func TestLockFromDifferentHostCleanedByAge(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create an old lock from a different host
	dir := filepath.Join(tmpDir, lockDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))

	info := Info{
		PID:       999999999,
		Hostname:  "other-host",
		Timestamp: time.Now().Add(-2 * time.Hour), // Stale by age
		Services:  []string{"svc"},
		Command:   "deploy",
	}
	data, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(FilePath(tmpDir), data, 0o644))

	// Should succeed — time-based staleness applies regardless of hostname
	err = Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	Release(tmpDir)
}

func TestReleaseIsIdempotent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Release when no lock exists — should not panic or error
	Release(tmpDir)
	Release(tmpDir)

	// Acquire, release twice
	err := Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	Release(tmpDir)
	Release(tmpDir) // Second release should be fine
}

func TestFilePath(t *testing.T) {
	t.Parallel()

	fp := FilePath("/my/project")
	require.Equal(t, filepath.Join("/my/project", ".pilum", "deploy.lock"), fp)
}

func TestAcquireCorruptedLockFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a corrupted lock file
	dir := filepath.Join(tmpDir, lockDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(FilePath(tmpDir), []byte("not json"), 0o644))

	// Should clean up corrupted lock and succeed
	err := Acquire(tmpDir, "deploy", []string{"api"}, false)
	require.NoError(t, err)

	Release(tmpDir)
}
