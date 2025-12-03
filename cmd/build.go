package cmd

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func BuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [services...]",
		Short: "Build services",
		Long:  "Build one or more services or all services if none specified",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			err := bindFlagsForDeploymentCommands(cmd)
			if err != nil {
				return errors.Wrap(err, "error binding flags for deployment commands: %v")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			tag := viper.GetString("tag")
			debug := viper.GetBool("debug")
			timeout := viper.GetInt("timeout")
			retries := viper.GetInt("retries")
			dryRun := viper.GetBool("dry-run")

			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services: %v", err.Error())
			}

			if len(services) == 0 {
				fmt.Println("No services found to build")
				return nil
			}

			fmt.Printf("Building %d service(s)...\n", len(services))

			// Create work queue for parallel execution
			buildQueue := workerqueue.NewWorkQueue(
				func(task *workerqueue.TaskInfo) bool {
					result, err := workerqueue.CommandWorker(task)
					if err != nil {
						fmt.Printf("Error building %s: %v\n", task.ServiceName, err)
						return false
					}
					return result
				},
				0, // Use default worker count (CPU/2)
			)

			// Generate build commands and add to queue
			for _, service := range services {
				cmd, _ := build.GenerateBuildCommand(service, "", tag)
				if cmd == nil {
					fmt.Printf("  Skipping %s: no build command configured\n", service.Name)
					continue
				}

				if dryRun {
					cmdStr := build.GenerateBuildCommandString(service)
					fmt.Printf("  [dry-run] %s: %s\n", service.Name, cmdStr)
					continue
				}

				fmt.Printf("  Queuing build for %s\n", service.Name)

				// Convert service env vars to map
				envVars := make(map[string]string)
				for _, ev := range service.BuildConfig.EnvVars {
					envVars[ev.Name] = ev.Value
				}

				task := workerqueue.NewTaskInfo(
					cmd,
					service.Path,
					service.Name,
					"service_dir",
					envVars,
					nil,
					timeout,
					debug,
					retries,
				)

				buildQueue.AddTask(task)
			}

			if dryRun {
				fmt.Println("\nDry run complete - no commands executed")
				return nil
			}

			// Execute all build tasks
			results := buildQueue.Execute()

			// Check results
			successCount := 0
			for _, result := range results {
				if result {
					successCount++
				}
			}

			if successCount != len(results) {
				return errors.New("build failed: %d/%d succeeded", successCount, len(results))
			}

			fmt.Printf("\nBuild complete: %d/%d succeeded\n", successCount, len(results))

			return nil
		},
	}

	cmdFlagStrings(cmd)

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(BuildCmd())
}
