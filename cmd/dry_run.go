package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run",
		Short: "Perform a dry run of one or more services",
		Long:  "Preform a dry run one or more services or all services if none specified for build, publish, push and deploy operations",
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			tag := viper.GetString("tag")

			if len(args) > 0 {
				// Build specific services
				fmt.Printf("Building services: %v with tag %s\n", args, tag)
				for _, service := range args {
					// Logic here for each service
					fmt.Printf("  Building service %s\n", service)
				}
			} else {
				// Build all services
				fmt.Printf("Building all services with tag %s\n", tag)
				// Logic here for all services
			}
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")

	return cmd
}

func init() {
	rootCmd.AddCommand(DryRunCmd())
}
