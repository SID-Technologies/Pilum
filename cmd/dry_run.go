package cmd

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	"github.com/sid-technologies/pilum/lib/registry"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run [services...]",
		Short: "Preview commands without executing",
		Long:  "Show what commands would be executed based on each service's recipe.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := viper.BindPFlag("tag", cmd.Flags().Lookup("tag")); err != nil {
				return errors.Wrap(err, "error binding tag flag")
			}
			if err := viper.BindPFlag("registry", cmd.Flags().Lookup("registry")); err != nil {
				return errors.Wrap(err, "error binding registry flag")
			}
			if err := viper.BindPFlag("template-path", cmd.Flags().Lookup("template-path")); err != nil {
				return errors.Wrap(err, "error binding template-path flag")
			}
			if err := viper.BindPFlag("recipe-path", cmd.Flags().Lookup("recipe-path")); err != nil {
				return errors.Wrap(err, "error binding recipe-path flag")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			registryPrefix := viper.GetString("registry")
			templatePath := viper.GetString("template-path")
			recipePath := viper.GetString("recipe-path")

			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}

			if len(services) == 0 {
				fmt.Println("No services found")
				return nil
			}

			// Load recipes
			recipes, err := recepie.LoadRecipesFromDirectory(recipePath)
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}

			// Create recipe map by provider
			recipeMap := make(map[string]*recepie.Recipe)
			for i := range recipes {
				recipeMap[recipes[i].Recipe.Provider] = &recipes[i].Recipe
			}

			// Validate all services first
			for _, service := range services {
				if err := service.Validate(); err != nil {
					return errors.Wrap(err, "service '%s' validation failed", service.Name)
				}
				if recipe, exists := recipeMap[service.Provider]; exists {
					if err := recipe.ValidateService(&service); err != nil {
						return errors.Wrap(err, "service '%s' validation failed", service.Name)
					}
				}
			}

			// Initialize command registry
			cmdRegistry := registry.NewCommandRegistry()
			registry.RegisterDefaultHandlers(cmdRegistry)

			output.Header("Dry Run Preview for %d service(s)", len(services))
			fmt.Println()

			for _, service := range services {
				// Get image name for this service
				_, imageName := build.GenerateBuildCommand(service, registryPrefix, tag)

				fmt.Printf("%s%s%s\n", output.Primary, service.Name, output.Reset)
				fmt.Printf("  %sPath:%s     %s\n", output.Muted, output.Reset, service.Path)
				fmt.Printf("  %sProvider:%s %s\n", output.Muted, output.Reset, service.Provider)
				fmt.Println()

				// Find recipe for this service
				recipe, hasRecipe := recipeMap[service.Provider]
				if !hasRecipe {
					fmt.Printf("  %sNo recipe found for provider '%s'%s\n\n", output.WarningColor, service.Provider, output.Reset)
					continue
				}

				fmt.Printf("  %sRecipe:%s %s\n", output.Muted, output.Reset, recipe.Name)
				fmt.Printf("  %sSteps:%s\n", output.Muted, output.Reset)

				// Build step context
				ctx := registry.StepContext{
					Service:      service,
					ImageName:    imageName,
					Tag:          tag,
					Registry:     registryPrefix,
					TemplatePath: templatePath,
				}

				// Iterate through recipe steps and show what each would do
				for i, step := range recipe.Steps {
					handler, found := cmdRegistry.GetHandler(step.Name, service.Provider)

					fmt.Printf("\n    %d. %s", i+1, step.Name)
					if step.Timeout > 0 {
						fmt.Printf(" %s(timeout: %ds)%s", output.Muted, step.Timeout, output.Reset)
					}
					fmt.Println()

					if !found {
						fmt.Printf("       %s[no handler registered]%s\n", output.Muted, output.Reset)
						continue
					}

					// Get the command from the handler
					cmd := handler(ctx)
					if cmd == nil {
						fmt.Printf("       %s[no command generated]%s\n", output.Muted, output.Reset)
						continue
					}

					// Format the command output
					switch c := cmd.(type) {
					case []string:
						fmt.Printf("       %s%v%s\n", output.SuccessColor, c, output.Reset)
					case string:
						fmt.Printf("       %s%s%s\n", output.SuccessColor, c, output.Reset)
					default:
						fmt.Printf("       %s%v%s\n", output.SuccessColor, c, output.Reset)
					}
				}

				fmt.Println()
				fmt.Println("  " + string(make([]byte, 50)))
				fmt.Println()
			}

			fmt.Printf("%s[Dry run complete - no commands were executed]%s\n", output.Muted, output.Reset)

			return nil
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")
	cmd.Flags().String("registry", "", "Docker registry prefix")
	cmd.Flags().String("template-path", "./_templates", "Path to Dockerfile templates")
	cmd.Flags().String("recipe-path", "./recepies", "Path to recipe definitions")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
