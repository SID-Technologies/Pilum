package cmd

import (
	"fmt"

	"github.com/sid-technologies/centurion/lib/configs"
	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/sid-technologies/centurion/lib/output"
	"github.com/spf13/cobra"
)

const version = "v0.1.0"

var listServicesCmd = &cobra.Command{
	Use:     "list-services",
	Aliases: []string{"ls", "list", "l"},
	Short:   "List all services with their configuration",
	RunE: func(_ *cobra.Command, _ []string) error {
		cl, err := configs.NewClient()
		if err != nil {
			return errors.Wrap(err, "error creating configs client: %v")
		}

		banner := output.PrintBanner(version)
		fmt.Print(banner)
		fmt.Println("Available Services:")

		configs := cl.Registry.List()
		output.DisplayConfigs(configs)

		fmt.Println("Usage:")
		fmt.Println("  centurion add [service] [flags]    Add a service")
		fmt.Println("  centurion add [service] --help     Add a service interactively")

		return nil
	},
}

//nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(listServicesCmd)
}
