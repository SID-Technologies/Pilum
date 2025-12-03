package cmd

import (
	"fmt"

	"github.com/sid-technologies/pilum/ingredients/build"
	"github.com/sid-technologies/pilum/ingredients/docker"
	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
	workerqueue "github.com/sid-technologies/pilum/lib/worker_queue"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [services...]",
		Short: "Push Docker images to registry",
		Long:  "Push Docker images for one or more services to the container registry. Assumes images are already built.",
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
			registry := viper.GetString("registry")

			services, err := serviceinfo.FindAndFilterServices(".", args)
			if err != nil {
				return errors.Wrap(err, "error finding services: %v", err.Error())
			}

			if len(services) == 0 {
				fmt.Println("No services found to push")
				return nil
			}

			fmt.Printf("Pushing %d service(s)...\n", len(services))

			pushQueue := workerqueue.NewWorkQueue(
				func(task *workerqueue.TaskInfo) bool {
					result, err := workerqueue.CommandWorker(task)
					if err != nil {
						fmt.Printf("  Error pushing %s: %v\n", task.ServiceName, err)
						return false
					}
					return result
				},
				0,
			)

			for _, service := range services {
				_, imageName := build.GenerateBuildCommand(service, registry, tag)
				cmd := docker.GenerateDockerPushCommand(imageName)

				if dryRun {
					fmt.Printf("  [dry-run] %s: docker push %s\n", service.Name, imageName)
					continue
				}

				fmt.Printf("  Pushing %s...\n", service.Name)

				task := workerqueue.NewTaskInfo(cmd, "", service.Name, "root", nil, nil, timeout, debug, retries)
				pushQueue.AddTask(task)
			}

			if dryRun {
				fmt.Println("\nDry run complete - no commands executed")
				return nil
			}

			results := pushQueue.Execute()

			successCount := 0
			for _, result := range results {
				if result {
					successCount++
				}
			}

			if successCount != len(results) {
				return errors.New("push failed: %d/%d succeeded", successCount, len(results))
			}

			fmt.Printf("\nPush complete: %d/%d succeeded\n", successCount, len(results))

			return nil
		},
	}

	cmdFlagStrings(cmd)
	cmd.Flags().String("registry", "", "Docker registry prefix (overrides service.yaml)")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

// nolint: gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(PushCmd())
}
