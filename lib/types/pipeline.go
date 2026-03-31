//nolint:revive // types is a common package name for shared type definitions
package types

import "time"

// TaskResult holds the result of a service task execution.
type TaskResult struct {
	ServiceName string
	StepName    string
	Success     bool
	Duration    time.Duration
	Error       error
}

// DryRunEntry represents a single dry-run step for JSON output.
type DryRunEntry struct {
	Service string `json:"service"`
	Step    string `json:"step"`
	Command string `json:"command"`
	Wave    int    `json:"wave,omitempty"`
}

// PipelineOptions configures the pipeline.
type PipelineOptions struct {
	Tag          string
	Registry     string // Docker registry prefix (overrides pilum.yaml)
	TemplatePath string // Default template path for services that don't specify one
	Debug        bool
	Timeout      int
	Retries      int
	DryRun       bool
	MaxWorkers   int
	MaxSteps     int      // Maximum number of steps to run (0 = all)
	ExcludeTags  []string // Exclude steps with these tags (e.g., "deploy")
	OnlyTags     []string // Only run steps with these tags (e.g., "deploy")
	NoDeps       bool     // Disable wave-based deployment ordering
}
