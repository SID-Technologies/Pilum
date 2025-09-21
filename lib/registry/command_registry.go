package registry

import (
	"fmt"

	"github.com/sid-technologies/centurion/lib/errors"
	r "github.com/sid-technologies/centurion/lib/recepie"
)

// CommandFunc represents a CLI command that returns string arrays.
type CommandFunc func() []string

// CommandRegistry maps provider/service/step combinations to command functions.
type CommandRegistry struct {
	commands map[string]CommandFunc
}

// NewCommandRegistry creates a new command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandFunc),
	}
}

// Register adds a command function to the registry.
func (cr *CommandRegistry) Register(provider, service, step string, cmdFunc CommandFunc) {
	key := fmt.Sprintf("%s:%s:%s", provider, service, step)
	cr.commands[key] = cmdFunc
}

// Get retrieves a command function for the specified provider, service, and step.
func (cr *CommandRegistry) Get(provider, service, step string) (CommandFunc, bool) {
	key := fmt.Sprintf("%s:%s:%s", provider, service, step)
	cmdFunc, exists := cr.commands[key]

	return cmdFunc, exists
}

// GetCommandsForRecipe retrieves all command strings for a recipe.
func (cr *CommandRegistry) GetCommandsForRecipe(recipe r.Recipe) ([]string, error) {
	var allCommands []string

	for _, step := range recipe.Steps {
		cmdFunc, exists := cr.Get(recipe.Provider, recipe.Service, step.Name)
		if !exists {
			return nil, errors.New("no command function found for %s:%s:%s", recipe.Provider, recipe.Service, step.Name)
		}

		// Execute the command function to get the string array
		commands := cmdFunc()
		allCommands = append(allCommands, commands...)
	}

	return allCommands, nil
}
