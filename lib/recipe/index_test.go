package recipe_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestIndexLookup(t *testing.T) {
	t.Parallel()

	ix := recipe.NewIndex([]recipe.Info{
		{Provider: "gcp", Service: "artifact-registry-image", Recipe: recipe.Recipe{Name: "gcp-artifact-registry-image"}},
		{Provider: "gcp", Service: "cloud-run", Recipe: recipe.Recipe{Name: "gcp-cloud-run"}},
		{Provider: "aws", Service: "lambda", Recipe: recipe.Recipe{Name: "aws-lambda"}},
		{Provider: "homebrew", Service: "package", Recipe: recipe.Recipe{Name: "homebrew"}},
	})

	tests := []struct {
		name     string
		svc      serviceinfo.ServiceInfo
		want     string
		errorHas string
	}{
		{name: "type matches", svc: serviceinfo.ServiceInfo{Type: "gcp-cloud-run", Provider: "gcp"}, want: "gcp-cloud-run"},
		{name: "provider with one recipe", svc: serviceinfo.ServiceInfo{Provider: "aws"}, want: "aws-lambda"},
		{name: "type equal to a one-recipe provider", svc: serviceinfo.ServiceInfo{Type: "homebrew", Provider: "homebrew"}, want: "homebrew"},
		{
			name:     "provider with several recipes is not guessed",
			svc:      serviceinfo.ServiceInfo{Provider: "gcp"},
			errorHas: "set type: to one of gcp-artifact-registry-image, gcp-cloud-run",
		},
		{
			name:     "unknown type does not fall back to the provider",
			svc:      serviceinfo.ServiceInfo{Type: "aws-ecs", Provider: "aws"},
			errorHas: "no recipe named 'aws-ecs'",
		},
		{name: "unknown provider", svc: serviceinfo.ServiceInfo{Provider: "fly"}, errorHas: "no recipe for 'fly'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec, err := ix.Lookup(&tt.svc)
			_, found := ix.Find(&tt.svc)

			if tt.errorHas != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorHas)
				require.False(t, found)
				return
			}

			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, tt.want, rec.Name)
		})
	}
}
