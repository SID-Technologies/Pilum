package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [services...]",
		Short: "Push services (pushes to registry), runs build and publish",
		Long:  "push one or more services or all services if none specified",
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
		},
		Run: func(cmd *cobra.Command, args []string) {
			tag := viper.GetString("tag")

			if len(args) > 0 {
				// push specific services
				fmt.Printf("pushing services: %v with tag %s\n", args, tag)
				for _, service := range args {
					// Logic here for each service
					fmt.Printf("  pushing service %s\n", service)
				}
			} else {
				// push all services
				fmt.Printf("pushing all services with tag %s\n", tag)
				// Logic here for all services
			}
		},
	}

	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")

	return cmd
}

func init() {
	rootCmd.AddCommand(PushCmd())
}
