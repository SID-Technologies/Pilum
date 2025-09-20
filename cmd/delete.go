package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/spf13/cobra"
)

var deleteBuildsCmd = &cobra.Command{
	Use:   "delete-builds",
	Short: "Delete all builds",
	RunE: func(_ *cobra.Command, _ []string) error {
		log.Println("Deleting all builds...")
		err := deleteBuilds()
		if err != nil {
			return errors.Wrap(err, "error deleting builds")
		}

		log.Println("All builds deleted successfully.")

		return nil
	},
}

func deleteBuilds() error {
	log.Println("Deleting all builds...")
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "dist" {
			log.Printf("Removing: %s\n", path)
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
