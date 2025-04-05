package cmd

import (
	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func bindFlagsForDeploymentCommands(cmd *cobra.Command) error {
	err := viper.BindPFlag("tag", cmd.Flags().Lookup("tag"))
	if err != nil {
		return errors.Wrap(err, "error binding tag flag")
	}
	err = viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
	if err != nil {
		return errors.Wrap(err, "error binding debug flag")
	}
	err = viper.BindPFlag("timeout", cmd.Flags().Lookup("timeout"))
	if err != nil {
		return errors.Wrap(err, "error binding timeout flag")
	}
	err = viper.BindPFlag("retries", cmd.Flags().Lookup("retries"))
	if err != nil {
		return errors.Wrap(err, "error binding retries flag")
	}
	err = viper.BindPFlag("dry-run", cmd.Flags().Lookup("dry-run"))
	if err != nil {
		return errors.Wrap(err, "error binding dry-run flag")
	}

	return nil
}

func cmdFlagStrings(cmd *cobra.Command) {
	cmd.Flags().StringP("tag", "t", "latest", "Tag for the services")
	cmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
	cmd.Flags().IntP("timeout", "T", 60, "Timeout for the build process in seconds")
	cmd.Flags().IntP("retries", "r", 3, "Number of retries for the build process")
	cmd.Flags().BoolP("dry-run", "D", false, "Perform a dry run without executing the build")
}
