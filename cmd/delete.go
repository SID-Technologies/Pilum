package cmd

import (
	"os"
	"path/filepath"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/output"

	"github.com/spf13/cobra"
)

var deleteBuildsCmd = &cobra.Command{
	Use:   "delete-builds",
	Short: "Delete all builds",
	RunE: func(_ *cobra.Command, _ []string) error {
		output.Info("Deleting all builds...")
		err := deleteBuilds()
		if err != nil {
			return errors.Wrap(err, "error deleting builds")
		}

		output.Success("All builds deleted successfully.")

		return nil
	},
}

func deleteBuilds() error {
	output.Info("Deleting all builds...")
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "dist" {
			output.Dimmed("Removing: %s", path)
			return os.RemoveAll(path)
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "error deleting builds")
	}

	return nil
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(deleteBuildsCmd)
}
