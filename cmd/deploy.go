package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [services...]",
		Short: "Deploy services (build, publish, push, deploy)",
		Long:  "Deploy one or more services or all services if none specified. This command will build, publish, push and deploy the services to the specified environment.",
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			tag := viper.GetString("tag")

			if len(args) > 0 {
				// deploy specific services
				fmt.Printf("deploying services: %v with tag %s\n", args, tag)
				for _, service := range args {
					// Logic here for each service
					fmt.Printf("  deploying service %s\n", service)
				}
			} else {
				// deploy all services
				fmt.Printf("deploying all services with tag %s\n", tag)
				// Logic here for all services
			}
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")

	return cmd
}

func init() {
	rootCmd.AddCommand(DeployCmd())
}
