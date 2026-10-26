package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/sid-technologies/pilum/lib/telemetry"

	"github.com/spf13/cobra"
)

// TelemetryCmd shows and changes the opt-in telemetry setting.
func TelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Show or change whether Pilum shares anonymous build telemetry",
		Long: `Pilum can share anonymous build telemetry: which recipe and step ran,
whether it succeeded and how long it took. It is off unless you turn it on,
and never includes service names, project IDs, paths or error messages.

DO_NOT_TRACK=1 always turns it off; PILUM_TELEMETRY=1 or 0 overrides the
saved setting. Details: https://pilum.dev/docs/reference/telemetry/`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show whether telemetry is on and what decided it",
		RunE: func(cmd *cobra.Command, _ []string) error {
			on, src := telemetry.Enabled()
			state := "off"
			if on {
				state = "on"
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Telemetry: %s (set by %s)\n", state, src)
			if id := telemetry.Load().InstallID; id != "" {
				_, _ = fmt.Fprintf(out, "Install ID: %s\n", id)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Share anonymous build telemetry",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := telemetry.Enable(); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Telemetry is on. `pilum telemetry last` shows what was last sent.")
			warnIfOverridden(cmd)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Stop sharing telemetry and delete the install ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := telemetry.Disable(); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Telemetry is off and the install ID is deleted.")
			warnIfOverridden(cmd)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "last",
		Short: "Print the last telemetry payload sent from this machine",
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := telemetry.Last()
			if err != nil {
				// Nothing sent yet is a normal state, not a failure.
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No telemetry has been sent from this machine.")
				return nil //nolint:nilerr // see above
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	})
	return cmd
}

// warnIfOverridden says so when an env var outranks the saved setting.
func warnIfOverridden(cmd *cobra.Command) {
	if _, src := telemetry.Enabled(); src == telemetry.SourceDoNotTrack || src == telemetry.SourceEnv {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Note: %s is set in this shell and takes precedence.\n", src)
	}
}

// askTelemetryOnFirstRun shows the opt-in prompt on the first interactive run
// of any command. It is skipped in CI, without a terminal, for --json and
// --quiet, for help, version and completion, and once answered.
func askTelemetryOnFirstRun() {
	args := os.Args[1:]
	sub, _, err := rootCmd.Find(args)
	if err != nil || sub == nil || sub == rootCmd {
		return
	}
	has := func(flags ...string) bool {
		return slices.ContainsFunc(args, func(a string) bool { return slices.Contains(flags, a) })
	}
	ctx := telemetry.PromptContext{
		Command:     sub.Name(),
		Interactive: isInteractive() && stdoutIsTerminal(),
		Quiet:       has("--json", "--quiet", "-q"),
		HelpOrVer:   has("-h", "--help", "--version") || (sub.Parent() != nil && sub.Parent().Name() == "telemetry"),
	}
	if telemetry.ShouldPrompt(ctx) {
		telemetry.Ask(os.Stdin, os.Stdout)
	}
}

func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(TelemetryCmd())
}
