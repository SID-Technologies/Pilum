package cmd

import (
	"path/filepath"

	ingredients "github.com/sid-technologies/centurion/_ingredients"
	"github.com/sid-technologies/centurion/lib/configs"
	"github.com/sid-technologies/centurion/lib/errors"
	"github.com/sid-technologies/centurion/lib/flags"
	"github.com/sid-technologies/centurion/lib/output"
	"github.com/sid-technologies/centurion/lib/types"
	"github.com/sid-technologies/centurion/lib/utils"
	"github.com/sid-technologies/centurion/lib/writer"
	"github.com/spf13/cobra"
)

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "add [services...]",
		Aliases:            []string{"a"},
		Short:              "Add services",
		Long:               "Add one or more services or all services if none specified",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				listServicesCmd.Run(cmd, args)
				return nil
			}

			templateName := args[0]
			cl, err := configs.NewClient()
			if err != nil {
				return errors.Wrap(err, "error creating configs client: %v")
			}

			config, exists := cl.Registry.Get(templateName)
			if !exists {
				return errors.New("service %s not found. Run 'centurion list-services' to see available services", templateName)
			}

			for _, arg := range args {
				if arg == "-h" || arg == "--help" {
					output.PrintAddHelp(config)
					return nil
				}
			}

			pathFlag := types.FlagArg{
				Name:        "path",
				Flag:        "--path",
				Type:        "string",
				Default:     ".",
				Required:    false,
				Description: "Path to add the service to",
			}

			config.Options = append(config.Options, pathFlag)

			options, err := flags.ParseArgs(args[1:], config.Options)
			if err != nil {
				return errors.Wrap(err, "error parsing args: %v")
			}

			path, ok := options["path"].(string)
			if !ok {
				path = "."
			}

			err = handleAdd(config, path)
			if err != nil {
				return errors.Wrap(err, "error handling add: %v")
			}

			output.PrintNextSteps(config)
			return nil
		},
	}

	cmdFlagStrings(cmd)

	return cmd
}

func handleAdd(config types.Config, outputPath string) error {
	basePath, err := utils.FindProjectRoot()
	if err != nil {
		return errors.Wrap(err, "error finding project root: %v")
	}

	ingredientsPath, err := ingredients.GetPath()
	if err != nil {
		return errors.Wrap(err, "error getting ingredients path: %v")
	}

	outputPath = filepath.Join(basePath, outputPath)
	writer := writer.NewFileWriter(
		ingredientsPath,
		outputPath,
	)

	files := make([]struct{ Source, Output string }, len(config.Files))
	for i, file := range config.Files {
		files[i] = struct{ Source, Output string }{
			Source: file.Path,
			Output: file.OutputPath,
		}
	}

	err = writer.ReadAndWriteFiles(files)
	if err != nil {
		return errors.Wrap(err, "error writing files: %v")
	}

	return nil
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(addCmd())
}
