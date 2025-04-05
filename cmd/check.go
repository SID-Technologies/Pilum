package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func CheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check servvice.yaml",
		Short: "Check the configuration of the services",
		Long:  "Check the configuration of the services",
		Run: func(cmd *cobra.Command, args []string) {
			// find services
			// check configuration files
			file := ""
			fmt.Printf("Checking configuration of the services in %s\n", file)

		},
	}

	return cmd
}

func init() {
	rootCmd.AddCommand(CheckCmd())
}
