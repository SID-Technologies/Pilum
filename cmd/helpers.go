package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sid-technologies/pilum/lib/ci"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/history"
	"github.com/sid-technologies/pilum/lib/lock"
	"github.com/sid-technologies/pilum/lib/orchestrator"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"
	"github.com/sid-technologies/pilum/lib/recipe"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	"github.com/sid-technologies/pilum/lib/types"
	"github.com/sid-technologies/pilum/lib/webhook"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// deploymentOptions holds parsed flag values for deployment commands.
type deploymentOptions struct {
	Tag          string
	Debug        bool
	Timeout      int
	Retries      int
	DryRun       bool
	Force        bool
	GitHubStatus bool
	MaxWorkers   int
	OnlyTags     []string
	ExcludeTags  []string
	OnlyChanged  bool
	Since        string
	NoDeps       bool
	Env          string
	Provider     string
	RecipePath   string
}

// getDeploymentOptions extracts all standard deployment flags from viper.
func getDeploymentOptions() deploymentOptions {
	return deploymentOptions{
		Tag:          viper.GetString("tag"),
		Debug:        viper.GetBool("debug"),
		Timeout:      viper.GetInt("timeout"),
		Retries:      viper.GetInt("retries"),
		DryRun:       viper.GetBool("dry-run"),
		Force:        viper.GetBool("force"),
		GitHubStatus: viper.GetBool("github-status"),
		MaxWorkers:   viper.GetInt("max-workers"),
		OnlyTags:     parseCommaSeparated(viper.GetString("only-tags")),
		ExcludeTags:  parseCommaSeparated(viper.GetString("exclude-tags")),
		OnlyChanged:  viper.GetBool("only-changed"),
		Since:        viper.GetString("since"),
		NoDeps:       viper.GetBool("no-deps"),
		Env:          viper.GetString("env"),
		Provider:     viper.GetString("provider"),
		RecipePath:   viper.GetString("recipe-path"),
	}
}

// toPipelineOptions converts deploymentOptions to types.PipelineOptions.
func (o deploymentOptions) toPipelineOptions() types.PipelineOptions {
	return types.PipelineOptions{
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
		"force",
		"github-status",
		"max-workers",
		"only-tags",
		"exclude-tags",
		"only-changed",
		"since",
		"no-deps",
		"env",
		"provider",
		"recipe-path",
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
	cmd.Flags().BoolP("force", "f", false, "Force operation (override deployment lock)")
	cmd.Flags().Bool("github-status", false, "Post commit status to GitHub (auto-detected in GitHub Actions)")
	cmd.Flags().String("recipe-path", "", "Load recipes from a directory instead of embedded recipes")

	if includeDryRun {
		cmd.Flags().BoolP("dry-run", "D", false, "Perform a dry run without executing the build")
	}
}

// runPipeline executes the common deployment pipeline: find services → load recipes → run.
// cmdName identifies the command (e.g. "deploy", "build") for history recording.
// The noServicesMsg is shown as a warning if no services are found.
func runPipeline(cmdName string, args []string, opts deploymentOptions, noServicesMsg string) error {
	output.SetDebug(opts.Debug)

	// Normalize service names: support both space-separated and comma-separated inputs.
	// "service-a,service-b" → ["service-a", "service-b"]
	var normalizedArgs []string
	for _, arg := range args {
		for _, name := range strings.Split(arg, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				normalizedArgs = append(normalizedArgs, name)
			}
		}
	}

	filterOpts := serviceinfo.FilterOptions{
		Names:       normalizedArgs,
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
		// If the user explicitly named services but none were found, fail hard.
		// If no names were given (deploy all), warn and succeed.
		if len(normalizedArgs) > 0 {
			return exitcodes.WithCode(exitcodes.NoServices, errors.New(noServicesMsg))
		}
		output.Warning(noServicesMsg)
		return nil
	}

	var recipes []recipe.Info
	if opts.RecipePath != "" {
		recipes, err = recipe.LoadRecipesFromDirectory(opts.RecipePath)
		if err != nil {
			return exitcodes.WithCode(exitcodes.Config, errors.Wrap(err, "error loading recipes from %s", opts.RecipePath))
		}
	} else {
		recipes, err = recipe.LoadEmbeddedRecipes()
		if err != nil {
			return exitcodes.WithCode(exitcodes.Config, errors.Wrap(err, "error loading recipes"))
		}
	}

	if len(recipes) == 0 {
		output.Warning("No recipes found")
		return nil
	}

	// Find project root for lock
	projectRoot, _ := path.FindProjectRoot()

	// Acquire deployment lock (skip for dry-runs)
	if !opts.DryRun && projectRoot != "" {
		serviceNames := make([]string, len(services))
		for i, svc := range services {
			serviceNames[i] = svc.Name
		}
		if lockErr := lock.Acquire(projectRoot, cmdName, serviceNames, opts.Force); lockErr != nil {
			return exitcodes.WithCode(exitcodes.Lock, lockErr)
		}
		defer lock.Release(projectRoot)

		// Handle SIGINT/SIGTERM to release the lock on Ctrl+C or kill.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			lock.Release(projectRoot)
			// Re-raise the signal so the process exits with the correct status.
			signal.Reset(sig)
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(sig)
		}()
		defer signal.Stop(sigCh)
	}

	// GitHub commit status (opt-in via flag or auto-detected in CI)
	var ghEnv *ci.GitHubEnv
	if opts.GitHubStatus {
		ghEnv = ci.DetectGitHub()
	}
	statusContext := fmt.Sprintf("pilum/%s", cmdName)
	if ghEnv != nil {
		_ = ghEnv.PostStatus(ci.StatePending, statusContext,
			fmt.Sprintf("Deploying %d service(s)...", len(services)), "")
	}

	// Load webhook configs from .pilum.yml
	webhookConfigs := loadWebhookConfigs()
	serviceNames := make([]string, len(services))
	for i, svc := range services {
		serviceNames[i] = svc.Name
	}

	// Fire start webhook
	if len(webhookConfigs) > 0 {
		webhook.Send(webhookConfigs, webhook.Payload{
			Event:    webhook.EventStart,
			Command:  cmdName,
			Tag:      opts.Tag,
			Services: serviceNames,
			Success:  true,
			Text:     fmt.Sprintf("Starting %s for %d service(s) with tag %s", cmdName, len(services), opts.Tag),
		})
	}

	pipelineOpts := opts.toPipelineOptions()
	pipeline := orchestrator.NewPipeline(services, recipes, pipelineOpts)
	startTime := time.Now()
	runErr := pipeline.Run()
	duration := time.Since(startTime).Round(time.Second)

	// Post final GitHub status
	if ghEnv != nil {
		if runErr != nil {
			_ = ghEnv.PostStatus(ci.StateFailure, statusContext,
				fmt.Sprintf("Failed after %s", duration), "")
		} else {
			_ = ghEnv.PostStatus(ci.StateSuccess, statusContext,
				fmt.Sprintf("Deployed %d service(s) in %s", len(services), duration), "")
		}
	}

	// Fire completion webhook
	if len(webhookConfigs) > 0 {
		p := webhook.Payload{
			Command:  cmdName,
			Tag:      opts.Tag,
			Services: serviceNames,
			Duration: duration.String(),
		}
		if runErr != nil {
			p.Event = webhook.EventFailure
			p.Success = false
			p.Error = runErr.Error()
			p.Text = fmt.Sprintf("%s failed after %s: %s", cmdName, duration, runErr.Error())
		} else {
			p.Event = webhook.EventSuccess
			p.Success = true
			p.Text = fmt.Sprintf("%s completed for %d service(s) in %s", cmdName, len(services), duration)
		}
		webhook.Send(webhookConfigs, p)
	}

	// Record history (skip dry-runs)
	if !opts.DryRun {
		recordHistory(cmdName, opts.Tag, pipeline, time.Since(startTime), runErr)
	}

	if runErr != nil {
		return exitcodes.WithCode(exitcodes.Deploy, runErr)
	}
	return nil
}

// recordHistory saves a pipeline run to the history file.
func recordHistory(cmdName, tag string, pipeline *orchestrator.Pipeline, duration time.Duration, runErr error) {
	root, err := path.FindProjectRoot()
	if err != nil || root == "" {
		return
	}

	results := pipeline.Results()
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

// loadWebhookConfigs reads the webhooks array from .pilum.yml via Viper.
func loadWebhookConfigs() []webhook.Config {
	var configs []webhook.Config

	raw := viper.Get("webhooks")
	if raw == nil {
		return nil
	}

	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		cfg := webhook.Config{}

		if u, ok := m["url"].(string); ok {
			cfg.URL = u
		}
		if cfg.URL == "" {
			continue
		}

		if events, ok := m["events"].([]any); ok {
			for _, e := range events {
				if s, ok := e.(string); ok {
					cfg.Events = append(cfg.Events, webhook.Event(s))
				}
			}
		}

		configs = append(configs, cfg)
	}

	return configs
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
