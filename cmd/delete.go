package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var deleteBuildsCmd = &cobra.Command{
	Use:   "delete-builds",
	Short: "Delete all builds",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deleting all builds...")
		deleteBuilds()
	},
}

func deleteBuilds() {
	fmt.Println("Deleting all builds...")
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "dist" {
			fmt.Printf("Removing: %s\n", path)
			return os.RemoveAll(path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error deleting builds: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(deleteBuildsCmd)
}
