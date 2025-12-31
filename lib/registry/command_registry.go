package registry

import (
	"fmt"
	"strings"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// StepContext provides all context needed to generate a command for a step.
type StepContext struct {
	Service      serviceinfo.ServiceInfo
	ImageName    string
	Tag          string
	Registry     string
	TemplatePath string
}

// StepHandler generates a command for a specific step type.
// Returns nil if no command should be executed.
type StepHandler func(ctx StepContext) any

// CommandRegistry maps step patterns to handlers.
type CommandRegistry struct {
	handlers map[string]StepHandler
}

// NewCommandRegistry creates a new command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		handlers: make(map[string]StepHandler),
	}
}

// Register adds a step handler to the registry.
// pattern is matched against step names (case-insensitive, supports partial match).
// provider is optional - if empty, matches all providers.
func (cr *CommandRegistry) Register(pattern string, provider string, handler StepHandler) {
	key := cr.buildKey(pattern, provider)
	cr.handlers[key] = handler
}

// GetHandler finds the appropriate handler for a step.
// Uses exact matching on step names for deterministic behavior.
func (cr *CommandRegistry) GetHandler(stepName string, provider string) (StepHandler, bool) {
	stepLower := strings.ToLower(stepName)

	// Try provider-specific exact match first
	if provider != "" {
		key := cr.buildKey(stepLower, provider)
		if handler, ok := cr.handlers[key]; ok {
			return handler, true
		}
	}

	// Try generic exact match (no provider)
	if handler, ok := cr.handlers[stepLower]; ok {
		return handler, true
	}

	return nil, false
}

// buildKey creates a registry key from pattern and provider.
func (*CommandRegistry) buildKey(pattern string, provider string) string {
	pattern = strings.ToLower(pattern)
	if provider != "" {
		return fmt.Sprintf("%s:%s", pattern, provider)
	}
	return pattern
}
