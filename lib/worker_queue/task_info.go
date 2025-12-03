package workerqueue

// TaskInfo holds configuration for a command execution task.
type TaskInfo struct {
	Command       any               // string or []string
	Cwd           string            // Working directory
	ServiceName   string            // Name for logging
	ExecutionMode string            // "root" or "service_dir"
	EnvVars       map[string]string // Environment variables
	BuildFlags    map[string]any    // Build flags (string or []string)
	Timeout       int               // Timeout in seconds
	Debug         bool              // Enable debug output
	Retries       int               // Number of retries
}

// NewTaskInfo creates a new TaskInfo with default values.
func NewTaskInfo(
	command any,
	cwd string,
	serviceName string,
	executionMode string,
	envVars map[string]string,
	buildFlags map[string]any,
	timeout int,
	debug bool,
	retries int,
) *TaskInfo {
	if envVars == nil {
		envVars = make(map[string]string)
	}
	if buildFlags == nil {
		buildFlags = make(map[string]any)
	}
	if timeout == 0 {
		timeout = 300
	}
	if retries == 0 {
		retries = 3
	}

	return &TaskInfo{
		Command:       command,
		Cwd:           cwd,
		ServiceName:   serviceName,
		ExecutionMode: executionMode,
		EnvVars:       envVars,
		BuildFlags:    buildFlags,
		Timeout:       timeout,
		Debug:         debug,
		Retries:       retries,
	}
}
