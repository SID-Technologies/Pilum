package providers_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/providers"

	"github.com/stretchr/testify/require"
)

func TestIsValidProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider string
		valid    bool
	}{
		{"gcp", true},
		{"aws", true},
		{"azure", true},
		{"homebrew", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.valid, providers.IsValidProvider(tt.provider))
		})
	}
}

func TestIsValidService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider string
		service  string
		valid    bool
	}{
		{"gcp cloud-run", "gcp", "cloud-run", true},
		{"gcp gke", "gcp", "gke", true},
		{"gcp invalid", "gcp", "invalid", false},
		{"aws lambda", "aws", "lambda", true},
		{"aws ecs", "aws", "ecs", true},
		{"homebrew empty", "homebrew", "", true},
		{"homebrew with service", "homebrew", "something", false},
		{"invalid provider", "invalid", "cloud-run", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.valid, providers.IsValidService(tt.provider, tt.service))
		})
	}
}

func TestGetProviders(t *testing.T) {
	t.Parallel()

	providerList := providers.GetProviders()

	require.Contains(t, providerList, "gcp")
	require.Contains(t, providerList, "aws")
	require.Contains(t, providerList, "azure")
	require.Contains(t, providerList, "homebrew")
}

func TestGetServices(t *testing.T) {
	t.Parallel()

	gcpServices := providers.GetServices("gcp")
	require.Contains(t, gcpServices, "cloud-run")
	require.Contains(t, gcpServices, "gke")

	awsServices := providers.GetServices("aws")
	require.Contains(t, awsServices, "lambda")
	require.Contains(t, awsServices, "ecs")

	homebrewServices := providers.GetServices("homebrew")
	require.Empty(t, homebrewServices)

	invalidServices := providers.GetServices("invalid")
	require.Nil(t, invalidServices)
}

func TestGetAllRecipeKeys(t *testing.T) {
	t.Parallel()

	keys := providers.GetAllRecipeKeys()

	require.Contains(t, keys, "gcp-cloud-run")
	require.Contains(t, keys, "gcp-gke")
	require.Contains(t, keys, "aws-lambda")
	require.Contains(t, keys, "homebrew")
}

func TestGetProviderName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Google Cloud Platform", providers.GetProviderName("gcp"))
	require.Equal(t, "Amazon Web Services", providers.GetProviderName("aws"))
	require.Equal(t, "invalid", providers.GetProviderName("invalid"))
}
