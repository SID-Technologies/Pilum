package cmd

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/ingredients/docker"
	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run [services...]",
		Short: "Preview commands without executing",
		Long:  "Show what commands would be executed for build, publish, push, and deploy operations without actually running them.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			err := viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
			if err != nil {
				return errors.Wrap(err, "error binding tag flag: %v", err)
			}
			err = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))
			if err != nil {
				return errors.Wrap(err, "error binding registry flag: %v", err)
			}
			err = viper.BindPFlag("template-path", cmd.Flags().Lookup("template-path"))
			if err != nil {
				return errors.Wrap(err, "error binding template-path flag: %v", err)
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			registry := viper.GetString("registry")
			templatePath := viper.GetString("template-path")

			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}

			if len(services) == 0 {
				fmt.Println("No services found")
				return nil
			}

			// Load recipes to show what would be executed
			recipes, err := recepie.LoadRecipesFromDirectory("./recepies")
			if err != nil {
				// Recipes are optional for dry-run
				fmt.Printf("Note: Could not load recipes: %v\n\n", err)
			}

			// Create a map of recipes by provider/type
			recipeMap := make(map[string]recepie.Recipe)
			for _, r := range recipes {
				key := r.Recipe.Provider
				recipeMap[key] = r.Recipe
			}

			fmt.Printf("Dry Run Preview for %d service(s)\n", len(services))
			fmt.Println("=" + string(make([]byte, 50)))

			for _, service := range services {
				fmt.Printf("\nService: %s\n", service.Name)
				fmt.Printf("  Path: %s\n", service.Path)
				fmt.Printf("  Provider: %s\n", service.Provider)
				fmt.Printf("  Template: %s\n", service.Template)
				fmt.Println()

				// Show build command
				buildCmd, imageName := build.GenerateBuildCommand(service, registry, tag)
				if buildCmd != nil {
					cmdStr := build.GenerateBuildCommandString(service)
					fmt.Println("  1. BUILD:")
					fmt.Printf("     cd %s\n", service.Path)
					fmt.Printf("     %s\n", cmdStr)
					if len(service.BuildConfig.EnvVars) > 0 {
						fmt.Println("     Environment:")
						for _, ev := range service.BuildConfig.EnvVars {
							fmt.Printf("       %s=%s\n", ev.Name, ev.Value)
						}
					}
					fmt.Println()
				}

				// Show docker build command
				if service.Template != "" {
					dockerfilePath := fmt.Sprintf("%s/%s", templatePath, service.Template)
					dockerCmd := docker.GenerateDockerBuildCommand(service, imageName, dockerfilePath)
					fmt.Println("  2. DOCKER BUILD:")
					fmt.Printf("     %v\n", dockerCmd)
					fmt.Println()
				}

				// Show docker push command
				pushCmd := docker.GenerateDockerPushCommand(imageName)
				fmt.Println("  3. DOCKER PUSH:")
				fmt.Printf("     %v\n", pushCmd)
				fmt.Println()

				// Show deploy command based on provider
				fmt.Println("  4. DEPLOY:")
				switch service.Provider {
				case "gcp":
					deployCmd := gcp.GenerateGCPDeployCommand(service, imageName)
					fmt.Printf("     %v\n", deployCmd)
				case "aws":
					fmt.Println("     [AWS deployment - provider not yet implemented]")
				case "azure":
					fmt.Println("     [Azure deployment - provider not yet implemented]")
				default:
					fmt.Printf("     [Unknown provider: %s]\n", service.Provider)
				}
				fmt.Println()

				// Show recipe steps if available
				if r, exists := recipeMap[service.Provider]; exists {
					fmt.Printf("  Recipe: %s\n", r.Name)
					fmt.Println("  Steps:")
					for i, step := range r.Steps {
						fmt.Printf("    %d. %s", i+1, step.Name)
						if step.Command != nil {
							fmt.Printf(" -> %v", step.Command)
						}
						if step.Timeout > 0 {
							fmt.Printf(" (timeout: %ds)", step.Timeout)
						}
						fmt.Println()
					}
				}

				fmt.Println("  " + string(make([]byte, 40)))
			}

			fmt.Println("\n[Dry run complete - no commands were executed]")

			return nil
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")
	cmd.Flags().String("registry", "", "Docker registry prefix")
	cmd.Flags().String("template-path", "./_templates", "Path to Dockerfile templates")

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
