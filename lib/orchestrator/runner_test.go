package orchestrator

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "service-a", Provider: "gcp"},
		{Name: "service-b", Provider: "gcp"},
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

	opts := RunnerOptions{
		Tag:        "v1.0.0",
		Debug:      true,
		Timeout:    60,
		MaxWorkers: 2,
	}

	runner := NewRunner(services, recipes, opts)

	require.NotNil(t, runner)
	require.Len(t, runner.services, 2)
	require.NotNil(t, runner.recipes)
	require.NotNil(t, runner.output)
	require.NotNil(t, runner.registry)
	require.Equal(t, opts.Tag, runner.options.Tag)
	require.Equal(t, opts.Debug, runner.options.Debug)
}

func TestNewRunnerEmptyServices(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{})

	require.NotNil(t, runner)
	require.Empty(t, runner.services)
	require.Empty(t, runner.recipes)
}

func TestRunnerFindMaxSteps(t *testing.T) {
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

			opts := RunnerOptions{MaxSteps: tt.maxSteps}
			runner := NewRunner(tt.services, tt.recipes, opts)
			result := runner.findMaxSteps()

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRunnerShouldSkipStep(t *testing.T) {
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

			opts := RunnerOptions{
				OnlyTags:    tt.onlyTags,
				ExcludeTags: tt.excludeTags,
			}
			runner := NewRunner(nil, nil, opts)
			result := runner.shouldSkipStep(tt.step)

			require.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestRunnerStepHasAnyTag(t *testing.T) {
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

			runner := NewRunner(nil, nil, RunnerOptions{})
			result := runner.stepHasAnyTag(tt.step, tt.tags)

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRunnerBuildStepName(t *testing.T) {
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

			runner := NewRunner(nil, nil, RunnerOptions{})
			result := runner.buildStepName(tt.names)

			if len(tt.names) == 1 {
				require.Equal(t, tt.expected, result)
			} else {
				// For multiple names, just verify it contains " / "
				require.Contains(t, result, " / ")
			}
		})
	}
}

func TestRunnerSubstituteVars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "gcp",
		Region:   "us-central1",
		Project:  "my-project",
	}

	opts := RunnerOptions{Tag: "v1.0.0"}
	runner := NewRunner(nil, nil, opts)

	tests := []struct {
		name     string
		cmd      any
		expected any
	}{
		{
			name:     "string with service name",
			cmd:      "echo ${name}",
			expected: "echo myservice",
		},
		{
			name:     "string with service.name",
			cmd:      "echo ${service.name}",
			expected: "echo myservice",
		},
		{
			name:     "string with multiple vars",
			cmd:      "deploy ${name} to ${region} in ${project}",
			expected: "deploy myservice to us-central1 in my-project",
		},
		{
			name:     "string with tag",
			cmd:      "build:${tag}",
			expected: "build:v1.0.0",
		},
		{
			name:     "string with build.version",
			cmd:      "version=${build.version}",
			expected: "version=v1.0.0",
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

			result := runner.substituteVars(tt.cmd, svc)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRunnerGetWorkerCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		maxWorkers int
		services   int
		expected   int
	}{
		{
			name:       "explicit max workers",
			maxWorkers: 8,
			services:   10,
			expected:   8,
		},
		{
			name:       "auto with few services",
			maxWorkers: 0,
			services:   2,
			expected:   2,
		},
		{
			name:       "auto with many services",
			maxWorkers: 0,
			services:   10,
			expected:   4,
		},
		{
			name:       "auto with exactly 4 services",
			maxWorkers: 0,
			services:   4,
			expected:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			services := make([]serviceinfo.ServiceInfo, tt.services)
			for i := range services {
				services[i] = serviceinfo.ServiceInfo{Name: "svc"}
			}

			opts := RunnerOptions{MaxWorkers: tt.maxWorkers}
			runner := NewRunner(services, nil, opts)
			result := runner.getWorkerCount()

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRunnerRunNoServices(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{})
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerRunNoRecipeSteps(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc", Provider: "unknown"},
	}

	runner := NewRunner(services, nil, RunnerOptions{})
	err := runner.Run()

	// Services without matching recipes now cause validation errors
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no matching recipe")
}

func TestRunnerGenerateCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "gcp",
		Region:   "us-central1",
	}

	opts := RunnerOptions{Tag: "v1.0.0"}
	runner := NewRunner(nil, nil, opts)

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
			expected: "echo myservice",
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

			result := runner.generateCommand(svc, tt.step)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskResult(t *testing.T) {
	t.Parallel()

	result := TaskResult{
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

func TestRunnerOptions(t *testing.T) {
	t.Parallel()

	opts := RunnerOptions{
		Tag:          "v1.0.0",
		Registry:     "gcr.io/myproject",
		TemplatePath: "./templates",
		Debug:        true,
		Timeout:      120,
		Retries:      3,
		DryRun:       true,
		MaxWorkers:   4,
		MaxSteps:     2,
		ExcludeTags:  []string{"deploy"},
		OnlyTags:     []string{"build"},
	}

	require.Equal(t, "v1.0.0", opts.Tag)
	require.Equal(t, "gcr.io/myproject", opts.Registry)
	require.Equal(t, "./templates", opts.TemplatePath)
	require.True(t, opts.Debug)
	require.Equal(t, 120, opts.Timeout)
	require.Equal(t, 3, opts.Retries)
	require.True(t, opts.DryRun)
	require.Equal(t, 4, opts.MaxWorkers)
	require.Equal(t, 2, opts.MaxSteps)
	require.Equal(t, []string{"deploy"}, opts.ExcludeTags)
	require.Equal(t, []string{"build"}, opts.OnlyTags)
}

func TestRunnerGenerateCommandWithRegistry(t *testing.T) {
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

	opts := RunnerOptions{Tag: "v1.0.0", TemplatePath: "./_templates"}
	runner := NewRunner([]serviceinfo.ServiceInfo{svc}, recipes, opts)

	// Test with a registered handler (build)
	step := &recepie.RecipeStep{Name: "build"}
	result := runner.generateCommand(svc, step)
	// Build command should return something (may be nil if no build cmd configured)
	// Just verify it doesn't panic
	_ = result
}

func TestRunnerGenerateCommandWithAnySlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "gcp",
		Region:   "us-central1",
	}

	opts := RunnerOptions{Tag: "v1.0.0"}
	runner := NewRunner(nil, nil, opts)

	// Test with []any command
	step := &recepie.RecipeStep{
		Name:    "custom",
		Command: []any{"echo", "${name}", "--tag", "${tag}"},
	}

	result := runner.generateCommand(svc, step)
	expected := []any{"echo", "myservice", "--tag", "v1.0.0"}
	require.Equal(t, expected, result)
}

func TestRunnerDryRun(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "gcp", Region: "us-central1"},
		{Name: "svc2", Provider: "gcp", Region: "us-east1"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "gcp",
			Recipe: recepie.Recipe{
				Name:     "gcp-cloud-run",
				Provider: "gcp",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root", Tags: []string{"build"}},
					{Name: "deploy", ExecutionMode: "root", Tags: []string{"deploy"}},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerDryRunWithExplicitCommands(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "custom",
						Command:       "echo ${name}",
						ExecutionMode: "root",
						Tags:          []string{"build"},
					},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerWithOnlyTags(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root", Tags: []string{"build"}},
					{Name: "deploy", ExecutionMode: "root", Tags: []string{"deploy"}},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:      "v1.0.0",
		DryRun:   true,
		OnlyTags: []string{"build"},
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerWithExcludeTags(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root", Tags: []string{"build"}},
					{Name: "deploy", ExecutionMode: "root", Tags: []string{"deploy"}},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:         "v1.0.0",
		DryRun:      true,
		ExcludeTags: []string{"deploy"},
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerWithMaxSteps(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{Name: "step1", ExecutionMode: "root", Tags: []string{"build"}},
					{Name: "step2", ExecutionMode: "root", Tags: []string{"build"}},
					{Name: "step3", ExecutionMode: "root", Tags: []string{"build"}},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:      "v1.0.0",
		DryRun:   true,
		MaxSteps: 1,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerMultipleProviders(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "gcp-svc", Provider: "gcp"},
		{Name: "aws-svc", Provider: "aws"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "gcp",
			Recipe: recepie.Recipe{
				Name:     "gcp-recipe",
				Provider: "gcp",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root", Tags: []string{"build"}},
				},
			},
		},
		{
			Provider: "aws",
			Recipe: recepie.Recipe{
				Name:     "aws-recipe",
				Provider: "aws",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root", Tags: []string{"build"}},
					{Name: "deploy", ExecutionMode: "root", Tags: []string{"deploy"}},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerServiceWithoutRecipe(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc-with-recipe", Provider: "gcp"},
		{Name: "svc-without-recipe", Provider: "unknown"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "gcp",
			Recipe: recepie.Recipe{
				Name:     "gcp-recipe",
				Provider: "gcp",
				Steps: []recepie.RecipeStep{
					{Name: "build", ExecutionMode: "root", Tags: []string{"build"}},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	// Services without matching recipes now cause validation errors
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no matching recipe")
}

func TestRunnerExecuteTaskNilCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	// Step with no command and no handler registered
	step := &recepie.RecipeStep{
		Name:          "unknown-step",
		ExecutionMode: "root",
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0"})
	result := runner.executeTask(svc, step)

	// Nil command should return success
	require.True(t, result.Success)
	require.Equal(t, "myservice", result.ServiceName)
	require.Equal(t, "unknown-step", result.StepName)
	require.Nil(t, result.Error)
}

func TestRunnerExecuteTaskWithSimpleCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	// Step with a simple command that should succeed
	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "hello"},
		ExecutionMode: "root",
		Timeout:       5,
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
	require.Nil(t, result.Error)
}

func TestRunnerExecuteTaskWithStringCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       "echo hello world",
		ExecutionMode: "root",
		Timeout:       5,
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
	require.Nil(t, result.Error)
}

func TestRunnerExecuteTaskWithEnvVars(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
		BuildConfig: serviceinfo.BuildConfig{
			EnvVars: []serviceinfo.EnvVars{
				{Name: "SVC_VAR", Value: "svc_value"},
			},
		},
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       5,
		EnvVars: map[string]string{
			"STEP_VAR": "step_value",
		},
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestRunnerExecuteTaskServiceDirMode(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
		Path:     ".",
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "service_dir",
		Timeout:       5,
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestRunnerExecuteTaskWithStepTimeout(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       30, // Step-specific timeout
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestRunnerExecuteTaskWithStepRetries(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       5,
		Retries:       2, // Step-specific retries
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10, Retries: 1})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestRunnerExecuteTasksParallel(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
		{Name: "svc2", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "echo",
						Command:       []string{"echo", "hello"},
						ExecutionMode: "root",
						Tags:          []string{"build"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:        "v1.0.0",
		Timeout:    10,
		MaxWorkers: 2,
	}

	runner := NewRunner(services, recipes, opts)

	// Create tasks
	tasks := []stepTask{
		{service: services[0], recipe: recipes[0].Recipe, step: &recipes[0].Recipe.Steps[0]},
		{service: services[1], recipe: recipes[0].Recipe, step: &recipes[0].Recipe.Steps[0]},
	}

	err := runner.executeTasksParallel(tasks)
	require.NoError(t, err)
}

func TestRunnerExecuteTasksParallelWithFailure(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	opts := RunnerOptions{
		Tag:        "v1.0.0",
		Timeout:    2,
		MaxWorkers: 1,
	}

	runner := NewRunner(services, nil, opts)

	// Create a task with a command that will fail
	step := &recepie.RecipeStep{
		Name:          "fail",
		Command:       []string{"false"}, // This command always fails
		ExecutionMode: "root",
		Timeout:       2,
	}

	tasks := []stepTask{
		{service: services[0], step: step},
	}

	err := runner.executeTasksParallel(tasks)
	require.Error(t, err)
	require.Contains(t, err.Error(), "step failed for")
}

func TestRunnerFullRunWithExecution(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "step1",
						Command:       []string{"echo", "step1"},
						ExecutionMode: "root",
						Tags:          []string{"build"},
						Timeout:       5,
					},
					{
						Name:          "step2",
						Command:       []string{"echo", "step2"},
						ExecutionMode: "root",
						Tags:          []string{"build"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:        "v1.0.0",
		Timeout:    10,
		MaxWorkers: 1,
		DryRun:     false,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerFullRunMultipleServicesParallel(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
		{Name: "svc2", Provider: "test"},
		{Name: "svc3", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "build",
						Command:       []string{"echo", "building ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"build"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{
		Tag:        "v1.0.0",
		Timeout:    10,
		MaxWorkers: 3, // Run all in parallel
		DryRun:     false,
	}

	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestRunnerExecuteTaskWithBuildFlags(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       5,
		BuildFlags: map[string]any{
			"verbose": true,
		},
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
}

// --- Wave-based deployment ordering tests ---

func TestStepRequiresWaves_DeployTag(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{})
	tasks := []stepTask{
		{
			service: serviceinfo.ServiceInfo{Name: "db", DependsOn: nil},
			step:    &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
		},
		{
			service: serviceinfo.ServiceInfo{Name: "api", DependsOn: []string{"db"}},
			step:    &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
		},
	}

	require.True(t, runner.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_ExecuteTag(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{})
	tasks := []stepTask{
		{
			service: serviceinfo.ServiceInfo{Name: "db", DependsOn: nil},
			step:    &recepie.RecipeStep{Name: "execute job", Tags: []string{"execute"}},
		},
		{
			service: serviceinfo.ServiceInfo{Name: "api", DependsOn: []string{"db"}},
			step:    &recepie.RecipeStep{Name: "execute job", Tags: []string{"execute"}},
		},
	}

	require.True(t, runner.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_BuildTag(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{})
	tasks := []stepTask{
		{
			service: serviceinfo.ServiceInfo{Name: "db", DependsOn: nil},
			step:    &recepie.RecipeStep{Name: "build", Tags: []string{"build"}},
		},
		{
			service: serviceinfo.ServiceInfo{Name: "api", DependsOn: []string{"db"}},
			step:    &recepie.RecipeStep{Name: "build", Tags: []string{"build"}},
		},
	}

	// Build steps should NOT use waves even with deps
	require.False(t, runner.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_NoDepsFlag(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{NoDeps: true})
	tasks := []stepTask{
		{
			service: serviceinfo.ServiceInfo{Name: "db", DependsOn: nil},
			step:    &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
		},
		{
			service: serviceinfo.ServiceInfo{Name: "api", DependsOn: []string{"db"}},
			step:    &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
		},
	}

	require.False(t, runner.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_NoDependencies(t *testing.T) {
	t.Parallel()

	runner := NewRunner(nil, nil, RunnerOptions{})
	tasks := []stepTask{
		{
			service: serviceinfo.ServiceInfo{Name: "svc1"},
			step:    &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
		},
		{
			service: serviceinfo.ServiceInfo{Name: "svc2"},
			step:    &recepie.RecipeStep{Name: "deploy", Tags: []string{"deploy"}},
		},
	}

	require.False(t, runner.stepRequiresWaves(tasks))
}

func TestWaveExecution_LinearChain(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "test"},
		{Name: "api", Provider: "test", DependsOn: []string{"db"}},
		{Name: "gateway", Provider: "test", DependsOn: []string{"api"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
	require.Len(t, runner.results, 3)
}

func TestWaveExecution_Diamond(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "base", Provider: "test"},
		{Name: "auth", Provider: "test", DependsOn: []string{"base"}},
		{Name: "users", Provider: "test", DependsOn: []string{"base"}},
		{Name: "gateway", Provider: "test", DependsOn: []string{"auth", "users"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 4}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
	require.Len(t, runner.results, 4)
	for _, r := range runner.results {
		require.True(t, r.Success, "service %s should succeed", r.ServiceName)
	}
}

func TestWaveExecution_NoDependencies(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
		{Name: "svc2", Provider: "test"},
		{Name: "svc3", Provider: "test"},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
	// All should succeed in single wave (flat parallel)
	require.Len(t, runner.results, 3)
}

func TestWaveExecution_FailurePropagation(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "test"},
		{Name: "api", Provider: "test", DependsOn: []string{"db"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"false"}, // always fails
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.Error(t, err)
	// db should fail, api should be skipped due to failed dependency
	require.Len(t, runner.results, 2)

	dbResult := findResult(runner.results, "db")
	require.NotNil(t, dbResult)
	require.False(t, dbResult.Success)

	apiResult := findResult(runner.results, "api")
	require.NotNil(t, apiResult)
	require.False(t, apiResult.Success)
	require.Contains(t, apiResult.Error.Error(), "dependency db failed")
}

func TestWaveExecution_PartialWaveFailure(t *testing.T) {
	t.Parallel()

	// db fails, but independent-svc (no deps) should still succeed
	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "fail-test"},
		{Name: "independent", Provider: "test"},
		{Name: "api", Provider: "test", DependsOn: []string{"db"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "fail-test",
			Recipe: recepie.Recipe{
				Name:     "fail-recipe",
				Provider: "fail-test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"false"}, // fails
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.Error(t, err) // Overall error because db failed

	independentResult := findResult(runner.results, "independent")
	require.NotNil(t, independentResult)
	require.True(t, independentResult.Success) // independent should succeed

	apiResult := findResult(runner.results, "api")
	require.NotNil(t, apiResult)
	require.False(t, apiResult.Success) // api skipped due to db failure
}

func TestWaveExecution_BuildStepIgnoresDeps(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "test"},
		{Name: "api", Provider: "test", DependsOn: []string{"db"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "build",
						Command:       []string{"echo", "building ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"build"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
	// Both should complete (build steps run in flat parallel, no waves)
	require.Len(t, runner.results, 2)
	for _, r := range runner.results {
		require.True(t, r.Success)
	}
}

func TestWaveExecution_DryRunWithWaves(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "test"},
		{Name: "api", Provider: "test", DependsOn: []string{"db"}},
		{Name: "gateway", Provider: "test", DependsOn: []string{"api"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", DryRun: true}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
}

func TestWaveExecution_NoDepsFlag(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "test"},
		{Name: "api", Provider: "test", DependsOn: []string{"db"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	// With NoDeps, all services run in flat parallel regardless of deps
	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2, NoDeps: true}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
	require.Len(t, runner.results, 2)
	for _, r := range runner.results {
		require.True(t, r.Success)
	}
}

func TestWaveExecution_MixedProviders(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "db", Provider: "test"},
		{Name: "gcp-api", Provider: "gcp-test", DependsOn: []string{"db"}},
		{Name: "aws-api", Provider: "aws-test", DependsOn: []string{"db"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name}"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
		{
			Provider: "gcp-test",
			Recipe: recepie.Recipe{
				Name:     "gcp-test-recipe",
				Provider: "gcp-test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name} to gcp"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
		{
			Provider: "aws-test",
			Recipe: recepie.Recipe{
				Name:     "aws-test-recipe",
				Provider: "aws-test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"echo", "deploying ${name} to aws"},
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.NoError(t, err)
	require.Len(t, runner.results, 3)
	for _, r := range runner.results {
		require.True(t, r.Success)
	}
}

func TestWaveExecution_TransitiveFailure(t *testing.T) {
	t.Parallel()

	// A fails → B skipped → C skipped
	services := []serviceinfo.ServiceInfo{
		{Name: "a", Provider: "test"},
		{Name: "b", Provider: "test", DependsOn: []string{"a"}},
		{Name: "c", Provider: "test", DependsOn: []string{"b"}},
	}

	recipes := []recepie.RecipeInfo{
		{
			Provider: "test",
			Recipe: recepie.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recepie.RecipeStep{
					{
						Name:          "deploy",
						Command:       []string{"false"}, // always fails
						ExecutionMode: "root",
						Tags:          []string{"deploy"},
						Timeout:       5,
					},
				},
			},
		},
	}

	opts := RunnerOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 1}
	runner := NewRunner(services, recipes, opts)
	err := runner.Run()

	require.Error(t, err)
	require.Len(t, runner.results, 3)

	aResult := findResult(runner.results, "a")
	require.NotNil(t, aResult)
	require.False(t, aResult.Success)

	bResult := findResult(runner.results, "b")
	require.NotNil(t, bResult)
	require.False(t, bResult.Success)
	require.Contains(t, bResult.Error.Error(), "dependency a failed")

	cResult := findResult(runner.results, "c")
	require.NotNil(t, cResult)
	require.False(t, cResult.Success)
	require.Contains(t, cResult.Error.Error(), "dependency b failed")
}

// findResult finds a result by service name prefix in the results slice.
func findResult(results []TaskResult, namePrefix string) *TaskResult {
	for _, r := range results {
		if r.ServiceName == namePrefix {
			return &r
		}
	}
	return nil
}

// Regression: when a handler returns a typed nil (e.g., nil []string),
// Go wraps it in a non-nil any interface. generateCommand must detect this
// and return an untyped nil so the caller can properly skip the step.
// This was the root cause of "execute job" showing "failed:" with no message.
func TestRunnerGenerateCommandHandlesTypedNilSlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0"})

	// Register a handler that returns a typed nil []string (simulates
	// GenerateExecuteJobCommand when execute_on_deploy is false)
	runner.registry.Register("typed-nil-step", "test-nil", func(_ registry.StepContext) any {
		var cmd []string // nil []string
		return cmd       // Returns typed nil wrapped in any — NOT == nil
	})

	step := &recepie.RecipeStep{Name: "typed-nil-step"}
	result := runner.generateCommand(svc, step)

	// Must be actual nil, not a typed nil wrapped in any
	require.Nil(t, result)
}

// Regression: same as above but for nil []any slices.
func TestRunnerGenerateCommandHandlesTypedNilAnySlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0"})

	runner.registry.Register("typed-nil-any-step", "test-nil", func(_ registry.StepContext) any {
		var cmd []any // nil []any
		return cmd
	})

	step := &recepie.RecipeStep{Name: "typed-nil-any-step"}
	result := runner.generateCommand(svc, step)

	require.Nil(t, result)
}

// Regression: a handler returning untyped nil should still work.
func TestRunnerGenerateCommandHandlesUntypedNil(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0"})

	runner.registry.Register("untyped-nil-step", "test-nil", func(_ registry.StepContext) any {
		return nil
	})

	step := &recepie.RecipeStep{Name: "untyped-nil-step"}
	result := runner.generateCommand(svc, step)

	require.Nil(t, result)
}

// Regression: a handler returning a non-nil []string should pass through normally.
func TestRunnerGenerateCommandPassesThroughNonNilSlice(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0"})

	runner.registry.Register("real-command-step", "test-nil", func(_ registry.StepContext) any {
		return []string{"echo", "hello"}
	})

	step := &recepie.RecipeStep{Name: "real-command-step"}
	result := runner.generateCommand(svc, step)

	require.NotNil(t, result)
	require.Equal(t, []string{"echo", "hello"}, result)
}

// Regression: executeTask should treat typed nil commands as success (skip).
// Before the fix, a typed nil []string from a handler would pass the nil check,
// reach CommandWorker with a nil slice, and fail silently with (false, nil).
func TestRunnerExecuteTaskTypedNilCommandIsSuccess(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test-nil",
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})

	runner.registry.Register("skip-step", "test-nil", func(_ registry.StepContext) any {
		var cmd []string // typed nil — simulates execute_on_deploy=false
		return cmd
	})

	step := &recepie.RecipeStep{
		Name:          "skip-step",
		ExecutionMode: "root",
		Timeout:       5,
	}

	result := runner.executeTask(svc, step)

	require.True(t, result.Success, "typed nil command should be treated as skip (success)")
	require.Nil(t, result.Error)
}

func TestRunnerExecuteTaskEmptyExecutionMode(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recepie.RecipeStep{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "", // Empty should default to "root"
		Timeout:       5,
	}

	runner := NewRunner(nil, nil, RunnerOptions{Tag: "v1.0.0", Timeout: 10})
	result := runner.executeTask(svc, step)

	require.True(t, result.Success)
}
