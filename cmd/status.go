package cmd

import (
	"fmt"
	"sync"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/query"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func StatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status [services...]",
		Aliases: []string{"st"},
		Short:   "Show status of deployed services",
		Long:    "Query the status of one or more deployed cloud services. Non-cloud providers (npm, homebrew) are skipped automatically.",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			for _, flag := range []string{"timeout", "debug", "env", "provider"} {
				if f := cmd.Flags().Lookup(flag); f != nil {
					if err := viper.BindPFlag(flag, f); err != nil {
						return errors.Wrap(err, "error binding %s flag", flag)
					}
				}
			}
			return nil
		},
		RunE: withJSON(func(_ *cobra.Command, args []string) (any, error) {
			return runStatus(args)
		}),
	}

	cmd.Flags().IntP("timeout", "T", 30, "Query timeout in seconds")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")
	cmd.Flags().String("provider", "", "Filter services by provider (e.g., gcp, aws, azure)")

	return cmd
}

func runStatus(args []string) (any, error) {
	timeout := viper.GetInt("timeout")
	debug := viper.GetBool("debug")
	env := viper.GetString("env")
	provider := viper.GetString("provider")

	filterOpts := serviceinfo.FilterOptions{
		Names:       args,
		NoGitIgnore: NoGitIgnore(),
		Env:         env,
		Provider:    provider,
	}

	services, err := serviceinfo.FindAndFilterServicesWithOptions(".", filterOpts)
	if err != nil {
		return nil, exitcodes.WithCode(exitcodes.NoServices, errors.Wrap(err, "error finding services"))
	}

	if len(services) == 0 {
		output.Warning("No services found")
		return nil, nil
	}

	// Filter to cloud-only providers
	cloudServices := query.FilterCloudServices(services)
	if len(cloudServices) == 0 {
		output.Warning("No cloud services found (non-cloud providers like npm, homebrew are skipped)")
		return nil, nil
	}

	skipped := len(services) - len(cloudServices)
	if skipped > 0 {
		output.Debugf("Skipped %d non-cloud service(s)", skipped)
	}

	// Query all services in parallel
	type statusResult struct {
		status query.ServiceStatus
		err    error
	}

	results := make([]statusResult, len(cloudServices))
	var wg sync.WaitGroup

	for i, svc := range cloudServices {
		wg.Add(1)
		go func(idx int, s serviceinfo.ServiceInfo) {
			defer wg.Done()

			cmd, cmdErr := query.GenerateStatusCommand(s)
			if cmdErr != nil {
				results[idx] = statusResult{err: cmdErr}
				return
			}

			taskInfo := workers.NewTaskInfo(
				cmd,
				"",
				s.DisplayName(),
				"root",
				nil,
				nil,
				timeout,
				debug,
				0,
			)

			jsonOutput, captureErr := workers.CaptureWorker(taskInfo)
			if captureErr != nil {
				results[idx] = statusResult{
					status: query.ServiceStatus{
						Name:     s.DisplayName(),
						Provider: s.Provider,
						Region:   s.Region,
						Status:   "Error",
					},
					err: captureErr,
				}
				return
			}

			results[idx] = statusResult{
				status: query.ParseStatusResponse(jsonOutput, s),
			}
		}(i, svc)
	}

	wg.Wait()

	// Collect statuses for output
	statuses := make([]query.ServiceStatus, 0, len(results))
	for _, r := range results {
		statuses = append(statuses, r.status)
	}

	// Text output
	printStatusTable(statuses)

	return statuses, nil
}

func printStatusTable(statuses []query.ServiceStatus) {
	output.Header("Service Status")

	for _, s := range statuses {
		var symbol, color string

		switch s.Status {
		case "Ready", "Running", "Succeeded", "Active":
			symbol = output.SymbolSuccess
			color = output.SuccessColor
		case "Error", "NotReady", "Failed":
			symbol = output.SymbolFailure
			color = output.ErrorColor
		case "Unknown":
			symbol = output.SymbolWarning
			color = output.WarningColor
		default:
			symbol = output.SymbolInfo
			color = output.InfoColor
		}

		image := s.Image
		if image == "" {
			image = "—"
		}

		fmt.Printf("  %s%s%s  %-20s %-12s %-18s %s\n",
			color, symbol, output.Reset,
			s.Name, s.Status, s.Region, image)

		if s.URL != "" {
			fmt.Printf("      %s%s%s\n", output.Muted, s.URL, output.Reset)
		}
	}
	fmt.Println()
}

//nolint:gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(StatusCmd())
}
