package workerqueue_test

import (
	"testing"

	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/stretchr/testify/require"
)

func TestCaptureWorkerStringCommand(t *testing.T) {
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

	out, err := workerqueue.CaptureWorker(taskInfo)

	require.NoError(t, err)
	require.Equal(t, "hello\n", out)
}

func TestCaptureWorkerStringSliceCommand(t *testing.T) {
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

	out, err := workerqueue.CaptureWorker(taskInfo)

	require.NoError(t, err)
	require.Equal(t, "hello world\n", out)
}

func TestCaptureWorkerAnySliceCommand(t *testing.T) {
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

	out, err := workerqueue.CaptureWorker(taskInfo)

	require.NoError(t, err)
	require.Equal(t, "hello\n", out)
}

func TestCaptureWorkerEmptyStringSlice(t *testing.T) {
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

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
}

func TestCaptureWorkerInvalidCommandType(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		12345,
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
}

func TestCaptureWorkerFailingCommand(t *testing.T) {
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
		0,
	)

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
}

func TestCaptureWorkerTimeout(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"sleep 10",
		"",
		"test-service",
		"root",
		nil,
		nil,
		1,
		false,
		0,
	)

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
}

func TestCaptureWorkerMultilineOutput(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo line1 && echo line2 && echo line3",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	out, err := workerqueue.CaptureWorker(taskInfo)

	require.NoError(t, err)
	require.Equal(t, "line1\nline2\nline3\n", out)
}

func TestCaptureWorkerWithEnvVars(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"MY_TEST_VAR": "captured_value",
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

	out, err := workerqueue.CaptureWorker(taskInfo)

	require.NoError(t, err)
	require.Equal(t, "captured_value\n", out)
}

func TestCaptureWorkerServiceDirMode(t *testing.T) {
	t.Parallel()

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

	out, err := workerqueue.CaptureWorker(taskInfo)

	require.NoError(t, err)
	require.Contains(t, out, tmpDir)
}

func TestCaptureWorkerInvalidExecutionMode(t *testing.T) {
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

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
}

// Regression: when a command fails and only writes to stdout (not stderr),
// the error message should include the stdout output as a fallback.
func TestCaptureWorkerFailingCommandCapturesStdout(t *testing.T) {
	t.Parallel()

	// Write error info to stdout (not stderr) then exit with failure
	taskInfo := workerqueue.NewTaskInfo(
		"echo 'error details on stdout' && exit 1",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
	require.Contains(t, err.Error(), "error details on stdout")
}

// Regression: when stderr has content, it should be preferred over stdout.
func TestCaptureWorkerFailingCommandPrefersStderr(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"echo 'stdout info' && echo 'stderr info' >&2 && exit 1",
		"",
		"test-service",
		"root",
		nil,
		nil,
		10,
		false,
		0,
	)

	_, err := workerqueue.CaptureWorker(taskInfo)

	require.Error(t, err)
	require.Contains(t, err.Error(), "stderr info")
}
