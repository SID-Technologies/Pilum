package recepie

import (
	"github.com/sid-technologies/centurion/lib/worker_queue"
)

type Recipe struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Provider    string       `yaml:"provider"`
	Service     string       `yaml:"service"`
	Steps       []RecipeStep `yaml:"steps"`
}

type RecipeStep struct {
	Name          string            `yaml:"name"`
	ExecutionMode string            `yaml:"execution_mode"`
	EnvVars       map[string]string `yaml:"env_vars,omitempty"`
	BuildFlags    map[string]any    `yaml:"build_flags,omitempty"`
	Timeout       int               `yaml:"timeout,omitempty"`
	Debug         bool              `yaml:"debug,omitempty"`
	Retries       int               `yaml:"retries,omitempty"`
	CommandQueue  worker_queue.WorkQueue
}
