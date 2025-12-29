package workerqueue_test

import (
	"testing"

	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/stretchr/testify/require"
)

func TestNewTaskInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		command         any
		cwd             string
		serviceName     string
		executionMode   string
		envVars         map[string]string
		buildFlags      map[string]any
		timeout         int
		debug           bool
		retries         int
		expectedTimeout int
		expectedRetries int
	}{
		{
			name:            "all values provided",
			command:         "go build",
			cwd:             "/path/to/service",
			serviceName:     "myservice",
			executionMode:   "service_dir",
			envVars:         map[string]string{"GO111MODULE": "on"},
			buildFlags:      map[string]any{"ldflags": "-s -w"},
			timeout:         120,
			debug:           true,
			retries:         5,
			expectedTimeout: 120,
			expectedRetries: 5,
		},
		{
			name:            "default timeout when zero",
			command:         "go build",
			cwd:             "",
			serviceName:     "svc",
			executionMode:   "root",
			envVars:         nil,
			buildFlags:      nil,
			timeout:         0,
			debug:           false,
			retries:         0,
			expectedTimeout: 300, // default
			expectedRetries: 3,   // default
		},
		{
			name:            "nil envVars and buildFlags",
			command:         []string{"go", "build"},
			cwd:             "",
			serviceName:     "svc",
			executionMode:   "root",
			envVars:         nil,
			buildFlags:      nil,
			timeout:         60,
			debug:           false,
			retries:         2,
			expectedTimeout: 60,
			expectedRetries: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			taskInfo := workerqueue.NewTaskInfo(
				tt.command,
				tt.cwd,
				tt.serviceName,
				tt.executionMode,
				tt.envVars,
				tt.buildFlags,
				tt.timeout,
				tt.debug,
				tt.retries,
			)

			require.NotNil(t, taskInfo)
			require.Equal(t, tt.command, taskInfo.Command)
			require.Equal(t, tt.cwd, taskInfo.Cwd)
			require.Equal(t, tt.serviceName, taskInfo.ServiceName)
			require.Equal(t, tt.executionMode, taskInfo.ExecutionMode)
			require.Equal(t, tt.debug, taskInfo.Debug)
			require.Equal(t, tt.expectedTimeout, taskInfo.Timeout)
			require.Equal(t, tt.expectedRetries, taskInfo.Retries)
			require.NotNil(t, taskInfo.EnvVars)
			require.NotNil(t, taskInfo.BuildFlags)
		})
	}
}

func TestNewTaskInfoEnvVarsInitialized(t *testing.T) {
	t.Parallel()

	taskInfo := workerqueue.NewTaskInfo(
		"command",
		"",
		"svc",
		"root",
		nil, // nil envVars
		nil, // nil buildFlags
		60,
		false,
		1,
	)

	// Should be initialized to empty maps, not nil
	require.NotNil(t, taskInfo.EnvVars)
	require.NotNil(t, taskInfo.BuildFlags)
	require.Empty(t, taskInfo.EnvVars)
	require.Empty(t, taskInfo.BuildFlags)
}

func TestNewTaskInfoWithEnvVars(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"GO111MODULE": "on",
		"CGO_ENABLED": "0",
	}

	taskInfo := workerqueue.NewTaskInfo(
		"go build",
		"/path",
		"svc",
		"root",
		envVars,
		nil,
		60,
		false,
		1,
	)

	require.Equal(t, "on", taskInfo.EnvVars["GO111MODULE"])
	require.Equal(t, "0", taskInfo.EnvVars["CGO_ENABLED"])
}

func TestNewTaskInfoWithBuildFlags(t *testing.T) {
	t.Parallel()

	buildFlags := map[string]any{
		"ldflags": []string{"-s", "-w"},
		"gcflags": "-N -l",
	}

	taskInfo := workerqueue.NewTaskInfo(
		"go build",
		"/path",
		"svc",
		"root",
		nil,
		buildFlags,
		60,
		false,
		1,
	)

	require.Equal(t, []string{"-s", "-w"}, taskInfo.BuildFlags["ldflags"])
	require.Equal(t, "-N -l", taskInfo.BuildFlags["gcflags"])
}

func TestNewTaskInfoCommandTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command any
	}{
		{
			name:    "string command",
			command: "go build -o app",
		},
		{
			name:    "string slice command",
			command: []string{"go", "build", "-o", "app"},
		},
		{
			name:    "any slice command",
			command: []any{"go", "build", "-o", "app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			taskInfo := workerqueue.NewTaskInfo(
				tt.command,
				"",
				"svc",
				"root",
				nil,
				nil,
				60,
				false,
				1,
			)

			require.Equal(t, tt.command, taskInfo.Command)
		})
	}
}

func TestTaskInfoStruct(t *testing.T) {
	t.Parallel()

	// Test that TaskInfo can be created directly
	taskInfo := &workerqueue.TaskInfo{
		Command:       "echo hello",
		Cwd:           "/tmp",
		ServiceName:   "test-service",
		ExecutionMode: "root",
		EnvVars:       map[string]string{"KEY": "value"},
		BuildFlags:    map[string]any{"flag": "value"},
		Timeout:       30,
		Debug:         true,
		Retries:       2,
	}

	require.Equal(t, "echo hello", taskInfo.Command)
	require.Equal(t, "/tmp", taskInfo.Cwd)
	require.Equal(t, "test-service", taskInfo.ServiceName)
	require.Equal(t, "root", taskInfo.ExecutionMode)
	require.Equal(t, "value", taskInfo.EnvVars["KEY"])
	require.Equal(t, "value", taskInfo.BuildFlags["flag"])
	require.Equal(t, 30, taskInfo.Timeout)
	require.True(t, taskInfo.Debug)
	require.Equal(t, 2, taskInfo.Retries)
}
