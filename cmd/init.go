package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/suggest"

	"github.com/spf13/cobra"
)

func InitCmd() *cobra.Command {
	var provider string
	var service string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new service configuration",
		Long:  "Interactively create a service.yaml file for a new service.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInit(provider, service)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Cloud provider (gcp, aws, azure, homebrew)")
	cmd.Flags().StringVarP(&service, "service", "s", "", "Deployment service (cloud-run, lambda, etc.)")

	return cmd
}

func runInit(provider, service string) error {
	reader := bufio.NewReader(os.Stdin)

	// Check if service.yaml already exists
	if _, err := os.Stat("service.yaml"); err == nil {
		output.Warning("service.yaml already exists in this directory")
		overwrite, err := prompt(reader, "Overwrite? (y/N)", "n")
		if err != nil {
			return err
		}
		if strings.ToLower(overwrite) != "y" && strings.ToLower(overwrite) != "yes" {
			output.Info("Init canceled")
			return nil
		}
	}

	// Load available recipes
	recipes, err := recepie.LoadEmbeddedRecipes()
	if err != nil {
		return errors.Wrap(err, "failed to load recipes")
	}

	if len(recipes) == 0 {
		return errors.New("no recipes found")
	}

	// If provider not specified, prompt for it using providers registry
	if provider == "" {
		availableProviders := providers.GetProviders()
		output.Header("Available providers")
		for i, p := range availableProviders {
			displayName := providers.GetProviderName(p)
			fmt.Printf("  %d. %s (%s)\n", i+1, p, displayName)
		}
		fmt.Println()

		provider, err = prompt(reader, "Select provider", availableProviders[0])
		if err != nil {
			return err
		}
	}

	// Validate provider
	if !providers.IsValidProvider(provider) {
		suggestion := suggest.FormatSuggestion(provider, providers.GetProviders())
		if suggestion != "" {
			return errors.New("unknown provider '%s' - %s", provider, suggestion)
		}
		return errors.New("unknown provider '%s'", provider)
	}

	// If service not specified and provider has services, prompt for it
	availableServices := providers.GetServices(provider)
	if service == "" && len(availableServices) > 0 {
		output.Header("Available services for %s", provider)
		for i, s := range availableServices {
			fmt.Printf("  %d. %s\n", i+1, s)
		}
		fmt.Println()

		service, err = prompt(reader, "Select service", availableServices[0])
		if err != nil {
			return err
		}
	}

	// Validate service if specified
	if service != "" && !providers.IsValidService(provider, service) {
		suggestion := suggest.FormatSuggestion(service, availableServices)
		if suggestion != "" {
			return errors.New("unknown service '%s' for provider '%s' - %s", service, provider, suggestion)
		}
		return errors.New("unknown service '%s' for provider '%s'", service, provider)
	}

	// Find the recipe for the provider-service combination
	recipeKey := provider
	if service != "" {
		recipeKey = provider + "-" + service
	}
	recipe := findRecipeByKey(recipes, provider, service)
	if recipe == nil {
		output.Warning("No recipe found for '%s' - using generic template", recipeKey)
	}

	if recipe != nil {
		output.Header("Creating service.yaml for %s", recipe.Name)
	} else {
		output.Header("Creating service.yaml for %s", recipeKey)
	}

	// Collect values for required fields
	values := make(map[string]string)

	// Always prompt for service name first
	name, err := prompt(reader, "Service name", filepath.Base(mustGetwd()))
	if err != nil {
		return err
	}
	values["name"] = name

	// Prompt for each required field from the recipe (if we have one)
	if recipe != nil {
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
	}

	// Generate the YAML content
	yaml := generateServiceYAML(provider, service, values)

	// Write to service.yaml
	if err := os.WriteFile("service.yaml", []byte(yaml), 0600); err != nil {
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
	var providerList []string
	for _, r := range recipes {
		if !seen[r.Provider] {
			seen[r.Provider] = true
			providerList = append(providerList, r.Provider)
		}
	}
	return providerList
}

func findRecipeByProvider(recipes []recepie.RecipeInfo, provider string) *recepie.Recipe {
	for _, r := range recipes {
		if r.Provider == provider {
			return &r.Recipe
		}
	}
	return nil
}

func findRecipeByKey(recipes []recepie.RecipeInfo, provider, service string) *recepie.Recipe {
	// First, try exact provider-service match
	for _, r := range recipes {
		if r.Provider == provider && r.Service == service {
			return &r.Recipe
		}
	}
	// Fallback to provider-only match if no exact match found
	return findRecipeByProvider(recipes, provider)
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "my-service"
	}
	return dir
}

func generateServiceYAML(provider, service string, values map[string]string) string {
	var sb strings.Builder

	// Write header comment
	sb.WriteString("# Service configuration for Pilum\n")
	sb.WriteString("# See: https://github.com/sid-technologies/pilum\n\n")

	// Write name, provider, and service first
	sb.WriteString(fmt.Sprintf("name: %s\n", values["name"]))
	sb.WriteString(fmt.Sprintf("provider: %s\n", provider))
	if service != "" {
		sb.WriteString(fmt.Sprintf("service: %s\n", service))
	}

	// Group nested keys (like homebrew.project_url) to write them together
	nestedKeys := make(map[string]map[string]string)

	// Write remaining fields
	for key, value := range values {
		if key == "name" {
			continue // Already written
		}

		// Handle nested keys like "homebrew.project_url"
		if strings.Contains(key, ".") {
			parts := strings.SplitN(key, ".", 2)
			if nestedKeys[parts[0]] == nil {
				nestedKeys[parts[0]] = make(map[string]string)
			}
			nestedKeys[parts[0]][parts[1]] = value
		} else {
			sb.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	// Write nested sections
	for section, fields := range nestedKeys {
		sb.WriteString(fmt.Sprintf("\n%s:\n", section))
		for key, value := range fields {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// Add build section placeholder
	sb.WriteString("\n# Build configuration\n")
	sb.WriteString("build:\n")
	sb.WriteString("  language: go\n")
	sb.WriteString("  version: \"1.23\"\n")

	// Add provider-specific config sections with example values
	sb.WriteString(generateProviderConfig(provider, service))

	return sb.String()
}

func generateProviderConfig(provider, service string) string {
	var sb strings.Builder

	switch provider {
	case "gcp":
		if service == "cloud-run" {
			sb.WriteString("\n# Cloud Run configuration\n")
			sb.WriteString("cloud_run:\n")
			sb.WriteString("  min_instances: 0      # Scale to zero when idle\n")
			sb.WriteString("  max_instances: 10     # Maximum instances to scale to\n")
			sb.WriteString("  cpu_throttling: true  # Throttle CPU when not processing requests\n")
			sb.WriteString("  memory: 512Mi         # Memory per instance\n")
			sb.WriteString("  cpu: \"1\"              # CPUs per instance\n")
			sb.WriteString("  concurrency: 80       # Max concurrent requests per instance\n")
			sb.WriteString("  timeout: 300          # Request timeout in seconds\n")
		}
	case "homebrew":
		// Homebrew config is already captured in required fields (homebrew.*)
		// but we can add a comment
		sb.WriteString("\n# Homebrew-specific configuration is in the 'homebrew' section above\n")
	case "aws":
		switch service {
		case "lambda":
			sb.WriteString("\n# Lambda configuration (example)\n")
			sb.WriteString("# lambda:\n")
			sb.WriteString("#   memory: 128          # Memory in MB\n")
			sb.WriteString("#   timeout: 30          # Timeout in seconds\n")
			sb.WriteString("#   runtime: provided.al2023\n")
		case "ecs", "fargate":
			sb.WriteString("\n# ECS/Fargate configuration (example)\n")
			sb.WriteString("# ecs:\n")
			sb.WriteString("#   cpu: 256             # CPU units\n")
			sb.WriteString("#   memory: 512          # Memory in MB\n")
			sb.WriteString("#   desired_count: 1     # Number of tasks\n")
		default:
			// No config for other AWS services
		}
	case "azure":
		if service == "container-apps" {
			sb.WriteString("\n# Container Apps configuration (example)\n")
			sb.WriteString("# container_apps:\n")
			sb.WriteString("#   min_replicas: 0\n")
			sb.WriteString("#   max_replicas: 10\n")
			sb.WriteString("#   cpu: 0.5\n")
			sb.WriteString("#   memory: 1Gi\n")
		}
	default:
		// No provider-specific config for unknown providers
	}

	return sb.String()
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(InitCmd())
}
