package orchestrator

import (
	"testing"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/sysinfo"
	"github.com/sid-technologies/pilum/lib/types"

	"github.com/stretchr/testify/require"
)

func TestInvokesDocker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  any
		want bool
	}{
		{"nil", nil, false},
		{"string docker", "docker build -t x .", true},
		{"string other", "go build ./...", false},
		{"string mentions docker later", "echo docker", false},
		{"slice docker", []string{"docker", "push", "img"}, true},
		{"slice absolute path", []string{"/usr/local/bin/docker", "info"}, true},
		{"slice other", []string{"gcloud", "run", "deploy"}, false},
		{"any slice docker", []any{"docker", "build"}, true},
		{"empty slice", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, invokesDocker(tt.cmd))
		})
	}
}

func dockerTestPipeline(opts types.PipelineOptions) *Pipeline {
	services := []serviceinfo.ServiceInfo{{Name: "svc", Provider: "test"}}
	recipes := []recipe.Info{{
		Provider: "test",
		Recipe: recipe.Recipe{
			Name:     "test",
			Provider: "test",
			Steps: []recipe.Step{
				{Name: "build binary", Command: "echo built", Tags: []string{"build"}},
				{Name: "build image", Command: []string{"docker", "build", "."}, Tags: []string{"docker"}},
			},
		},
	}}
	return NewPipeline(services, recipes, opts)
}

func TestRequiresDocker(t *testing.T) {
	t.Parallel()

	t.Run("docker step runs", func(t *testing.T) {
		t.Parallel()
		p := dockerTestPipeline(types.PipelineOptions{})
		require.True(t, p.requiresDocker(p.findMaxSteps()))
	})

	t.Run("docker step excluded by tag", func(t *testing.T) {
		t.Parallel()
		p := dockerTestPipeline(types.PipelineOptions{ExcludeTags: []string{"docker"}})
		require.False(t, p.requiresDocker(p.findMaxSteps()))
	})

	t.Run("docker step beyond max steps", func(t *testing.T) {
		t.Parallel()
		p := dockerTestPipeline(types.PipelineOptions{MaxSteps: 1})
		require.False(t, p.requiresDocker(p.findMaxSteps()))
	})
}

// Not parallel: swaps the package-level checkDocker.
//
//nolint:paralleltest // mutates package-level checkDocker
func TestRunFailsFastWhenDockerUnavailable(t *testing.T) {
	original := checkDocker
	t.Cleanup(func() { checkDocker = original })

	calls := 0
	checkDocker = func(time.Duration) (sysinfo.DockerResources, error) {
		calls++
		return sysinfo.DockerResources{}, errors.New("docker daemon is not running")
	}

	p := dockerTestPipeline(types.PipelineOptions{Timeout: 5})
	err := p.Run()

	require.Error(t, err)
	require.Equal(t, 1, calls)
	require.Empty(t, p.Results(), "no step should run when the preflight fails")

	// Dry runs execute nothing, so they must not require a daemon.
	calls = 0
	p = dockerTestPipeline(types.PipelineOptions{DryRun: true})
	require.NoError(t, p.Run())
	require.Zero(t, calls)
}
