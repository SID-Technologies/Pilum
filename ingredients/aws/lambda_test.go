package aws_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/aws"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func lambdaService(typ string, lambda map[string]any) serviceinfo.ServiceInfo {
	config := map[string]any{"name": "fn", "provider": "aws", "type": typ, "region": "us-east-1"}
	if lambda != nil {
		config["lambda"] = lambda
	}
	return *serviceinfo.NewServiceInfo(config, "services/fn")
}

func TestGenerateLambdaDeployArgsZip(t *testing.T) {
	t.Parallel()

	svc := lambdaService("aws-lambda", map[string]any{
		"runtime": "provided.al2023", "handler": "bootstrap", "architecture": "arm64",
		"memory": 256, "timeout": 15, "s3_bucket": "artifacts",
	})
	svc.EnvVars = []serviceinfo.EnvVars{{Name: "B", Value: "2"}, {Name: "A", Value: "1"}}

	require.Equal(t, []string{"lambda-deploy",
		"--function", "fn", "--alias", "live", "--version-description", "pilum v1.2.0", "--wait-timeout", "300",
		"--zip", filepath.Join("services", "fn", "dist"),
		"--region", "us-east-1",
		"--runtime", "provided.al2023", "--handler", "bootstrap", "--architecture", "arm64", "--s3-bucket", "artifacts",
		"--memory", "256", "--timeout-seconds", "15",
		"--env", "A=1", "--env", "B=2",
	}, aws.GenerateLambdaDeployArgs(svc, "", "v1.2.0"))
}

func TestGenerateLambdaDeployArgsImage(t *testing.T) {
	t.Parallel()

	svc := lambdaService("aws-lambda-image", map[string]any{"function": "prod-fn", "alias": "prod"})
	args := aws.GenerateLambdaDeployArgs(svc, ecrImage, "v2")
	require.Equal(t, []string{"lambda-deploy",
		"--function", "prod-fn", "--alias", "prod", "--version-description", "pilum v2", "--wait-timeout", "300",
		"--image", ecrImage, "--region", "us-east-1",
	}, args)
}

func TestLambdaPlatform(t *testing.T) {
	t.Parallel()

	require.Equal(t, "linux/arm64", aws.LambdaPlatform(lambdaService("aws-lambda-image", map[string]any{"architecture": "arm64"})))
	require.Empty(t, aws.LambdaPlatform(lambdaService("aws-lambda-image", nil)))
	require.Empty(t, aws.LambdaPlatform(lambdaService("aws-ecs", map[string]any{"architecture": "arm64"})))
}

func TestLambdaLogsAndStatus(t *testing.T) {
	t.Parallel()

	svc := lambdaService("aws-lambda", map[string]any{"function": "arn:aws:lambda:us-east-1:123456789012:function:prod-fn:live"})
	require.Equal(t, "/aws/lambda/prod-fn", aws.GenerateLogsCommand(svc, false)[3])
	require.Contains(t, aws.GenerateLambdaStatusCommand(svc), "--qualifier")
}

func TestValidateLambda(t *testing.T) {
	t.Parallel()

	svc := lambdaService("aws-lambda-image", nil)
	require.NoError(t, aws.ValidateLambda(svc))

	svc.Secrets = []serviceinfo.Secrets{{Name: "DB", Value: "db"}}
	require.ErrorContains(t, aws.ValidateLambda(svc), "lambda has no secret references")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "template.yaml"), []byte("Resources: {}\n"), 0o600))
	sam := lambdaService("aws-lambda", nil)
	sam.Path = dir
	require.ErrorContains(t, aws.ValidateLambda(sam), "use type: aws-lambda-sam")

	sam.Type = "aws-lambda-sam"
	require.NoError(t, aws.ValidateLambda(sam))
}
