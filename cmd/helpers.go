package cmd

import (
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func bindFlagsForDeploymentCommands(cmd *cobra.Command) error {
	flagBindings := []string{
		"tag",
		"debug",
		"timeout",
		"retries",
		"dry-run",
		"recipe-path",
		"max-workers",
		"only-tags",
		"exclude-tags",
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

func cmdFlagStrings(cmd *cobra.Command) {
	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().IntP("timeout", "T", 60, "Timeout for the build process in seconds")
	cmd.Flags().IntP("retries", "r", 3, "Number of retries for the build process")
	cmd.Flags().BoolP("dry-run", "D", false, "Perform a dry run without executing the build")
	cmd.Flags().String("recipe-path", "./recepies", "Path to recipe definitions")
	cmd.Flags().Int("max-workers", 0, "Maximum parallel workers (0 = auto)")
	cmd.Flags().String("only-tags", "", "Only run steps with these tags (comma-separated)")
	cmd.Flags().String("exclude-tags", "", "Exclude steps with these tags (comma-separated)")
}

// cmdFlagStringsNoDryRun adds all standard flags except --dry-run (for commands that are always dry-run).
func cmdFlagStringsNoDryRun(cmd *cobra.Command) {
	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().IntP("timeout", "T", 60, "Timeout for the build process in seconds")
	cmd.Flags().IntP("retries", "r", 3, "Number of retries for the build process")
	cmd.Flags().String("recipe-path", "./recepies", "Path to recipe definitions")
	cmd.Flags().Int("max-workers", 0, "Maximum parallel workers (0 = auto)")
	cmd.Flags().String("only-tags", "", "Only run steps with these tags (comma-separated)")
	cmd.Flags().String("exclude-tags", "", "Exclude steps with these tags (comma-separated)")
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
