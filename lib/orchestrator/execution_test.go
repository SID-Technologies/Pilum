package orchestrator

import (
	"os"
	"runtime"
	"testing"

	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/sysinfo"
	"github.com/sid-technologies/pilum/lib/templates"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

// planTasks builds n tasks of one step for services sharing a build config.
func planTasks(n int, step *recipe.Step, bc serviceinfo.BuildConfig) (*Pipeline, []stepTask) {
	services := make([]serviceinfo.ServiceInfo, n)
	tasks := make([]stepTask, n)
	for i := range services {
		services[i] = serviceinfo.ServiceInfo{Name: "svc", Provider: "test", BuildConfig: bc}
		tasks[i] = stepTask{service: services[i], step: step}
	}
	return NewPipeline(services, nil, types.PipelineOptions{}), tasks
}

func TestPlanWorkers(t *testing.T) {
	t.Parallel()

	cpus := runtime.NumCPU()
	buildStep := &recipe.Step{Name: "build binary", Command: "true", Tags: []string{"build"}}
	deployStep := &recipe.Step{Name: "deploy", Command: "true", Tags: []string{"deploy"}}
	goBuild := serviceinfo.BuildConfig{Language: "go"}

	t.Run("explicit max workers on build step splits cores", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(10, buildStep, goBuild)
		p.options.MaxWorkers = 8
		plan := p.planWorkers(tasks)
		require.Equal(t, 8, plan.workers)
		require.Equal(t, max(1, cpus/8), plan.threads)
	})

	t.Run("explicit max workers on deploy step leaves GOMAXPROCS alone", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(10, deployStep, goBuild)
		p.options.MaxWorkers = 8
		require.Equal(t, workerPlan{workers: 8}, p.planWorkers(tasks))
	})

	t.Run("network-bound steps get the larger IO limit", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(100, deployStep, goBuild)
		require.Equal(t, workerPlan{workers: maxIOWorkers}, p.planWorkers(tasks))

		p, tasks = planTasks(3, &recipe.Step{Name: "publish", Tags: []string{"push"}}, goBuild)
		require.Equal(t, workerPlan{workers: 3}, p.planWorkers(tasks))
	})

	t.Run("builds are charged default CPUs and never oversubscribe", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(100, buildStep, goBuild)
		plan := p.planWorkers(tasks)
		require.GreaterOrEqual(t, plan.workers, 1)
		require.LessOrEqual(t, plan.workers, max(1, cpus/defaultBuildCPUs))
		require.LessOrEqual(t, plan.workers*plan.threads, max(cpus, plan.threads))
	})

	t.Run("resources.cpu raises the per-build cost", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(10, buildStep, serviceinfo.BuildConfig{
			Language:  "go",
			Resources: serviceinfo.BuildResources{CPU: cpus},
		})
		require.Equal(t, workerPlan{workers: 1, threads: cpus}, p.planWorkers(tasks))
	})

	t.Run("single build gets every core", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(1, buildStep, goBuild)
		require.Equal(t, workerPlan{workers: 1, threads: cpus}, p.planWorkers(tasks))
	})

	t.Run("untagged steps are treated as CPU-bound", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(1, &recipe.Step{Name: "custom", Command: "true"}, goBuild)
		require.Positive(t, p.planWorkers(tasks).threads)
	})

	t.Run("docker steps are sized from the daemon", func(t *testing.T) {
		t.Parallel()
		dockerStep := &recipe.Step{Name: "image", Command: []string{"docker", "build", "."}, Tags: []string{"build"}}
		p, tasks := planTasks(100, dockerStep, goBuild)
		p.dockerResources = sysinfo.DockerResources{CPUs: 4, MemoryMB: 100000}
		require.Equal(t, workerPlan{workers: 2, threads: 2}, p.planWorkers(tasks))
	})

	t.Run("memory limit still applies", func(t *testing.T) {
		t.Parallel()
		p, tasks := planTasks(10, buildStep, serviceinfo.BuildConfig{
			Language:  "java",
			Resources: serviceinfo.BuildResources{Memory: 999999}, // absurdly high
		})
		require.Equal(t, 1, p.planWorkers(tasks).workers)
	})
}

func TestExecuteTaskSetsGOMAXPROCS(t *testing.T) {
	t.Parallel()

	if os.Getenv("GOMAXPROCS") != "" {
		t.Skip("GOMAXPROCS set in the test environment takes precedence")
	}

	p := NewPipeline(nil, nil, types.PipelineOptions{Timeout: 5})
	svc := serviceinfo.ServiceInfo{Name: "svc", Provider: "test"}

	step := &recipe.Step{Name: "check", Command: `[ "$GOMAXPROCS" = "3" ]`}
	require.True(t, p.executeTask(svc, step, 3).Success)

	// An explicit step env var wins over the computed value.
	step = &recipe.Step{Name: "check", Command: `[ "$GOMAXPROCS" = "7" ]`, EnvVars: map[string]string{"GOMAXPROCS": "7"}}
	require.True(t, p.executeTask(svc, step, 3).Success)

	// threads == 0 leaves it unset.
	step = &recipe.Step{Name: "check", Command: `[ -z "$GOMAXPROCS" ]`}
	require.True(t, p.executeTask(svc, step, 0).Success)
}

func TestPipelineMaxBuildMemoryMB(t *testing.T) {
	t.Parallel()

	t.Run("uses language defaults", func(t *testing.T) {
		t.Parallel()
		services := []serviceinfo.ServiceInfo{
			{Name: "go-svc", BuildConfig: serviceinfo.BuildConfig{Language: "go"}},
			{Name: "rust-svc", BuildConfig: serviceinfo.BuildConfig{Language: "rust"}},
		}
		pipeline := NewPipeline(services, nil, types.PipelineOptions{})
		// Rust (2048) > Go (1024), so max should be 2048
		require.Equal(t, 2048, pipeline.maxBuildMemoryMB())
	})

	t.Run("explicit resources override language defaults", func(t *testing.T) {
		t.Parallel()
		services := []serviceinfo.ServiceInfo{
			{Name: "svc", BuildConfig: serviceinfo.BuildConfig{
				Language:  "go",
				Resources: serviceinfo.BuildResources{Memory: 4096},
			}},
		}
		pipeline := NewPipeline(services, nil, types.PipelineOptions{})
		require.Equal(t, 4096, pipeline.maxBuildMemoryMB())
	})

	t.Run("unknown language uses default", func(t *testing.T) {
		t.Parallel()
		services := []serviceinfo.ServiceInfo{
			{Name: "svc", BuildConfig: serviceinfo.BuildConfig{Language: ""}},
		}
		pipeline := NewPipeline(services, nil, types.PipelineOptions{})
		require.Equal(t, templates.DefaultBuildMemoryMB, pipeline.maxBuildMemoryMB())
	})
}

func TestPipelineExecuteTaskNilCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	// Step with no command and no handler registered
	step := &recipe.Step{
		Name:          "unknown-step",
		ExecutionMode: "root",
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0"})
	result := pipeline.executeTask(svc, step, 0)

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
	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "hello"},
		ExecutionMode: "root",
		Timeout:       5,
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
	require.Nil(t, result.Error)
}

func TestPipelineExecuteTaskWithStringCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recipe.Step{
		Name:          "echo",
		Command:       "echo hello world",
		ExecutionMode: "root",
		Timeout:       5,
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

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

	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       5,
		EnvVars: map[string]string{
			"STEP_VAR": "step_value",
		},
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskServiceDirMode(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
		Path:     ".",
	}

	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "service_dir",
		Timeout:       5,
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskWithStepTimeout(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       30, // Step-specific timeout
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskWithStepRetries(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       5,
		Retries:       2, // Step-specific retries
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, Retries: 1})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
}

func TestPipelineExecuteTasksParallel(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc1", Provider: "test"},
		{Name: "svc2", Provider: "test"},
	}

	recipes := []recipe.Info{
		{
			Provider: "test",
			Recipe: recipe.Recipe{
				Name:     "test-recipe",
				Provider: "test",
				Steps: []recipe.Step{
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
	step := &recipe.Step{
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

	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "root",
		Timeout:       5,
		BuildFlags: map[string]any{
			"verbose": true,
		},
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
}

func TestPipelineExecuteTaskEmptyExecutionMode(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "myservice",
		Provider: "test",
	}

	step := &recipe.Step{
		Name:          "echo",
		Command:       []string{"echo", "test"},
		ExecutionMode: "", // Empty should default to "root"
		Timeout:       5,
	}

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{Tag: "v1.0.0", Timeout: 10})
	result := pipeline.executeTask(svc, step, 0)

	require.True(t, result.Success)
}
