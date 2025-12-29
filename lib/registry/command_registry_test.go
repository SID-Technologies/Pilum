package registry_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/stretchr/testify/require"
)

func TestNewCommandRegistry(t *testing.T) {
	t.Parallel()

	cr := registry.NewCommandRegistry()
	require.NotNil(t, cr)
}

func TestCommandRegistryRegisterAndGetHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		registerCalls  []struct{ pattern, provider string }
		lookupStepName string
		lookupProvider string
		shouldFind     bool
	}{
		{
			name: "register and find generic handler",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "build", provider: ""},
			},
			lookupStepName: "build",
			lookupProvider: "",
			shouldFind:     true,
		},
		{
			name: "register and find provider-specific handler",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "deploy", provider: "gcp"},
			},
			lookupStepName: "deploy",
			lookupProvider: "gcp",
			shouldFind:     true,
		},
		{
			name: "provider-specific takes precedence over generic",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "deploy", provider: ""},
				{pattern: "deploy", provider: "gcp"},
			},
			lookupStepName: "deploy",
			lookupProvider: "gcp",
			shouldFind:     true,
		},
		{
			name: "fall back to generic when provider-specific not found",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "build", provider: ""},
			},
			lookupStepName: "build",
			lookupProvider: "aws",
			shouldFind:     true,
		},
		{
			name: "case-insensitive step name matching",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "build", provider: ""},
			},
			lookupStepName: "BUILD",
			lookupProvider: "",
			shouldFind:     true,
		},
		{
			name: "partial match in step name",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "build", provider: ""},
			},
			lookupStepName: "docker build image",
			lookupProvider: "",
			shouldFind:     true,
		},
		{
			name:           "handler not found",
			registerCalls:  []struct{ pattern, provider string }{},
			lookupStepName: "build",
			lookupProvider: "",
			shouldFind:     false,
		},
		{
			name: "different provider not found when specific registered",
			registerCalls: []struct{ pattern, provider string }{
				{pattern: "deploy", provider: "gcp"},
			},
			lookupStepName: "deploy",
			lookupProvider: "aws",
			shouldFind:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cr := registry.NewCommandRegistry()

			// Register handlers
			for _, rc := range tt.registerCalls {
				cr.Register(rc.pattern, rc.provider, func(ctx registry.StepContext) any {
					return "test-command"
				})
			}

			// Try to get handler
			handler, found := cr.GetHandler(tt.lookupStepName, tt.lookupProvider)

			require.Equal(t, tt.shouldFind, found)
			if tt.shouldFind {
				require.NotNil(t, handler)
			} else {
				require.Nil(t, handler)
			}
		})
	}
}

func TestCommandRegistryHandlerExecution(t *testing.T) {
	t.Parallel()

	cr := registry.NewCommandRegistry()

	// Register a handler that returns specific content based on context
	cr.Register("build", "", func(ctx registry.StepContext) any {
		return []string{"go", "build", "-o", ctx.Service.Name}
	})

	cr.Register("deploy", "gcp", func(ctx registry.StepContext) any {
		return map[string]string{
			"provider": ctx.Service.Provider,
			"image":    ctx.ImageName,
		}
	})

	// Test generic handler execution
	handler, found := cr.GetHandler("build", "")
	require.True(t, found)

	ctx := registry.StepContext{
		Service: serviceinfo.ServiceInfo{Name: "myapp"},
	}

	// Execute the handler and verify it returns expected result
	result := handler(ctx)
	require.Equal(t, []string{"go", "build", "-o", "myapp"}, result)
}

func TestCommandRegistryMultipleHandlers(t *testing.T) {
	t.Parallel()

	cr := registry.NewCommandRegistry()

	// Register multiple handlers
	patterns := []struct{ pattern, provider string }{
		{pattern: "build", provider: ""},
		{pattern: "test", provider: ""},
		{pattern: "deploy", provider: "gcp"},
		{pattern: "deploy", provider: "aws"},
		{pattern: "push", provider: ""},
	}

	for _, p := range patterns {
		p := p // capture range variable
		cr.Register(p.pattern, p.provider, func(ctx registry.StepContext) any {
			return p.pattern + ":" + p.provider
		})
	}

	// Verify all handlers can be found
	_, found := cr.GetHandler("build", "")
	require.True(t, found)

	_, found = cr.GetHandler("test", "")
	require.True(t, found)

	_, found = cr.GetHandler("deploy", "gcp")
	require.True(t, found)

	_, found = cr.GetHandler("deploy", "aws")
	require.True(t, found)

	_, found = cr.GetHandler("push", "")
	require.True(t, found)

	// Verify non-existent handler
	_, found = cr.GetHandler("unknown", "")
	require.False(t, found)
}

func TestCommandRegistryOverwrite(t *testing.T) {
	t.Parallel()

	cr := registry.NewCommandRegistry()

	// Register initial handler
	cr.Register("build", "", func(ctx registry.StepContext) any {
		return "first"
	})

	// Overwrite with new handler
	cr.Register("build", "", func(ctx registry.StepContext) any {
		return "second"
	})

	handler, found := cr.GetHandler("build", "")
	require.True(t, found)

	// Execute and verify it's the second handler
	result := handler(registry.StepContext{})
	require.Equal(t, "second", result)
}

func TestCommandRegistryNilHandler(t *testing.T) {
	t.Parallel()

	cr := registry.NewCommandRegistry()

	// Register a handler that returns nil
	cr.Register("skip", "", func(ctx registry.StepContext) any {
		return nil
	})

	handler, found := cr.GetHandler("skip", "")
	require.True(t, found)

	result := handler(registry.StepContext{})
	require.Nil(t, result)
}
