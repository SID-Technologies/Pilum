package workerqueue_test

import (
	"os"
	"path/filepath"
	"testing"

	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/stretchr/testify/require"
)

func TestCommandWorkerStringCommand(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo hello",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerStringSliceCommand(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		[]string{"echo", "hello", "world"},
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerAnySliceCommand(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		[]any{"echo", "hello"},
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerEmptyStringSlice(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		[]string{},
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerEmptyAnySlice(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		[]any{},
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerInvalidCommandType(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		12345, // Invalid type
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerInvalidExecutionMode(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo hello",
		"",
		"test-service",
		"invalid_mode",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerServiceDirMode(t *testing.T) {
	t.Parallel()

	// Use a temp directory as the service directory
	tmpDir := t.TempDir()

	taskInfo := workerqueue.NewTaskInfo(
		"pwd",
		tmpDir,
		"test-service",
		"service_dir",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerWithEnvVars(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"MY_TEST_VAR": "test_value",
	}

	taskInfo := workerqueue.NewTaskInfo(
		"echo $MY_TEST_VAR",
		"",
		"test-service",
		"root",
		envVars,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerFailingCommand(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"exit 1",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0, // No retries
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerTimeout(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"sleep 10",
		"",
		"test-service",
		"root",
		nil,
		nil,
		1, // 1 second timeout
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
	require.Error(t, err)
	// Error could be timeout or process termination error
	require.True(t, err != nil)
}

func TestCommandWorkerWithDebug(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo debug test",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		true, // Debug enabled
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerWritesFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	taskInfo := workerqueue.NewTaskInfo(
		"echo hello > "+testFile,
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(testFile)
	require.NoError(t, err)
}

func TestCommandWorkerNonExistentCommand(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		[]string{"nonexistent_command_xyz_123"},
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, _ := workerqueue.CommandWorker(taskInfo)

	require.False(t, success)
}

func TestCommandWorkerMultipleCommands(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo one && echo two && echo three",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerWithPipe(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo hello | cat",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}

func TestCommandWorkerLongOutput(t *testing.T) {
	t.Parallel()

	// Generate a command that produces a lot of output
	taskInfo := workerqueue.NewTaskInfo(
		"seq 1 1000",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	success, err := workerqueue.CommandWorker(taskInfo)

	require.True(t, success)
	require.NoError(t, err)
}
