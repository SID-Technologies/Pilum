package cmd

import (
	"fmt"
	"os"

	"github.com/sid-technologies/pilum/lib/output"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFile string

// Output mode flags
var (
	verboseFlag bool
	quietFlag   bool
	jsonFlag    bool
)

// version is set at build time via ldflags:
// go build -ldflags="-X github.com/sid-technologies/pilum/cmd.version=v1.0.0"
var version = "dev"

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "pilum",
	Short: "Cloud-agnostic deployment CLI",
	Long: `Pilum - Define once, deploy anywhere.

A cloud-agnostic deployment CLI that lets you define a service once
and deploy it to any cloud provider (AWS, GCP, Azure).

Define your service in a service.yaml file, specify the target provider,
and Pilum handles the build, containerization, and deployment.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		output.Error(err.Error())
		//nolint: revive // standard practice to use os.Exit in main package
		os.Exit(1)
	}
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	cobra.OnInitialize(initConfig, initOutputMode)
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("pilum {{ .Version }}\n")
	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd.Parent() == nil {
			banner := output.PrintBanner(version)
			fmt.Print(banner)
		}
		defaultHelpFunc(cmd, args)
	})

	// Add persistent flags for output mode
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Stream command output in real-time")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Minimal output (CI-friendly)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON for scripting")

	// Mark flags as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet", "json")
}

// initOutputMode sets the global output mode based on flags.
func initOutputMode() {
	switch {
	case verboseFlag:
		output.SetMode(output.ModeVerbose)
	case quietFlag:
		output.SetMode(output.ModeQuiet)
	case jsonFlag:
		output.SetMode(output.ModeJSON)
	default:
		output.SetMode(output.ModeNormal)
	}
}

func initConfig() {
	configFile = ".pilum.yml"
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		output.Debugf("Using configuration file: %s", viper.ConfigFileUsed())
	}
}
