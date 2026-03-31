package orchestrator

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:        "v1.0.0",
		Debug:      true,
		Timeout:    60,
		MaxWorkers: 2,
	}

	pipeline := NewPipeline(services, recipes, opts)

	require.NotNil(t, pipeline)
	require.Len(t, pipeline.services, 2)
	require.NotNil(t, pipeline.recipes)
	require.NotNil(t, pipeline.output)
	require.NotNil(t, pipeline.registry)
	require.Equal(t, opts.Tag, pipeline.options.Tag)
	require.Equal(t, opts.Debug, pipeline.options.Debug)
}

func TestNewPipelineEmptyServices(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{})

	require.NotNil(t, pipeline)
	require.Empty(t, pipeline.services)
	require.Empty(t, pipeline.recipes)
}

func TestPipelineRunNoServices(t *testing.T) {
	t.Parallel()

	pipeline := NewPipeline(nil, nil, types.PipelineOptions{})
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineRunNoRecipeSteps(t *testing.T) {
	t.Parallel()

	services := []serviceinfo.ServiceInfo{
		{Name: "svc", Provider: "unknown"},
	}

	pipeline := NewPipeline(services, nil, types.PipelineOptions{})
	err := pipeline.Run()

	// Services without matching recipes now cause validation errors
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no matching recipe")
}

func TestPipelineOptions(t *testing.T) {
	t.Parallel()

	opts := types.PipelineOptions{
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

func TestPipelineDryRun(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineDryRunWithExplicitCommands(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineWithOnlyTags(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:      "v1.0.0",
		DryRun:   true,
		OnlyTags: []string{"build"},
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineWithExcludeTags(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:         "v1.0.0",
		DryRun:      true,
		ExcludeTags: []string{"deploy"},
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineWithMaxSteps(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:      "v1.0.0",
		DryRun:   true,
		MaxSteps: 1,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineMultipleProviders(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineServiceWithoutRecipe(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:    "v1.0.0",
		DryRun: true,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	// Services without matching recipes now cause validation errors
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no matching recipe")
}

func TestPipelineFullRunWithExecution(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:        "v1.0.0",
		Timeout:    10,
		MaxWorkers: 1,
		DryRun:     false,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineFullRunMultipleServicesParallel(t *testing.T) {
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

	opts := types.PipelineOptions{
		Tag:        "v1.0.0",
		Timeout:    10,
		MaxWorkers: 3, // Run all in parallel
		DryRun:     false,
	}

	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
}

func TestPipelineMultiConfig_SameNameDifferentProviders(t *testing.T) {
	t.Parallel()

	// Two services with the same name but different providers — both should execute independently
	services := []serviceinfo.ServiceInfo{
		{Name: "api", Provider: "gcp-test", Region: "us-central1"},
		{Name: "api", Provider: "aws-test", Region: "us-east-1"},
	}

	recipes := []recepie.RecipeInfo{
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	require.Len(t, pipeline.results, 2)
	for _, r := range pipeline.results {
		require.True(t, r.Success, "service %s should succeed", r.ServiceName)
	}
}

func TestPipelineMultiConfig_TwoServicesUnderSameParent(t *testing.T) {
	t.Parallel()

	// Simulates api-cloud-run/ and api-lambda/ under services/
	services := []serviceinfo.ServiceInfo{
		{Name: "api-cloud-run", Provider: "test", Path: "services/api-cloud-run"},
		{Name: "api-lambda", Provider: "test", Path: "services/api-lambda"},
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

	opts := types.PipelineOptions{Tag: "v1.0.0", Timeout: 10, MaxWorkers: 2}
	pipeline := NewPipeline(services, recipes, opts)
	err := pipeline.Run()

	require.NoError(t, err)
	require.Len(t, pipeline.results, 4) // 2 services * 2 steps
	for _, r := range pipeline.results {
		require.True(t, r.Success, "service %s step %s should succeed", r.ServiceName, r.StepName)
	}
}
