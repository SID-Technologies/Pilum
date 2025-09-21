package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/sid-technologies/centurion/lib/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "cli",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

//nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	cobra.OnInitialize(initConfig)
	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Parent() == nil {
			banner := output.PrintBanner(version)
			fmt.Print(banner)
		}
		defaultHelpFunc(cmd, args)
	})
}

func initConfig() {
	configFile = ".cobra-cli-samples.yml"
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using configuration file: ", viper.ConfigFileUsed())
	}
}
