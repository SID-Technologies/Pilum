package cmd

import (
	"fmt"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/history"
	"github.com/sid-technologies/pilum/lib/output"
	"github.com/sid-technologies/pilum/lib/path"

	"github.com/spf13/cobra"
)

func HistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "history",
		Aliases: []string{"hist"},
		Short:   "Show deployment history",
		Long:    "Show the history of pipeline runs (deploy, build, publish, push) recorded in this project.",
		RunE: withJSON(func(cmd *cobra.Command, _ []string) (any, error) {
			root, err := path.FindProjectRoot()
			if err != nil {
				return nil, exitcodes.WithCode(exitcodes.NoServices,
					errors.Wrap(err, "error finding project root"))
			}

			entries, err := history.Load(root)
			if err != nil {
				return nil, errors.Wrap(err, "error loading history")
			}

			if len(entries) == 0 {
				output.Info("No deployment history found")
				return entries, nil
			}

			limit, _ := cmd.Flags().GetInt("limit")
			service, _ := cmd.Flags().GetString("service")

			// Filter by service name if specified
			if service != "" {
				entries = filterByService(entries, service)
			}

			if len(entries) == 0 {
				output.Info("No history entries match the filter")
				return entries, nil
			}

			// Apply limit
			if limit > 0 && limit < len(entries) {
				entries = entries[:limit]
			}

			// Text output (suppressed in JSON mode)
			printHistory(entries)

			return entries, nil
		}),
	}

	cmd.Flags().IntP("limit", "n", 10, "Number of entries to show")
	cmd.Flags().StringP("service", "s", "", "Filter by service name")

	return cmd
}

func filterByService(entries []history.Entry, name string) []history.Entry {
	var filtered []history.Entry
	for _, e := range entries {
		for _, svc := range e.Services {
			if svc.Name == name {
				filtered = append(filtered, e)
				break
			}
		}
	}
	return filtered
}

func printHistory(entries []history.Entry) {
	output.Header("Deployment History")

	for _, e := range entries {
		status := output.SuccessColor + output.SymbolSuccess
		if !e.Success {
			status = output.ErrorColor + output.SymbolFailure
		}

		// Count successes
		total := len(e.Services)
		succeeded := 0
		for _, svc := range e.Services {
			if svc.Success {
				succeeded++
			}
		}

		ts := e.Timestamp.Local().Format("2006-01-02 15:04")

		fmt.Printf("  %s%s%s  %s%s%s  %-8s %-10s %s %d/%d%s  %s(%s)%s\n",
			output.Muted, e.ID, output.Reset,
			output.Muted, ts, output.Reset,
			e.Command,
			e.Tag,
			status, succeeded, total, output.Reset,
			output.Muted, e.Duration, output.Reset,
		)
	}
	fmt.Println()
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(HistoryCmd())
}
