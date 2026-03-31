package orchestrator

import (
	"runtime"
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

func TestPipelineGetWorkerCount(t *testing.T) {
	t.Parallel()

	cpus := runtime.NumCPU()

	t.Run("explicit max workers", func(t *testing.T) {
		t.Parallel()
		services := make([]serviceinfo.ServiceInfo, 10)
		for i := range services {
			services[i] = serviceinfo.ServiceInfo{Name: "svc"}
		}
		pipeline := NewPipeline(services, nil, types.PipelineOptions{MaxWorkers: 8})
		require.Equal(t, 8, pipeline.getWorkerCount())
	})

	t.Run("auto uses one worker per CPU", func(t *testing.T) {
		t.Parallel()
		services := make([]serviceinfo.ServiceInfo, 100)
		for i := range services {
			services[i] = serviceinfo.ServiceInfo{Name: "svc"}
		}
		pipeline := NewPipeline(services, nil, types.PipelineOptions{})
		require.Equal(t, cpus, pipeline.getWorkerCount())
	})

	t.Run("auto caps at service count when fewer than CPUs", func(t *testing.T) {
		t.Parallel()
		services := []serviceinfo.ServiceInfo{{Name: "svc"}}
		pipeline := NewPipeline(services, nil, types.PipelineOptions{})
		require.Equal(t, 1, pipeline.getWorkerCount())
	})
}

func TestPipelineExecuteTaskNilCommand(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0"})
	result := pipeline.executeTask(svc, step)

	// Nil command should return success
	require.True(t, result.Success)
	require.Equal(t, "myservice", result.ServiceName)
	require.Equal(t, "unknown-step", result.StepName)
	require.Nil(t, result.Error)
}

func TestPipelineExecuteTaskWithSimpleCommand(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
	require.Nil(t, result.Error)
}

func TestPipelineExecuteTaskWithStringCommand(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
	require.Nil(t, result.Error)
}

func TestPipelineExecuteTaskWithEnvVars(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskServiceDirMode(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskWithStepTimeout(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskWithStepRetries(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, Retries: 1})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestPipelineExecuteTasksParallel(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:        "v1.0.0",
		Timeout:    10,
		MaxWorkers: 2,
	}

	pipeline := NewPipeline(services, recipes, opts)

	// Create tasks
	tasks := []stepTask{
		{service: services[0], recipe: recipes[0].Recipe, step: &recipes[0].Recipe.Steps[0]},
		{service: services[1], recipe: recipes[0].Recipe, step: &recipes[0].Recipe.Steps[0]},
	}

	err := pipeline.executeTasksParallel(tasks)
	require.NoError(t, err)
}

func TestPipelineExecuteTasksParallelWithFailure(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
	}

	opts := types.PipelineOptions{
		Tag:        "v1.0.0",
		Timeout:    2,
		MaxWorkers: 1,
	}

	pipeline := NewPipeline(services, nil, opts)

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

	err := pipeline.executeTasksParallel(tasks)
	require.Error(t, err)
	require.Contains(t, err.Error(), "step failed for")
}

func TestPipelineExecuteTaskWithBuildFlags(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskEmptyExecutionMode(t *testing.T) {
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

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step)

	require.True(t, result.Success)
}
