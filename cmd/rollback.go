package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/history"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"
	"github.com/sid-technologies/pilum/lib/rollback"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rollbackResult is one service's outcome, for JSON output and history.
type rollbackResult struct {
	rollback.Plan
	Success  bool   `json:"success"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

func RollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <services...>",
		Short: "Roll services back to their previous version",
		Long: `Roll one or more deployed services back to the version before the one
currently deployed. Run it again to step back further.

Cloud Run services: moves 100% of traffic to the previous healthy revision.
No rebuild, and that revision's full config (env, CPU, secrets) comes back
with it. The next 'pilum deploy' sends traffic to the new revision again.

Cloud Run jobs: points the job at the image of its previous execution,
keeping the job's current config.

The previous version is read from the cloud provider, so this works from any
machine, including CI. Asks for confirmation unless --yes is passed.`,
		Example: `  pilum rollback api                      # Previous revision
  pilum rollback api --to api-00041-xyz   # Specific revision
  pilum rollback nightly-job --to v1.4.2  # Job: image tag
  pilum rollback api worker --dry-run     # Show the plan only
  pilum rollback api --yes                # No prompt (required in CI)`,
		Args: cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			for _, flag := range []string{"to", "yes", "dry-run", "force", "timeout", "debug", "env"} {
				if f := cmd.Flags().Lookup(flag); f != nil {
					if err := viper.BindPFlag(flag, f); err != nil {
						return errors.Wrap(err, "error binding %s flag", flag)
					}
				}
			}
			return nil
		},
		RunE: withJSON(func(_ *cobra.Command, args []string) (any, error) {
			return runRollback(args)
		}),
	}

	cmd.Flags().String("to", "", "Target revision (services) or image tag/reference (jobs); default: previous version")
	cmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt (required when not interactive)")
	cmd.Flags().BoolP("dry-run", "D", false, "Show the rollback plan without changing anything")
	cmd.Flags().BoolP("force", "f", false, "Force operation (override deployment lock)")
	cmd.Flags().IntP("timeout", "T", 60, "Timeout per command in seconds")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")

	return cmd
}

func runRollback(args []string) (any, error) {
	to := viper.GetString("to")
	dryRun := viper.GetBool("dry-run")
	timeout := viper.GetInt("timeout")
	debug := viper.GetBool("debug")
	output.SetDebug(debug)

	var names []string
	for _, arg := range args {
		for _, name := range strings.Split(arg, ",") {
			if name = strings.TrimSpace(name); name != "" {
				names = append(names, name)
			}
		}
	}

	services, err := serviceinfo.FindAndFilterServicesWithOptions(".", serviceinfo.FilterOptions{
		Names:       names,
		NoGitIgnore: NoGitIgnore(),
		Env:         viper.GetString("env"),
	})
	if err != nil {
		return nil, exitcodes.WithCode(exitcodes.NoServices, errors.Wrap(err, "error finding services"))
	}
	if len(services) == 0 {
		return nil, exitcodes.WithCode(exitcodes.NoServices, errors.New("no services found matching: %s", strings.Join(names, ", ")))
	}

	// Revision names are per service and per region, so an explicit target
	// only makes sense for a single instance.
	if to != "" && len(services) > 1 {
		return nil, exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("--to needs exactly one service instance, got %d (multi-region services roll back per region)", len(services)))
	}

	for _, svc := range services {
		if _, ok := rollback.KindFor(svc); !ok {
			return nil, exitcodes.WithCode(exitcodes.InvalidArgs,
				errors.New("rollback is not supported for %s (recipe %s); supported: Cloud Run services and jobs",
					svc.DisplayName(), svc.RecipeKey()))
		}
	}

	plans, err := planRollbacks(services, to, timeout, debug)
	if err != nil {
		return nil, exitcodes.WithCode(exitcodes.Deploy, err)
	}

	printRollbackPlan(plans)

	if dryRun {
		for _, p := range plans {
			output.Dimmed("  %s: %s", p.Service, output.FormatCommand(p.Command))
		}
		return plans, nil
	}

	if !viper.GetBool("yes") {
		if confirmErr := confirmRollback(len(plans)); confirmErr != nil {
			return nil, exitcodes.WithCode(exitcodes.InvalidArgs, confirmErr)
		}
	}

	if projectRoot, _ := path.FindProjectRoot(); projectRoot != "" {
		release, lockErr := acquireLock(projectRoot, "rollback", services, viper.GetBool("force"))
		if lockErr != nil {
			return nil, lockErr
		}
		defer release()
	}

	start := time.Now()
	results := executeRollbacks(plans, timeout, debug)
	recordRollbackHistory(results, time.Since(start))

	var failed []string
	for _, r := range results {
		if r.Success {
			output.Success("%s → %s", r.Service, r.To)
		} else {
			failed = append(failed, r.Service)
		}
	}
	if len(failed) > 0 {
		return results, exitcodes.WithCode(exitcodes.Deploy,
			errors.New("rollback failed for: %s", strings.Join(failed, ", ")))
	}
	return results, nil
}

// planRollbacks queries every service in parallel. Any planning failure aborts
// the whole rollback before anything changes, so a multi-service rollback
// never ends up half applied because one service had no earlier revision.
func planRollbacks(services []serviceinfo.ServiceInfo, to string, timeout int, debug bool) ([]rollback.Plan, error) {
	plans := make([]rollback.Plan, len(services))
	errs := make([]error, len(services))

	var wg sync.WaitGroup
	for i, svc := range services {
		wg.Add(1)
		go func(idx int, s serviceinfo.ServiceInfo) {
			defer wg.Done()
			run := func(cmd []string) (string, error) {
				return workers.CaptureWorker(workers.NewTaskInfo(cmd, "", s.DisplayName(), "root", nil, nil, timeout, debug, 1))
			}
			plans[idx], errs[idx] = rollback.NewPlan(s, to, run)
		}(i, svc)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return plans, nil
}

func executeRollbacks(plans []rollback.Plan, timeout int, debug bool) []rollbackResult {
	results := make([]rollbackResult, len(plans))

	var wg sync.WaitGroup
	for i, p := range plans {
		wg.Add(1)
		go func(idx int, plan rollback.Plan) {
			defer wg.Done()
			start := time.Now()
			ok, err := workers.CommandWorker(workers.NewTaskInfo(plan.Command, "", plan.Service, "root", nil, nil, timeout, debug, 1))
			results[idx] = rollbackResult{
				Plan:     plan,
				Success:  ok,
				Duration: time.Since(start).Round(time.Millisecond).String(),
			}
			if err != nil {
				results[idx].Error = err.Error()
			}
		}(i, p)
	}
	wg.Wait()

	return results
}

func printRollbackPlan(plans []rollback.Plan) {
	if output.IsJSON() {
		return // plans are emitted as the JSON result
	}
	output.Header("Rollback plan")
	for _, p := range plans {
		fmt.Printf("  %-24s %s%s%s → %s\n", p.Service, output.Muted, p.From, output.Reset, p.To)
		if p.Kind == rollback.KindService && p.ToImage != "" {
			fmt.Printf("  %-24s %s%s%s\n", "", output.Muted, p.ToImage, output.Reset)
		}
	}
	fmt.Println()
}

// confirmRollback asks before touching production. Without a terminal there is
// nobody to ask, so it refuses rather than guessing.
func confirmRollback(count int) error {
	if output.IsJSON() || !isInteractive() {
		return errors.New("refusing to roll back without confirmation; pass --yes to proceed non-interactively")
	}
	answer, err := prompt(bufio.NewReader(os.Stdin), fmt.Sprintf("Roll back %d service(s)? (y/N)", count), "n")
	if err != nil {
		// EOF: stdin closed before an answer, so treat it as "no".
		return errors.New("rollback canceled: no confirmation received (pass --yes to skip the prompt)")
	}
	if a := strings.ToLower(answer); a != "y" && a != "yes" {
		return errors.New("rollback canceled")
	}
	return nil
}

// ciEnvVars mark a CI runner, where nobody is there to answer a prompt.
var ciEnvVars = []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_URL", "BUILDKITE"}

// isInteractive reports whether stdin is a terminal a person could answer on.
// A character device alone isn't enough: /dev/null is one too, and it is what
// CI runners and `cmd </dev/null` hand us.
func isInteractive() bool {
	for _, v := range ciEnvVars {
		if os.Getenv(v) != "" {
			return false
		}
	}
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if devNull, err := os.Stat(os.DevNull); err == nil && os.SameFile(info, devNull) {
		return false
	}
	return true
}

func recordRollbackHistory(results []rollbackResult, duration time.Duration) {
	root, err := path.FindProjectRoot()
	if err != nil || root == "" {
		return
	}

	services := make([]history.ServiceResult, len(results))
	allSuccess := true
	for i, r := range results {
		services[i] = history.ServiceResult{
			Name:     r.Service,
			Step:     fmt.Sprintf("rollback %s → %s", r.From, r.To),
			Success:  r.Success,
			Duration: r.Duration,
			Error:    r.Error,
		}
		allSuccess = allSuccess && r.Success
	}

	if recErr := history.Record(root, history.NewEntry("rollback", "", allSuccess, duration, services)); recErr != nil {
		output.Debugf("Failed to record history: %v", recErr)
	}
}

//nolint:gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(RollbackCmd())
}
