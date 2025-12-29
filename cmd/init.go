package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"

	"github.com/spf13/cobra"
)

func InitCmd() *cobra.Command {
	var provider string
	var recipePath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new service configuration",
		Long:  "Interactively create a service.yaml file for a new service.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInit(provider, recipePath)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Cloud provider (gcp, aws, homebrew)")
	cmd.Flags().StringVar(&recipePath, "recipe-path", "./recepies", "Path to recipe directory")

	return cmd
}

func runInit(provider, recipePath string) error {
	reader := bufio.NewReader(os.Stdin)

	// Check if service.yaml already exists
	if _, err := os.Stat("service.yaml"); err == nil {
		output.Warning("service.yaml already exists in this directory")
		overwrite, err := prompt(reader, "Overwrite? (y/N)", "n")
		if err != nil {
			return err
		}
		if strings.ToLower(overwrite) != "y" && strings.ToLower(overwrite) != "yes" {
			output.Info("Init cancelled")
			return nil
		}
	}

	// Load available recipes
	recipes, err := recepie.LoadRecipesFromDirectory(recipePath)
	if err != nil {
		return errors.Wrap(err, "failed to load recipes")
	}

	if len(recipes) == 0 {
		return errors.New("no recipes found in %s", recipePath)
	}

	// If provider not specified, prompt for it
	if provider == "" {
		providers := getAvailableProviders(recipes)
		output.Header("Available providers")
		for i, p := range providers {
			fmt.Printf("  %d. %s\n", i+1, p)
		}
		fmt.Println()

		provider, err = prompt(reader, "Select provider", providers[0])
		if err != nil {
			return err
		}
	}

	// Find the recipe for the provider
	recipe := findRecipeByProvider(recipes, provider)
	if recipe == nil {
		return errors.New("no recipe found for provider: %s", provider)
	}

	output.Header("Creating service.yaml for %s", recipe.Name)

	// Collect values for required fields
	values := make(map[string]string)

	// Always prompt for service name first
	name, err := prompt(reader, "Service name", filepath.Base(mustGetwd()))
	if err != nil {
		return err
	}
	values["name"] = name

	// Prompt for each required field from the recipe
	for _, field := range recipe.RequiredFields {
		// Skip if we already have this field
		if _, ok := values[field.Name]; ok {
			continue
		}

		promptText := field.Name
		if field.Description != "" {
			promptText = fmt.Sprintf("%s (%s)", field.Name, field.Description)
		}

		defaultVal := field.Default
		value, err := prompt(reader, promptText, defaultVal)
		if err != nil {
			return err
		}
		values[field.Name] = value
	}

	// Generate the YAML content
	yaml := generateServiceYAML(provider, values)

	// Write to service.yaml
	if err := os.WriteFile("service.yaml", []byte(yaml), 0644); err != nil {
		return errors.Wrap(err, "failed to write service.yaml")
	}

	output.Success("Created service.yaml")
	output.Dimmed("Run 'pilum check' to validate your configuration")

	return nil
}

func prompt(reader *bufio.Reader, label, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.Wrap(err, "failed to read input")
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

func getAvailableProviders(recipes []recepie.RecipeInfo) []string {
	seen := make(map[string]bool)
	var providers []string
	for _, r := range recipes {
		if !seen[r.Provider] {
			seen[r.Provider] = true
			providers = append(providers, r.Provider)
		}
	}
	return providers
}

func findRecipeByProvider(recipes []recepie.RecipeInfo, provider string) *recepie.Recipe {
	for _, r := range recipes {
		if r.Provider == provider {
			return &r.Recipe
		}
	}
	return nil
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "my-service"
	}
	return dir
}

func generateServiceYAML(provider string, values map[string]string) string {
	var sb strings.Builder

	// Write header comment
	sb.WriteString("# Service configuration for Pilum\n")
	sb.WriteString("# See: https://github.com/sid-technologies/pilum\n\n")

	// Write name and provider first
	sb.WriteString(fmt.Sprintf("name: %s\n", values["name"]))
	sb.WriteString(fmt.Sprintf("provider: %s\n", provider))

	// Write remaining fields
	for key, value := range values {
		if key == "name" {
			continue // Already written
		}

		// Handle nested keys like "homebrew.project_url"
		if strings.Contains(key, ".") {
			parts := strings.SplitN(key, ".", 2)
			// For now, write as flat key - could be enhanced to write nested YAML
			sb.WriteString(fmt.Sprintf("%s:\n  %s: %s\n", parts[0], parts[1], value))
		} else {
			sb.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	// Add build section placeholder based on provider
	sb.WriteString("\n# Build configuration\n")
	sb.WriteString("build:\n")
	sb.WriteString("  language: go\n")
	sb.WriteString("  version: \"1.23\"\n")

	return sb.String()
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(InitCmd())
}
