package cmd

import (
	"time"

	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/lambdadeploy"
	"github.com/sid-technologies/pilum/lib/output"

	"github.com/spf13/cobra"
)

// lambdaPollInterval is how often lambda-deploy checks an update.
const lambdaPollInterval = 3 * time.Second

// LambdaDeployCmd is the "deploy to lambda" step of the aws-lambda and
// aws-lambda-image recipes: update the code and settings, publish a
// version, move the alias. Hidden: it's plumbing for the recipes, not
// something to run by hand.
func LambdaDeployCmd() *cobra.Command {
	var req lambdadeploy.Request
	var timeout, waitTimeout int

	cmd := &cobra.Command{
		Use:    "lambda-deploy",
		Short:  "Deploy new code to a Lambda function, publish a version and move its alias",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Only what the service sets is replaced.
			req.SetEnv = cmd.Flags().Changed("env")
			req.WaitTimeout = time.Duration(waitTimeout) * time.Second
			req.Interval = lambdaPollInterval
			clock := lambdadeploy.Clock{Now: time.Now, Sleep: time.Sleep}

			result, err := lambdadeploy.Deploy(req, awsRunner(req.Function, timeout), clock)
			if err != nil {
				return errors.Wrap(err, "deploying to Lambda")
			}
			switch result.Previous {
			case "":
				output.Success("Created alias %s:%s → version %s", req.Function, req.Alias, result.Version)
			case result.Version:
				output.Info("%s:%s already serves version %s", req.Function, req.Alias, result.Version)
			default:
				output.Success("%s:%s → version %s (was %s)", req.Function, req.Alias, result.Version, result.Previous)
			}
			return nil
		},
	}

	addAWSTargetFlags(cmd, &req.Region, &req.Profile)
	cmd.Flags().StringVar(&req.Function, "function", "", "Function name or ARN")
	cmd.Flags().StringVar(&req.Alias, "alias", "live", "Alias to move to the new version")
	cmd.Flags().StringVar(&req.Image, "image", "", "Image URI (image functions)")
	cmd.Flags().StringVar(&req.Zip, "zip", "", "A .zip, or a directory to zip (zip functions)")
	cmd.Flags().StringVar(&req.S3Bucket, "s3-bucket", "", "Upload the zip through this bucket")
	cmd.Flags().IntVar(&req.Memory, "memory", 0, "Memory in MB (default: keep)")
	cmd.Flags().IntVar(&req.Timeout, "timeout-seconds", 0, "Function timeout in seconds (default: keep)")
	cmd.Flags().StringVar(&req.Runtime, "runtime", "", "Runtime (default: keep)")
	cmd.Flags().StringVar(&req.Handler, "handler", "", "Handler (default: keep)")
	cmd.Flags().StringVar(&req.Architecture, "architecture", "", "x86_64 or arm64 (default: keep)")
	cmd.Flags().StringArrayVar(&req.Env, "env", nil, "KEY=VALUE env var, repeatable; replaces the function's environment")
	cmd.Flags().StringVar(&req.Description, "version-description", "", "Description of the published version")
	cmd.Flags().IntVar(&waitTimeout, "wait-timeout", 300, "Seconds to wait for each function update")
	cmd.Flags().IntVar(&timeout, "timeout", 120, "Timeout per aws command in seconds")

	return cmd
}

//nolint:gochecknoinits // Standard Cobra pattern for initializing commands
func init() {
	rootCmd.AddCommand(LambdaDeployCmd())
}
