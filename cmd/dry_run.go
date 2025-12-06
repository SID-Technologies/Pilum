package cmd

import (
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dry-run [services...]",
		Aliases: []string{"dr"},
		Short:   "Preview commands without executing",
		Long:    "Show what commands would be executed based on each service's recipe.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return bindFlagsForDeploymentCommands(cmd)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := getDeploymentOptions()
			opts.DryRun = true // Always dry-run for this command

			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services")
			}
			if len(services) == 0 {
				output.Warning("No services found")
				return nil
			}

			recipes, err := recepie.LoadRecipesFromDirectory(opts.RecipePath)
			if err != nil {
				return errors.Wrap(err, "error loading recipes")
			}
			if len(recipes) == 0 {
				output.Warning("No recipes found")
				return nil
			}

			runner := orchestrator.NewRunner(services, recipes, opts.toRunnerOptions())
			return runner.Run()
		},
	}

	cmdFlagStringsNoDryRun(cmd)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(DryRunCmd())
}
