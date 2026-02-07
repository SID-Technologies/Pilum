package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/providers"
	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/suggest"
	"github.com/sid-technologies/pilum/lib/templates"

	"github.com/spf13/cobra"
)

func InitCmd() *cobra.Command {
	var provider string
	var service string
	var name string
	var language string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new service configuration",
		Long: `Interactively create a pilum.yaml file for a new service.

Non-interactive mode: provide --provider, --name, and --language flags
to skip all prompts and generate directly. Useful for CI/CD and AI agents.

  pilum init --provider=gcp --service=cloud-run --name=my-api --language=go
  pilum init -p cloudflare -s pages -n my-site -l node --force`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInit(initOptions{
				provider: provider,
				service:  service,
				name:     name,
				language: language,
				force:    force,
			})
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Cloud provider (gcp, aws, azure, cloudflare, homebrew, npm)")
	cmd.Flags().StringVarP(&service, "service", "s", "", "Deployment service (cloud-run, lambda, pages, etc.)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Service name (non-interactive mode)")
	cmd.Flags().StringVarP(&language, "language", "l", "", "Build language (non-interactive mode)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing pilum.yaml without prompting")

	return cmd
}

// initOptions holds all flags for the init command.
type initOptions struct {
	provider string
	service  string
	name     string
	language string
	force    bool
}

// isNonInteractive returns true when enough flags are provided to skip all prompts.
func (o initOptions) isNonInteractive() bool {
	return o.provider != "" && o.name != "" && o.language != ""
}

func runInit(opts initOptions) error {
	// Non-interactive mode: skip all prompts
	if opts.isNonInteractive() {
		return runInitNonInteractive(opts)
	}

	return runInitInteractive(opts)
}

func runInitNonInteractive(opts initOptions) error {
	// Validate provider
	if !providers.IsValidProvider(opts.provider) {
		suggestion := suggest.FormatSuggestion(opts.provider, providers.GetProviders())
		if suggestion != "" {
			return exitcodes.WithCode(exitcodes.InvalidArgs,
				errors.New("unknown provider '%s' - %s", opts.provider, suggestion))
		}
		return exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("unknown provider '%s'", opts.provider))
	}

	// Validate service if provider has services and one was given
	availableServices := providers.GetServices(opts.provider)
	if len(availableServices) > 0 && opts.service == "" {
		return exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("provider '%s' requires --service flag (options: %s)",
				opts.provider, strings.Join(availableServices, ", ")))
	}
	if opts.service != "" && !providers.IsValidService(opts.provider, opts.service) {
		suggestion := suggest.FormatSuggestion(opts.service, availableServices)
		if suggestion != "" {
			return exitcodes.WithCode(exitcodes.InvalidArgs,
				errors.New("unknown service '%s' for provider '%s' - %s", opts.service, opts.provider, suggestion))
		}
		return exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("unknown service '%s' for provider '%s'", opts.service, opts.provider))
	}

	// Validate language
	languageList := templates.GetAvailableLanguages()
	if !slices.Contains(languageList, opts.language) {
		suggestion := suggest.FormatSuggestion(opts.language, languageList)
		if suggestion != "" {
			return exitcodes.WithCode(exitcodes.InvalidArgs,
				errors.New("unknown language '%s' - %s", opts.language, suggestion))
		}
		return exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("unknown language '%s' (available: %s)", opts.language, strings.Join(languageList, ", ")))
	}

	// Check for existing file
	if !opts.force {
		if _, err := os.Stat("pilum.yaml"); err == nil {
			return exitcodes.WithCode(exitcodes.IO,
				errors.New("pilum.yaml already exists (use --force to overwrite)"))
		}
	}

	// Load recipes and find match
	recipes, err := recepie.LoadEmbeddedRecipes()
	if err != nil {
		return exitcodes.WithCode(exitcodes.Config, errors.Wrap(err, "failed to load recipes"))
	}

	recipe := findRecipeByKey(recipes, opts.provider, opts.service)

	// Build values from recipe defaults
	values := map[string]string{"name": opts.name}
	if recipe != nil {
		// Apply defaults from required fields
		for _, field := range recipe.GetRequiredFields() {
			if field.Default != "" {
				if _, exists := values[field.Name]; !exists {
					values[field.Name] = field.Default
				}
			}
		}
		// Apply defaults from optional fields
		for _, field := range recipe.GetOptionalFields() {
			if field.Default != "" {
				values[field.Name] = field.Default
			}
		}
	}

	// Load build template
	buildConfig, err := templates.GetBuildConfig(opts.language)
	if err != nil {
		return errors.Wrap(err, "failed to load build template for %s", opts.language)
	}

	// Generate and write
	yaml := generateServiceYAML(opts.provider, opts.service, values, buildConfig)

	if err := os.WriteFile("pilum.yaml", []byte(yaml), 0600); err != nil {
		return exitcodes.WithCode(exitcodes.IO, errors.Wrap(err, "failed to write pilum.yaml"))
	}

	output.Success("Created pilum.yaml")
	output.Dimmed("Run 'pilum check' to validate your configuration")

	return nil
}

func runInitInteractive(opts initOptions) error {
	reader := bufio.NewReader(os.Stdin)

	// Check if pilum.yaml already exists
	if _, err := os.Stat("pilum.yaml"); err == nil {
		if opts.force {
			output.Warning("Overwriting existing pilum.yaml")
		} else {
			output.Warning("pilum.yaml already exists in this directory")
			overwrite, promptErr := prompt(reader, "Overwrite? (y/N)", "n")
			if promptErr != nil {
				return promptErr
			}
			if strings.ToLower(overwrite) != "y" && strings.ToLower(overwrite) != "yes" {
				output.Info("Init canceled")
				return nil
			}
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

	// Step 1: Select provider
	provider, err := selectProvider(reader, opts.provider)
	if err != nil {
		return err
	}

	// Step 2: Select service (if applicable)
	service, err := selectService(reader, provider, opts.service)
	if err != nil {
		return err
	}

	// Step 3: Find recipe
	recipe := findRecipeByKey(recipes, provider, service)
	recipeKey := provider
	if service != "" {
		recipeKey = provider + "-" + service
	}

	if recipe == nil {
		output.Warning("No recipe found for '%s' - using minimal template", recipeKey)
	} else {
		output.Header("Creating pilum.yaml for %s", recipe.Name)
	}

	// Step 4: Collect required field values
	values := make(map[string]string)

	// Use --name flag if provided, otherwise prompt
	if opts.name != "" {
		values["name"] = opts.name
	} else {
		name, promptErr := prompt(reader, "Service name", filepath.Base(mustGetwd()))
		if promptErr != nil {
			return promptErr
		}
		values["name"] = name
	}

	// Prompt for required fields from recipe
	if recipe != nil {
		if err := promptForFields(reader, recipe.GetRequiredFields(), values, true); err != nil {
			return err
		}
	}

	// Step 5: Prompt for optional fields
	if recipe != nil && len(recipe.GetOptionalFields()) > 0 {
		fmt.Println()
		output.Header("Optional configuration (press Enter to use defaults)")
		if err := promptForFields(reader, recipe.GetOptionalFields(), values, false); err != nil {
			return err
		}
	}

	// Step 6: Select build language
	var language string
	if opts.language != "" {
		language = opts.language
	} else {
		language, err = selectLanguage(reader)
		if err != nil {
			return err
		}
	}

	// Step 7: Load build template
	buildConfig, err := templates.GetBuildConfig(language)
	if err != nil {
		return errors.Wrap(err, "failed to load build template for %s", language)
	}

	// Step 8: Generate and write pilum.yaml
	yaml := generateServiceYAML(provider, service, values, buildConfig)

	if err := os.WriteFile("pilum.yaml", []byte(yaml), 0600); err != nil {
		return errors.Wrap(err, "failed to write pilum.yaml")
	}

	output.Success("Created pilum.yaml")
	output.Dimmed("Run 'pilum check' to validate your configuration")

	return nil
}

func selectProvider(reader *bufio.Reader, providerFlag string) (string, error) {
	if providerFlag != "" {
		if !providers.IsValidProvider(providerFlag) {
			suggestion := suggest.FormatSuggestion(providerFlag, providers.GetProviders())
			if suggestion != "" {
				return "", errors.New("unknown provider '%s' - %s", providerFlag, suggestion)
			}
			return "", errors.New("unknown provider '%s'", providerFlag)
		}
		return providerFlag, nil
	}

	availableProviders := providers.GetProviders()
	output.Header("Available providers")
	for i, p := range availableProviders {
		displayName := providers.GetProviderName(p)
		fmt.Printf("  %d. %s (%s)\n", i+1, p, displayName)
	}
	fmt.Println()

	provider, err := prompt(reader, "Select provider", availableProviders[0])
	if err != nil {
		return "", err
	}

	// Allow selection by number
	if num, parseErr := strconv.Atoi(provider); parseErr == nil && num > 0 && num <= len(availableProviders) {
		provider = availableProviders[num-1]
	}

	if !providers.IsValidProvider(provider) {
		suggestion := suggest.FormatSuggestion(provider, availableProviders)
		if suggestion != "" {
			return "", errors.New("unknown provider '%s' - %s", provider, suggestion)
		}
		return "", errors.New("unknown provider '%s'", provider)
	}

	return provider, nil
}

func selectService(reader *bufio.Reader, provider, serviceFlag string) (string, error) {
	availableServices := providers.GetServices(provider)
	if len(availableServices) == 0 {
		return "", nil
	}

	if serviceFlag != "" {
		if !providers.IsValidService(provider, serviceFlag) {
			suggestion := suggest.FormatSuggestion(serviceFlag, availableServices)
			if suggestion != "" {
				return "", errors.New("unknown service '%s' for provider '%s' - %s", serviceFlag, provider, suggestion)
			}
			return "", errors.New("unknown service '%s' for provider '%s'", serviceFlag, provider)
		}
		return serviceFlag, nil
	}

	output.Header("Available services for %s", provider)
	for i, s := range availableServices {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Println()

	service, err := prompt(reader, "Select service", availableServices[0])
	if err != nil {
		return "", err
	}

	// Allow selection by number
	if num, parseErr := strconv.Atoi(service); parseErr == nil && num > 0 && num <= len(availableServices) {
		service = availableServices[num-1]
	}

	if !providers.IsValidService(provider, service) {
		suggestion := suggest.FormatSuggestion(service, availableServices)
		if suggestion != "" {
			return "", errors.New("unknown service '%s' for provider '%s' - %s", service, provider, suggestion)
		}
		return "", errors.New("unknown service '%s' for provider '%s'", service, provider)
	}

	return service, nil
}

func selectLanguage(reader *bufio.Reader) (string, error) {
	languages := templates.GetAvailableLanguages()
	if len(languages) == 0 {
		return "go", nil // fallback
	}

	fmt.Println()
	output.Header("Build language")
	for i, lang := range languages {
		fmt.Printf("  %d. %s\n", i+1, lang)
	}
	fmt.Println()

	language, err := prompt(reader, "Select language", languages[0])
	if err != nil {
		return "", err
	}

	// Allow selection by number
	if num, parseErr := strconv.Atoi(language); parseErr == nil && num > 0 && num <= len(languages) {
		language = languages[num-1]
	}

	// Validate language exists
	if slices.Contains(languages, language) {
		return language, nil
	}

	suggestion := suggest.FormatSuggestion(language, languages)
	if suggestion != "" {
		return "", errors.New("unknown language '%s' - %s", language, suggestion)
	}
	return "", errors.New("unknown language '%s'", language)
}

func promptForFields(reader *bufio.Reader, fields []recepie.Field, values map[string]string, required bool) error {
	for _, field := range fields {
		// Skip if we already have this field (e.g., "name" is always prompted first)
		if _, ok := values[field.Name]; ok {
			continue
		}

		promptText := field.Name
		if field.Description != "" {
			promptText = fmt.Sprintf("%s (%s)", field.Name, field.Description)
		}

		value, err := prompt(reader, promptText, field.Default)
		if err != nil {
			return err
		}

		// For required fields, ensure a value is provided
		if required && value == "" && field.Default == "" {
			return errors.New("field '%s' is required", field.Name)
		}

		// Only store non-empty values for optional fields
		if value != "" {
			values[field.Name] = value
		}
	}
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

func findRecipeByKey(recipes []recepie.RecipeInfo, provider, service string) *recepie.Recipe {
	// First, try exact provider-service match
	for _, r := range recipes {
		if r.Provider == provider && r.Service == service {
			return &r.Recipe
		}
	}
	// Fallback to provider-only match if no exact match found
	for _, r := range recipes {
		if r.Provider == provider && r.Service == "" {
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

func generateServiceYAML(provider, service string, values map[string]string, buildConfig *templates.BuildConfig) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Service configuration for Pilum\n")
	sb.WriteString("# See: https://github.com/sid-technologies/pilum\n\n")

	// Core fields
	sb.WriteString(fmt.Sprintf("name: %s\n", values["name"]))
	sb.WriteString(fmt.Sprintf("provider: %s\n", provider))
	if service != "" {
		sb.WriteString(fmt.Sprintf("type: %s-%s\n", provider, service))
	}

	// Group values by section (nested keys like "cloud_run.min_instances")
	topLevel := make(map[string]string)
	nested := make(map[string]map[string]string)

	for key, value := range values {
		if key == "name" {
			continue // Already written
		}

		if strings.Contains(key, ".") {
			parts := strings.SplitN(key, ".", 2)
			section := parts[0]
			field := parts[1]
			if nested[section] == nil {
				nested[section] = make(map[string]string)
			}
			nested[section][field] = value
		} else {
			topLevel[key] = value
		}
	}

	// Write top-level fields
	for key, value := range topLevel {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}

	// Write build configuration
	sb.WriteString("\n# Build configuration\n")
	sb.WriteString("build:\n")
	sb.WriteString(fmt.Sprintf("  language: %s\n", buildConfig.Language))
	sb.WriteString(fmt.Sprintf("  version: \"%s\"\n", buildConfig.Version))
	sb.WriteString(fmt.Sprintf("  cmd: \"%s\"\n", buildConfig.Cmd))

	if len(buildConfig.EnvVars) > 0 {
		sb.WriteString("  env_vars:\n")
		for k, v := range buildConfig.EnvVars {
			sb.WriteString(fmt.Sprintf("    %s: \"%s\"\n", k, v))
		}
	}

	if len(buildConfig.Flags) > 0 {
		sb.WriteString("  flags:\n")
		for k, v := range buildConfig.Flags {
			sb.WriteString(fmt.Sprintf("    %s:\n", k))
			switch val := v.(type) {
			case []any:
				for _, item := range val {
					sb.WriteString(fmt.Sprintf("      - \"%v\"\n", item))
				}
			case string:
				sb.WriteString(fmt.Sprintf("      - \"%s\"\n", val))
			}
		}
	}

	// Write nested sections (e.g., cloud_run, homebrew, lambda)
	for section, fields := range nested {
		sb.WriteString(fmt.Sprintf("\n# %s configuration\n", section))
		sb.WriteString(fmt.Sprintf("%s:\n", section))
		for key, value := range fields {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	return sb.String()
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(InitCmd())
}
