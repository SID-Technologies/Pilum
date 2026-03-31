package orchestrator

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

func TestPipelineFindMaxSteps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		services []serviceinfo.ServiceInfo
		recipes  []recepie.RecipeInfo
		maxSteps int
		expected int
	}{
		{
			name: "single recipe with 3 steps",
			services: []serviceinfo.ServiceInfo{
				{Name: "svc", Provider: "gcp"},
			},
			recipes: []recepie.RecipeInfo{
				{
					Provider: "gcp",
					Recipe: recepie.Recipe{
						Steps: []recepie.RecipeStep{
							{Name: "build"},
							{Name: "push"},
							{Name: "deploy"},
						},
					},
				},
			},
			maxSteps: 0,
			expected: 3,
		},
		{
			name: "multiple recipes with different step counts",
			services: []serviceinfo.ServiceInfo{
				{Name: "svc1", Provider: "gcp"},
				{Name: "svc2", Provider: "aws"},
			},
			recipes: []recepie.RecipeInfo{
				{
					Provider: "gcp",
					Recipe: recepie.Recipe{
						Steps: []recepie.RecipeStep{{Name: "build"}, {Name: "push"}},
					},
				},
				{
					Provider: "aws",
					Recipe: recepie.Recipe{
						Steps: []recepie.RecipeStep{{Name: "build"}, {Name: "push"}, {Name: "deploy"}, {Name: "verify"}},
					},
				},
			},
			maxSteps: 0,
			expected: 4,
		},
		{
			name: "limited by MaxSteps option",
			services: []serviceinfo.ServiceInfo{
				{Name: "svc", Provider: "gcp"},
			},
			recipes: []recepie.RecipeInfo{
				{
					Provider: "gcp",
					Recipe: recepie.Recipe{
						Steps: []recepie.RecipeStep{
							{Name: "build"},
							{Name: "push"},
							{Name: "deploy"},
						},
					},
				},
			},
			maxSteps: 2,
			expected: 2,
		},
		{
			name: "service without matching recipe",
			services: []serviceinfo.ServiceInfo{
				{Name: "svc", Provider: "unknown"},
			},
			recipes:  []recepie.RecipeInfo{},
			maxSteps: 0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.PipelineOptions{MaxSteps: tt.maxSteps}
			pipeline := NewPipeline(tt.services, tt.recipes, opts)
			result := pipeline.findMaxSteps()

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineShouldSkipStep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		step        *recepie.RecipeStep
		onlyTags    []string
		excludeTags []string
		shouldSkip  bool
	}{
		{
			name:        "no filters - don't skip",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{"build"}},
			onlyTags:    nil,
			excludeTags: nil,
			shouldSkip:  false,
		},
		{
			name:        "only tags - step has matching tag",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{"build"}},
			onlyTags:    []string{"build"},
			excludeTags: nil,
			shouldSkip:  false,
		},
		{
			name:        "only tags - step does not have matching tag",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{"build"}},
			onlyTags:    []string{"deploy"},
			excludeTags: nil,
			shouldSkip:  true,
		},
		{
			name:        "exclude tags - step has excluded tag",
			step:        &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
			onlyTags:    nil,
			excludeTags: []string{"deploy"},
			shouldSkip:  true,
		},
		{
			name:        "exclude tags - step does not have excluded tag",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{"build"}},
			onlyTags:    nil,
			excludeTags: []string{"deploy"},
			shouldSkip:  false,
		},
		{
			name:        "both filters - step passes both",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{"build", "ci"}},
			onlyTags:    []string{"build"},
			excludeTags: []string{"deploy"},
			shouldSkip:  false,
		},
		{
			name:        "case insensitive matching",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{"BUILD"}},
			onlyTags:    []string{"build"},
			excludeTags: nil,
			shouldSkip:  false,
		},
		{
			name:        "step with no tags - only filter",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{}},
			onlyTags:    []string{"build"},
			excludeTags: nil,
			shouldSkip:  true,
		},
		{
			name:        "step with no tags - exclude filter",
			step:        &recepie.RecipeStep{Name: "build", Tags: []string{}},
			onlyTags:    nil,
			excludeTags: []string{"deploy"},
			shouldSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.PipelineOptions{
				OnlyTags:    tt.onlyTags,
				ExcludeTags: tt.excludeTags,
			}
			pipeline := NewPipeline(nil, nil, opts)
			result := pipeline.shouldSkipStep(tt.step)

			require.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestPipelineStepHasAnyTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		step     *recepie.RecipeStep
		tags     []string
		expected bool
	}{
		{
			name:     "step has matching tag",
			step:     &recepie.RecipeStep{Tags: []string{"build", "ci"}},
			tags:     []string{"build"},
			expected: true,
		},
		{
			name:     "step has multiple matching tags",
			step:     &recepie.RecipeStep{Tags: []string{"build", "ci"}},
			tags:     []string{"build", "ci"},
			expected: true,
		},
		{
			name:     "step has no matching tags",
			step:     &recepie.RecipeStep{Tags: []string{"build"}},
			tags:     []string{"deploy"},
			expected: false,
		},
		{
			name:     "step has no tags",
			step:     &recepie.RecipeStep{Tags: []string{}},
			tags:     []string{"build"},
			expected: false,
		},
		{
			name:     "empty tags to check",
			step:     &recepie.RecipeStep{Tags: []string{"build"}},
			tags:     []string{},
			expected: false,
		},
		{
			name:     "case insensitive match",
			step:     &recepie.RecipeStep{Tags: []string{"BUILD"}},
			tags:     []string{"build"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
			result := pipeline.stepHasAnyTag(tt.step, tt.tags)

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineBuildStepName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		names    map[string]bool
		expected string
	}{
		{
			name:     "single step name",
			names:    map[string]bool{"build": true},
			expected: "build",
		},
		{
			name:     "multiple step names",
			names:    map[string]bool{"build": true, "test": true},
			expected: "", // Can't predict exact order due to map iteration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
			result := pipeline.buildStepName(tt.names)

			if len(tt.names) == 1 {
				require.Equal(t, tt.expected, result)
			} else {
				// For multiple names, just verify it contains " / "
				require.Contains(t, result, " / ")
			}
		})
	}
}

func TestPipelineSubstituteVars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "gcp",
		Region:   "us-central1",
		Project:  "my-project",
	}

	opts := types.PipelineOptions{Tag: "v1.0.0"}
	pipeline := NewPipeline(nil, nil, opts)

	tests := []struct {
		name     string
		cmd      any
		expected any
	}{
		{
			name:     "string with service name",
			cmd:      "echo ${name}",
			expected: "echo 'myservice'",
		},
		{
			name:     "string with service.name",
			cmd:      "echo ${service.name}",
			expected: "echo 'myservice'",
		},
		{
			name:     "string with multiple vars",
			cmd:      "deploy ${name} to ${region} in ${project}",
			expected: "deploy 'myservice' to 'us-central1' in 'my-project'",
		},
		{
			name:     "string with tag",
			cmd:      "build:${tag}",
			expected: "build:'v1.0.0'",
		},
		{
			name:     "string with build.version",
			cmd:      "version=${build.version}",
			expected: "version='v1.0.0'",
		},
		{
			name:     "string slice",
			cmd:      []string{"deploy", "${name}", "--region", "${region}"},
			expected: []string{"deploy", "myservice", "--region", "us-central1"},
		},
		{
			name:     "any slice",
			cmd:      []any{"deploy", "${name}", "--tag", "${tag}"},
			expected: []any{"deploy", "myservice", "--tag", "v1.0.0"},
		},
		{
			name:     "any slice with non-string",
			cmd:      []any{"timeout", 30, "${name}"},
			expected: []any{"timeout", 30, "myservice"},
		},
		{
			name:     "no vars to substitute",
			cmd:      "echo hello",
			expected: "echo hello",
		},
		{
			name:     "int passes through",
			cmd:      42,
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := pipeline.substituteVars(tt.cmd, svc)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPipelineGenerateCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "gcp",
		Region:   "us-central1",
	}

	opts := types.PipelineOptions{Tag: "v1.0.0"}
	pipeline := NewPipeline(nil, nil, opts)

	tests := []struct {
		name     string
		step     *recepie.RecipeStep
		expected any
	}{
		{
			name: "step with explicit string command",
			step: &recepie.RecipeStep{
				Name:    "custom",
				Command: "echo ${name}",
			},
			expected: "echo 'myservice'",
		},
		{
			name: "step with explicit array command",
			step: &recepie.RecipeStep{
				Name:    "custom",
				Command: []string{"echo", "${name}"},
			},
			expected: []string{"echo", "myservice"},
		},
		{
			name: "step without command uses registry",
			step: &recepie.RecipeStep{
				Name: "unknown-step",
			},
			expected: nil, // No handler registered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := pipeline.generateCommand(svc, tt.step)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskResult(t *testing.T) {
	t.Parallel()

	result := types.TaskResult{
		ServiceName: "myservice",
		StepName:    "build",
		Success:     true,
		Error:       nil,
	}

	require.Equal(t, "myservice", result.ServiceName)
	require.Equal(t, "build", result.StepName)
	require.True(t, result.Success)
	require.Nil(t, result.Error)
}

func TestPipelineGenerateCommandWithRegistry(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:         "myservice",
		Provider:     "gcp",
		Region:       "us-central1",
		RegistryName: "gcr.io/project",
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "gcp",
			Recipe: recepie.Recipe{
				Name:     "gcp-cloud-run",
				Provider: "gcp",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root"},
				},
			},
		},
	}

	opts := types.PipelineOptions{Tag: "v1.0.0", TemplatePath: "./_templates"}
	pipeline := NewPipeline([]serviceinfo.ServiceInfo{svc}, recipes, opts)

	// Test with a registered handler (build)
	step := &recepie.RecipeStep{Name: "build"}
	result := pipeline.generateCommand(svc, step)
	// Build command should return something (may be nil if no build cmd configured)
	// Just verify it doesn't panic
	_ = result
}

func TestPipelineGenerateCommandWithAnySlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "gcp",
		Region:   "us-central1",
	}

	opts := types.PipelineOptions{Tag: "v1.0.0"}
	pipeline := NewPipeline(nil, nil, opts)

	// Test with []any command
	step := &recepie.RecipeStep{
		Name:    "custom",
		Command: []any{"echo", "${name}", "--tag", "${tag}"},
	}

	result := pipeline.generateCommand(svc, step)
	expected := []any{"echo", "myservice", "--tag", "v1.0.0"}
	require.Equal(t, expected, result)
}

// Regression: when a handler returns a typed nil (e.g., nil []string),
// Go wraps it in a non-nil any interface. generateCommand must detect this
// and return an untyped nil so the caller can properly skip the step.
// This was the root cause of "execute job" showing "failed:" with no message.
func TestPipelineGenerateCommandHandlesTypedNilSlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0"})

	// Register a handler that returns a typed nil []string (simulates
	// GenerateExecuteJobCommand when execute_on_deploy is false)
	pipeline.registry.Register("typed-nil-step", "test-nil", func(_ registry.StepContext) any {
		var cmd []string // nil []string
		return cmd       // Returns typed nil wrapped in any — NOT == nil
	})

	step := &recepie.RecipeStep{Name: "typed-nil-step"}
	result := pipeline.generateCommand(svc, step)

	// Must be actual nil, not a typed nil wrapped in any
	require.Nil(t, result)
}

// Regression: same as above but for nil []any slices.
func TestPipelineGenerateCommandHandlesTypedNilAnySlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0"})

	pipeline.registry.Register("typed-nil-any-step", "test-nil", func(_ registry.StepContext) any {
		var cmd []any // nil []any
		return cmd
	})

	step := &recepie.RecipeStep{Name: "typed-nil-any-step"}
	result := pipeline.generateCommand(svc, step)

	require.Nil(t, result)
}

// Regression: a handler returning untyped nil should still work.
func TestPipelineGenerateCommandHandlesUntypedNil(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0"})

	pipeline.registry.Register("untyped-nil-step", "test-nil", func(_ registry.StepContext) any {
		return nil
	})

	step := &recepie.RecipeStep{Name: "untyped-nil-step"}
	result := pipeline.generateCommand(svc, step)

	require.Nil(t, result)
}

// Regression: a handler returning a non-nil []string should pass through normally.
func TestPipelineGenerateCommandPassesThroughNonNilSlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0"})

	pipeline.registry.Register("real-command-step", "test-nil", func(_ registry.StepContext) any {
		return []string{"echo", "hello"}
	})

	step := &recepie.RecipeStep{Name: "real-command-step"}
	result := pipeline.generateCommand(svc, step)

	require.NotNil(t, result)
	require.Equal(t, []string{"echo", "hello"}, result)
}

// Regression: executeTask should treat typed nil commands as success (skip).
// Before the fix, a typed nil []string from a handler would pass the nil check,
// reach CommandWorker with a nil slice, and fail silently with (false, nil).
func TestPipelineExecuteTaskTypedNilCommandIsSuccess(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})

	pipeline.registry.Register("skip-step", "test-nil", func(_ registry.StepContext) any {
		var cmd []string // typed nil — simulates execute_on_deploy=false
		return cmd
	})

	step := &recepie.RecipeStep{
		Name:          "skip-step",
		ExecutionMode: "root",
		Timeout:       5,
	}

	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success, "typed nil command should be treated as skip (success)")
	require.Nil(t, result.Error)
}
