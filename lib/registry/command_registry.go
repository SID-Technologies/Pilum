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
func (cr *CommandRegistry) GetHandler(stepName string, provider string) (StepHandler, bool) {
	stepLower := strings.ToLower(stepName)

	// Try provider-specific first
	if provider != "" {
		for pattern, handler := range cr.handlers {
			patternParts := strings.Split(pattern, ":")
			if len(patternParts) == 2 {
				patternName := patternParts[0]
				patternProvider := patternParts[1]
				if patternProvider == provider && strings.Contains(stepLower, patternName) {
					return handler, true
				}
			}
		}
	}

	// Fall back to generic handlers (no provider specified)
	for pattern, handler := range cr.handlers {
		if !strings.Contains(pattern, ":") && strings.Contains(stepLower, pattern) {
			return handler, true
		}
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
