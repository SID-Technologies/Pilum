package cmd

import (
	"time"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/ecsdeploy"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/orchestrator/workers"
	"github.com/sid-technologies/pilum/lib/output"

	"github.com/spf13/cobra"
)

// ecsPollInterval is how often ecs-wait checks the rollout.
const ecsPollInterval = 10 * time.Second

// ECSDeployCmd is the "deploy to ecs" step of the aws-ecs recipe: it
// registers the service's next task definition and points the service at
// it. Hidden: it's plumbing for the recipe, not something to run by hand.
func ECSDeployCmd() *cobra.Command {
	var req ecsdeploy.Request
	var timeout int

	cmd := &cobra.Command{
		Use:    "ecs-deploy",
		Short:  "Register an ECS service's next task definition with a new image and deploy it",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Only what the service sets is replaced; the rest stays as the
			// current task definition has it.
			req.SetEnv = cmd.Flags().Changed("env")
			req.SetSecrets = cmd.Flags().Changed("secret")

			result, err := ecsdeploy.Deploy(req, awsRunner(req.Service, timeout))
			if err != nil {
				return errors.Wrap(err, "deploying to ECS")
			}
			if result.Registered {
				output.Success("Registered %s (was %s)", ecsdeploy.Family(result.TaskDefinition), ecsdeploy.Family(result.Previous))
			} else {
				output.Info("%s already runs this image and config; deploying it as is", ecsdeploy.Family(result.TaskDefinition))
			}
			return nil
		},
	}

	addAWSTargetFlags(cmd, &req.Region, &req.Profile)
	cmd.Flags().StringVar(&req.Cluster, "cluster", "", "ECS cluster name or ARN")
	cmd.Flags().StringVar(&req.Service, "service", "", "ECS service name")
	cmd.Flags().StringVar(&req.Image, "image", "", "Image the container runs")
	cmd.Flags().StringVar(&req.Container, "container", "", "Container that runs the image (default: the only one, or the one named like the service)")
	cmd.Flags().StringVar(&req.CPU, "cpu", "", "Task CPU units (default: keep)")
	cmd.Flags().StringVar(&req.Memory, "memory", "", "Task memory in MiB (default: keep)")
	cmd.Flags().IntVar(&req.DesiredCount, "desired-count", -1, "Task count (default: keep)")
	cmd.Flags().StringArrayVar(&req.Env, "env", nil, "KEY=VALUE env var, repeatable; replaces the container's environment")
	cmd.Flags().StringArrayVar(&req.Secrets, "secret", nil, "NAME=secret name or ARN, repeatable; replaces the container's secrets")
	cmd.Flags().IntVar(&timeout, "timeout", 60, "Timeout per aws command in seconds")

	return cmd
}

// ECSWaitCmd is the "wait for ecs deployment" step: it blocks until the
// service runs the new image, and fails if the rollout fails or is rolled
// back.
func ECSWaitCmd() *cobra.Command {
	var req ecsdeploy.WaitRequest
	var timeout, waitTimeout int

	cmd := &cobra.Command{
		Use:    "ecs-wait",
		Short:  "Wait for an ECS service to finish deploying an image",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			req.Timeout = time.Duration(waitTimeout) * time.Second
			req.Interval = ecsPollInterval
			clock := ecsdeploy.Clock{Now: time.Now, Sleep: time.Sleep}
			progress := func(status string) { output.Info("%s", status) }

			run := awsRunner(req.Service, timeout)
			err := ecsdeploy.Wait(req, run, clock, progress)
			var rollout *ecsdeploy.RolloutError
			if errors.As(err, &rollout) && rollout.TaskDefinition != "" {
				markFailed(req, rollout.TaskDefinition, run)
			}
			if err != nil {
				return err
			}
			output.Success("%s runs %s", req.Service, req.Image)
			return nil
		},
	}

	addAWSTargetFlags(cmd, &req.Region, &req.Profile)
	cmd.Flags().StringVar(&req.Cluster, "cluster", "", "ECS cluster name or ARN")
	cmd.Flags().StringVar(&req.Service, "service", "", "ECS service name")
	cmd.Flags().StringVar(&req.Image, "image", "", "Image the service should end up running")
	cmd.Flags().IntVar(&waitTimeout, "wait-timeout", 600, "Seconds to wait for the rollout")
	cmd.Flags().IntVar(&timeout, "timeout", 60, "Timeout per aws command in seconds")

	return cmd
}

// markFailed tags a revision whose rollout failed, so `pilum rollback`
// doesn't pick it. Best effort: the deploy has failed either way.
func markFailed(req ecsdeploy.WaitRequest, taskDefinition string, run func([]string) (string, error)) {
	_, err := run(aws.TagFailedRolloutCommand(req.Target, taskDefinition))
	if err != nil {
		output.Warning("couldn't mark %s as failed (pilum rollback may pick it): %v", ecsdeploy.Family(taskDefinition), err)
		return
	}
	output.Info("Marked %s as failed; pilum rollback skips it", ecsdeploy.Family(taskDefinition))
}

func addAWSTargetFlags(cmd *cobra.Command, region, profile *string) {
	cmd.Flags().StringVar(region, "region", "", "AWS region")
	cmd.Flags().StringVar(profile, "profile", "", "AWS CLI profile")
}

// awsRunner runs aws commands for the hidden AWS subcommands, capturing
// their JSON output.
func awsRunner(name string, timeout int) func([]string) (string, error) {
	return func(c []string) (string, error) {
		return workers.CaptureWorker(workers.NewTaskInfo(c, "", name, "root", nil, nil, timeout, false, 1))
	}
}

//nolint:gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(ECSDeployCmd())
	rootCmd.AddCommand(ECSWaitCmd())
}
