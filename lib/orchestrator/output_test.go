package orchestrator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0ms",
		},
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "one millisecond",
			duration: 1 * time.Millisecond,
			expected: "1ms",
		},
		{
			name:     "999 milliseconds",
			duration: 999 * time.Millisecond,
			expected: "999ms",
		},
		{
			name:     "exactly one second",
			duration: 1 * time.Second,
			expected: "1.0s",
		},
		{
			name:     "seconds with decimal",
			duration: 2500 * time.Millisecond,
			expected: "2.5s",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			expected: "59.0s",
		},
		{
			name:     "exactly one minute",
			duration: 1 * time.Minute,
			expected: "1.0m",
		},
		{
			name:     "minutes with decimal",
			duration: 90 * time.Second,
			expected: "1.5m",
		},
		{
			name:     "multiple minutes",
			duration: 5 * time.Minute,
			expected: "5.0m",
		},
		{
			name:     "mixed minutes and seconds",
			duration: 3*time.Minute + 30*time.Second,
			expected: "3.5m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatDuration(tt.duration)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cmd      any
		expected string
	}{
		{
			name:     "nil command",
			cmd:      nil,
			expected: "",
		},
		{
			name:     "string command",
			cmd:      "echo hello",
			expected: "echo hello",
		},
		{
			name:     "empty string",
			cmd:      "",
			expected: "",
		},
		{
			name:     "string slice",
			cmd:      []string{"go", "build", "-o", "app"},
			expected: "go build -o app",
		},
		{
			name:     "empty string slice",
			cmd:      []string{},
			expected: "",
		},
		{
			name:     "any slice",
			cmd:      []any{"docker", "run", "-p", 8080},
			expected: "docker run -p 8080",
		},
		{
			name:     "empty any slice",
			cmd:      []any{},
			expected: "",
		},
		{
			name:     "int command",
			cmd:      42,
			expected: "42",
		},
		{
			name:     "bool command",
			cmd:      true,
			expected: "true",
		},
		{
			name:     "struct command",
			cmd:      struct{ Name string }{Name: "test"},
			expected: "{test}",
		},
		{
			name:     "single element string slice",
			cmd:      []string{"ls"},
			expected: "ls",
		},
		{
			name:     "mixed types in any slice",
			cmd:      []any{"timeout", 30, "command", true},
			expected: "timeout 30 command true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatCommand(tt.cmd)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputManagerPadName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		maxNameLen int
		inputName  string
		expected   string
	}{
		{
			name:       "shorter name gets padded",
			maxNameLen: 20,
			inputName:  "myservice",
			expected:   "myservice           ",
		},
		{
			name:       "exact length name no padding",
			maxNameLen: 9,
			inputName:  "myservice",
			expected:   "myservice",
		},
		{
			name:       "longer name no truncation",
			maxNameLen: 5,
			inputName:  "myservice",
			expected:   "myservice",
		},
		{
			name:       "empty name",
			maxNameLen: 10,
			inputName:  "",
			expected:   "          ",
		},
		{
			name:       "zero maxNameLen uses default 20",
			maxNameLen: 0,
			inputName:  "svc",
			expected:   "svc                 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			om := NewOutputManager()
			om.SetMaxNameLength(tt.maxNameLen)
			result := om.padName(tt.inputName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNewOutputManager(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	require.NotNil(t, om)
	require.True(t, om.useColors)
	require.NotNil(t, om.serviceState)
}

func TestOutputManagerSetMaxNameLength(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	require.Equal(t, 0, om.maxNameLen)

	om.SetMaxNameLength(25)
	require.Equal(t, 25, om.maxNameLen)

	om.SetMaxNameLength(10)
	require.Equal(t, 10, om.maxNameLen)
}

func TestOutputManagerPrintHeader(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()

	require.NotPanics(t, func() {
		om.PrintHeader("Deploying 3 services")
	})
}

func TestOutputManagerPrintStepHeader(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()

	require.NotPanics(t, func() {
		om.PrintStepHeader(1, 5, "build")
		om.PrintStepHeader(3, 5, "deploy")
	})
}

func TestOutputManagerPrintRunning(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintRunning("myservice", "building")
	})

	require.Equal(t, "running", om.serviceState["myservice"])
}

func TestOutputManagerPrintSuccess(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintSuccess("myservice", 2*time.Second)
	})

	require.Equal(t, "success", om.serviceState["myservice"])
}

func TestOutputManagerPrintFailure(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintFailure("myservice", nil)
	})
	require.Equal(t, "failed", om.serviceState["myservice"])
}

func TestOutputManagerPrintFailureWithError(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintFailure("myservice", &testError{msg: "build failed"})
	})
	require.Equal(t, "failed", om.serviceState["myservice"])
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestOutputManagerPrintSkipped(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintSkipped("myservice", "no recipe")
	})

	require.Equal(t, "skipped", om.serviceState["myservice"])
}

func TestOutputManagerPrintDryRun(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintDryRun("myservice", "build", []string{"go", "build"})
	})
}

func TestOutputManagerPrintDryRunNilCommand(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintDryRun("myservice", "build", nil)
	})
}

func TestOutputManagerPrintDryRunStringCommand(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	require.NotPanics(t, func() {
		om.PrintDryRun("myservice", "build", "make build")
	})
}

func TestOutputManagerPrintInfo(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()

	require.NotPanics(t, func() {
		om.PrintInfo("Some informational message")
	})
}

func TestOutputManagerPrintCompleteAllSuccess(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()

	results := []TaskResult{
		{ServiceName: "svc1", StepName: "build", Success: true, Duration: time.Second},
		{ServiceName: "svc2", StepName: "build", Success: true, Duration: 2 * time.Second},
	}

	require.NotPanics(t, func() {
		om.PrintComplete(results)
	})
}

func TestOutputManagerPrintCompleteWithFailures(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()

	results := []TaskResult{
		{ServiceName: "svc1", StepName: "build", Success: true, Duration: time.Second},
		{ServiceName: "svc2", StepName: "build", Success: false, Duration: 500 * time.Millisecond},
		{ServiceName: "svc3", StepName: "build", Success: false, Duration: 300 * time.Millisecond},
	}

	require.NotPanics(t, func() {
		om.PrintComplete(results)
	})
}

func TestOutputManagerPrintCompleteEmpty(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()

	require.NotPanics(t, func() {
		om.PrintComplete(nil)
	})
}

func TestOutputManagerConcurrentAccess(t *testing.T) {
	t.Parallel()

	om := NewOutputManager()
	om.SetMaxNameLength(20)

	// Test that concurrent access doesn't cause race conditions
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			om.PrintRunning("service", "step")
			om.PrintSuccess("service", time.Second)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
