package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish [services...]",
		Short: "Publish services (build, push, publish)",
		Long:  "Publish one or more services or all services if none specified. This command will build, push and publish the services to the specified environment.",
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			tag := viper.GetString("tag")

			if len(args) > 0 {
				// publish specific services
				fmt.Printf("publishing services: %v with tag %s\n", args, tag)
				for _, service := range args {
					// Logic here for each service
					fmt.Printf("  publishing service %s\n", service)
				}
			} else {
				// publish all services
				fmt.Printf("publishing all services with tag %s\n", tag)
				// Logic here for all services
			}
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")

	return cmd
}

func init() {
	rootCmd.AddCommand(PublishCmd())
}
