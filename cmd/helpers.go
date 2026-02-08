package cmd

import (
	"strings"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/history"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"
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
	NoDeps      bool
	Env         string
	Provider    string
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
		NoDeps:      viper.GetBool("no-deps"),
		Env:         viper.GetString("env"),
		Provider:    viper.GetString("provider"),
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
		NoDeps:      o.NoDeps,
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
		"no-deps",
		"env",
		"provider",
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
	cmd.Flags().Bool("no-deps", false, "Disable dependency-based deployment ordering")
	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")
	cmd.Flags().String("provider", "", "Filter services by provider (e.g., gcp, aws, azure)")

	if includeDryRun {
		cmd.Flags().BoolP("dry-run", "D", false, "Perform a dry run without executing the build")
	}
}

// runPipeline executes the common deployment pipeline: find services → load recipes → run.
// cmdName identifies the command (e.g. "deploy", "build") for history recording.
// The noServicesMsg is shown as a warning if no services are found.
func runPipeline(cmdName string, args []string, opts deploymentOptions, noServicesMsg string) error {
	filterOpts := serviceinfo.FilterOptions{
		Names:       args,
		OnlyChanged: opts.OnlyChanged,
		Since:       opts.Since,
		NoGitIgnore: NoGitIgnore(),
		Env:         opts.Env,
		Provider:    opts.Provider,
	}

	services, err := serviceinfo.FindAndFilterServicesWithOptions(".", filterOpts)
	if err != nil {
		return exitcodes.WithCode(exitcodes.NoServices, errors.Wrap(err, "error finding services"))
	}

	if len(services) == 0 {
		output.Warning(noServicesMsg)
		return nil
	}

	recipes, err := recepie.LoadEmbeddedRecipes()
	if err != nil {
		return exitcodes.WithCode(exitcodes.Config, errors.Wrap(err, "error loading recipes"))
	}

	if len(recipes) == 0 {
		output.Warning("No recipes found")
		return nil
	}

	runner := orchestrator.NewRunner(services, recipes, opts.toRunnerOptions())
	startTime := time.Now()
	runErr := runner.Run()

	// Record history (skip dry-runs)
	if !opts.DryRun {
		recordHistory(cmdName, opts.Tag, runner, time.Since(startTime), runErr)
	}

	if runErr != nil {
		return exitcodes.WithCode(exitcodes.Deploy, runErr)
	}
	return nil
}

// recordHistory saves a pipeline run to the history file.
func recordHistory(cmdName, tag string, runner *orchestrator.Runner, duration time.Duration, runErr error) {
	root, err := path.FindProjectRoot()
	if err != nil || root == "" {
		return
	}

	results := runner.Results()
	services := make([]history.ServiceResult, len(results))
	allSuccess := runErr == nil
	for i, r := range results {
		sr := history.ServiceResult{
			Name:     r.ServiceName,
			Step:     r.StepName,
			Success:  r.Success,
			Duration: r.Duration.Round(time.Millisecond).String(),
		}
		if r.Error != nil {
			sr.Error = r.Error.Error()
		}
		services[i] = sr
	}

	entry := history.NewEntry(cmdName, tag, allSuccess, duration, services)
	_ = history.Record(root, entry)
}

// withJSON wraps a command function that returns structured data.
// In JSON mode: emits the returned value as JSON, suppresses text output.
// In normal mode: ignores the returned value, text output works as usual.
func withJSON(fn func(*cobra.Command, []string) (any, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		result, err := fn(cmd, args)
		if output.IsJSON() {
			if err != nil {
				output.JSON(map[string]any{"success": false, "error": err.Error()})
			} else if result != nil {
				output.JSON(result)
			}
			return err
		}
		return err
	}
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
