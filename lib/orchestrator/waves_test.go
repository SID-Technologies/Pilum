package orchestrator

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

// --- Wave-based deployment ordering tests ---

func TestStepRequiresWaves_DeployTag(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
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

	require.True(t, pipeline.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_ExecuteTag(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
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

	require.True(t, pipeline.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_BuildTag(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
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

	// Build steps should use waves when services have deps (e.g. npm workspace packages)
	require.True(t, pipeline.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_NoDepsFlag(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{NoDeps: true})
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

	require.False(t, pipeline.stepRequiresWaves(tasks))
}

func TestStepRequiresWaves_NoDependencies(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
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

	require.False(t, pipeline.stepRequiresWaves(tasks))
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	require.Len(t, pipeline.results, 3)
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 4}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	require.Len(t, pipeline.results, 4)
	for _, r := range pipeline.results {
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	// All should succeed in single wave (flat parallel)
	require.Len(t, pipeline.results, 3)
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.Error(t, err)
	// db should fail, api should be skipped due to failed dependency
	require.Len(t, pipeline.results, 2)

	dbResult := findResult(pipeline.results, "db")
	require.NotNil(t, dbResult)
	require.False(t, dbResult.Success)

	apiResult := findResult(pipeline.results, "api")
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.Error(t, err) // Overall error because db failed

	independentResult := findResult(pipeline.results, "independent")
	require.NotNil(t, independentResult)
	require.True(t, independentResult.Success) // independent should succeed

	apiResult := findResult(pipeline.results, "api")
	require.NotNil(t, apiResult)
	require.False(t, apiResult.Success) // api skipped due to db failure
}

func TestWaveExecution_BuildStepRespectsDeps(t *testing.T) {
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	// Both should complete (build steps respect waves when deps exist)
	require.Len(t, pipeline.results, 2)
	for _, r := range pipeline.results {
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

	opts := types.PipelineOptions{Tag: "v1.0.0", DryRun: true}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

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
	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2, NoDeps: true}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	require.Len(t, pipeline.results, 2)
	for _, r := range pipeline.results {
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 3}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	require.Len(t, pipeline.results, 3)
	for _, r := range pipeline.results {
		require.True(t, r.Success)
	}
}

func TestWaveExecution_TransitiveFailure(t *testing.T) {
	t.Parallel()

	// A fails -> B skipped -> C skipped
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 1}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.Error(t, err)
	require.Len(t, pipeline.results, 3)

	aResult := findResult(pipeline.results, "a")
	require.NotNil(t, aResult)
	require.False(t, aResult.Success)

	bResult := findResult(pipeline.results, "b")
	require.NotNil(t, bResult)
	require.False(t, bResult.Success)
	require.Contains(t, bResult.Error.Error(), "dependency a failed")

	cResult := findResult(pipeline.results, "c")
	require.NotNil(t, cResult)
	require.False(t, cResult.Success)
	require.Contains(t, cResult.Error.Error(), "dependency b failed")
}

// findResult finds a result by service name prefix in the results slice.
func findResult(results []types.TaskResult, namePrefix string) *types.TaskResult {
	for _, r := range results {
		if r.ServiceName == namePrefix {
			return &r
		}
	}
	return nil
}
