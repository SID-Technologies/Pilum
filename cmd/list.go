package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var listServicesCmd = &cobra.Command{
	Use:   "list-services",
	Short: "List all services with their configuration",
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("Listing all services...")
		// if err := executor.ExecutePythonScript("-m", "scripts.list_services"); err != nil {
		// 	fmt.Printf("Error listing services: %v\n", err)
		// }
	},
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(listServicesCmd)
}
