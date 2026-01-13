package cmd

import (
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deploymentOptions holds parsed flag values for deployment commands.
type deploymentOptions struct {
	Tag         string
	Debug       bool
	Timeout     int
	Retries     int
	DryRun      bool
	MaxWorkers  int
	OnlyTags    []string
	ExcludeTags []string
	OnlyChanged bool
	Since       string
}

// getDeploymentOptions extracts all standard deployment flags from viper.
func getDeploymentOptions() deploymentOptions {
	return deploymentOptions{
		Tag:         viper.GetString("tag"),
		Debug:       viper.GetBool("debug"),
		Timeout:     viper.GetInt("timeout"),
		Retries:     viper.GetInt("retries"),
		DryRun:      viper.GetBool("dry-run"),
		MaxWorkers:  viper.GetInt("max-workers"),
		OnlyTags:    parseCommaSeparated(viper.GetString("only-tags")),
		ExcludeTags: parseCommaSeparated(viper.GetString("exclude-tags")),
		OnlyChanged: viper.GetBool("only-changed"),
		Since:       viper.GetString("since"),
	}
}

// toRunnerOptions converts deploymentOptions to orchestrator.RunnerOptions.
func (o deploymentOptions) toRunnerOptions() orchestrator.RunnerOptions {
	return orchestrator.RunnerOptions{
		Tag:         o.Tag,
		Debug:       o.Debug,
		Timeout:     o.Timeout,
		Retries:     o.Retries,
		DryRun:      o.DryRun,
		MaxWorkers:  o.MaxWorkers,
		OnlyTags:    o.OnlyTags,
		ExcludeTags: o.ExcludeTags,
	}
}

func bindFlagsForDeploymentCommands(cmd *cobra.Command) error {
	flagBindings := []string{
		"tag",
		"debug",
		"timeout",
		"retries",
		"dry-run",
		"max-workers",
		"only-tags",
		"exclude-tags",
		"only-changed",
		"since",
	}

	for _, flag := range flagBindings {
		if f := cmd.Flags().Lookup(flag); f != nil {
			if err := viper.BindPFlag(flag, f); err != nil {
				return errors.Wrap(err, "error binding %s flag", flag)
			}
		}
	}

	return nil
}

// addCommandFlags adds standard deployment flags to a command.
// Set includeDryRun to false for commands that are always dry-run mode.
func addCommandFlags(cmd *cobra.Command, includeDryRun bool) {
	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().IntP("timeout", "T", 60, "Timeout for the build process in seconds")
	cmd.Flags().IntP("retries", "r", 3, "Number of retries for the build process")
	cmd.Flags().Int("max-workers", 0, "Maximum parallel workers (0 = auto)")
	cmd.Flags().String("only-tags", "", "Only run steps with these tags (comma-separated)")
	cmd.Flags().String("exclude-tags", "", "Exclude steps with these tags (comma-separated)")
	cmd.Flags().Bool("only-changed", false, "Only deploy services with changes since base branch")
	cmd.Flags().String("since", "", "Git ref to compare against (default: main or master)")

	if includeDryRun {
		cmd.Flags().BoolP("dry-run", "D", false, "Perform a dry run without executing the build")
	}
}

// runPipeline executes the common deployment pipeline: find services → load recipes → run.
// The noServicesMsg is shown as a warning if no services are found.
func runPipeline(args []string, opts deploymentOptions, noServicesMsg string) error {
	filterOpts := serviceinfo.FilterOptions{
		Names:       args,
		OnlyChanged: opts.OnlyChanged,
		Since:       opts.Since,
		NoGitIgnore: NoGitIgnore(),
	}

	services, err := serviceinfo.FindAndFilterServicesWithOptions(".", filterOpts)
	if err != nil {
		return errors.Wrap(err, "error finding services")
	}

	if len(services) == 0 {
		output.Warning(noServicesMsg)
		return nil
	}

	recipes, err := recepie.LoadEmbeddedRecipes()
	if err != nil {
		return errors.Wrap(err, "error loading recipes")
	}

	if len(recipes) == 0 {
		output.Warning("No recipes found")
		return nil
	}

	runner := orchestrator.NewRunner(services, recipes, opts.toRunnerOptions())
	return runner.Run()
}

// parseCommaSeparated splits a comma-separated string into a slice, trimming whitespace.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
