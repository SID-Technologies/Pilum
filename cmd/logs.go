package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/query"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func LogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logs <service>",
		Aliases: []string{"log"},
		Short:   "Show logs for a deployed service",
		Long:    "Read logs for a deployed cloud service. Use --follow to stream logs in real-time.",
		Args:    cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			for _, flag := range []string{"lines", "follow", "timeout", "debug", "env"} {
				if f := cmd.Flags().Lookup(flag); f != nil {
					if err := viper.BindPFlag(flag, f); err != nil {
						return errors.Wrap(err, "error binding %s flag", flag)
					}
				}
			}
			return nil
		},
		RunE: withJSON(func(_ *cobra.Command, args []string) (any, error) {
			return runLogs(args[0])
		}),
	}

	cmd.Flags().IntP("lines", "n", 50, "Number of log lines to show")
	cmd.Flags().BoolP("follow", "f", false, "Stream logs in real-time")
	cmd.Flags().IntP("timeout", "T", 30, "Query timeout in seconds")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().StringP("env", "e", "", "Environment to apply (merges overrides from environments block)")

	return cmd
}

func runLogs(serviceName string) (any, error) {
	lines := viper.GetInt("lines")
	follow := viper.GetBool("follow")
	timeout := viper.GetInt("timeout")
	debug := viper.GetBool("debug")
	env := viper.GetString("env")

	filterOpts := serviceinfo.FilterOptions{
		Names:       []string{serviceName},
		NoGitIgnore: NoGitIgnore(),
		Env:         env,
	}

	services, err := serviceinfo.FindAndFilterServicesWithOptions(".", filterOpts)
	if err != nil {
		return nil, exitcodes.WithCode(exitcodes.NoServices, errors.Wrap(err, "error finding services"))
	}

	if len(services) == 0 {
		return nil, exitcodes.WithCode(exitcodes.NoServices,
			errors.New("service '%s' not found", serviceName))
	}

	svc := services[0]

	if !query.IsCloudProvider(svc.Provider) {
		return nil, exitcodes.WithCode(exitcodes.InvalidArgs,
			errors.New("logs not available for provider '%s'", svc.Provider))
	}

	logsCmd, err := query.GenerateLogsCommand(svc, lines, follow)
	if err != nil {
		return nil, err
	}

	if debug {
		output.Debugf("Logs command: %v", logsCmd)
	}

	// Follow mode: pipe directly to terminal
	if follow {
		return nil, runFollowMode(logsCmd)
	}

	// Snapshot mode: capture and display
	taskInfo := workerqueue.NewTaskInfo(
		logsCmd,
		"",
		svc.DisplayName(),
		"root",
		nil,
		nil,
		timeout,
		debug,
		0,
	)

	logOutput, captureErr := workerqueue.CaptureWorker(taskInfo)
	if captureErr != nil {
		return nil, captureErr
	}

	fmt.Print(logOutput)

	// Return structured data for JSON mode
	type jsonLogs struct {
		Service string `json:"service"`
		Output  string `json:"output"`
	}
	return jsonLogs{Service: svc.DisplayName(), Output: logOutput}, nil
}

func runFollowMode(args []string) error {
	if len(args) < 1 {
		return errors.New("empty command")
	}

	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // Command comes from trusted dispatch
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run() //nolint:wrapcheck // Pass through exit code directly
}

//nolint:gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(LogsCmd())
}
