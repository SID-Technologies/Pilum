package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	// "your-module-name/internal/executor"
)

var listServicesCmd = &cobra.Command{
	Use:   "list-services",
	Short: "List all services with their configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing all services...")
		// if err := executor.ExecutePythonScript("-m", "scripts.list_services"); err != nil {
		// 	fmt.Printf("Error listing services: %v\n", err)
		// }
	},
}

func init() {
	rootCmd.AddCommand(listServicesCmd)
}