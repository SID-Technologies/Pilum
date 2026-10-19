package aws_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/aws"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func ecsService(ecs map[string]any) serviceinfo.ServiceInfo {
	return *serviceinfo.NewServiceInfo(map[string]any{
		"name": "api", "provider": "aws", "type": "aws-ecs", "region": "us-east-1",
		"aws": map[string]any{"account_id": "123456789012", "profile": "prod"},
		"ecs": ecs,
	}, "services/api")
}

func TestParseECSConfig(t *testing.T) {
	t.Parallel()

	cfg := aws.ParseECSConfig(ecsService(map[string]any{"cluster": "prod"}))
	require.Equal(t, aws.ECSConfig{
		Cluster: "prod", Service: "api", DesiredCount: -1,
		WaitTimeout: aws.DefaultECSWaitTimeout, LogGroup: "/ecs/api",
	}, cfg)

	// YAML numbers are fine for the sizes.
	cfg = aws.ParseECSConfig(ecsService(map[string]any{
		"cluster": "prod", "service": "api-svc", "cpu": 512, "memory": 1024, "desired_count": 0,
	}))
	require.Equal(t, "api-svc", cfg.Service)
	require.Equal(t, "512", cfg.CPU)
	require.Equal(t, "1024", cfg.Memory)
	require.Equal(t, 0, cfg.DesiredCount)
	require.Equal(t, "/ecs/api-svc", cfg.LogGroup)
}

func TestGenerateECSDeployArgs(t *testing.T) {
	t.Parallel()

	svc := ecsService(map[string]any{"cluster": "prod", "container": "web", "cpu": "512", "desired_count": 2})
	require.Equal(t, []string{"ecs-deploy",
		"--cluster", "prod", "--service", "api", "--image", ecrImage,
		"--region", "us-east-1", "--profile", "prod",
		"--container", "web", "--cpu", "512", "--desired-count", "2",
	}, aws.GenerateECSDeployArgs(svc, ecrImage))

	svc.EnvVars = []serviceinfo.EnvVars{{Name: "B", Value: "2"}, {Name: "A", Value: "1"}}
	svc.Secrets = []serviceinfo.Secrets{{Name: "DB_URL", Value: "api/db-url"}}
	args := aws.GenerateECSDeployArgs(svc, ecrImage)
	require.Equal(t, []string{"--env", "A=1", "--env", "B=2", "--secret", "DB_URL=api/db-url"}, args[len(args)-6:])
}

func TestGenerateECSWaitArgs(t *testing.T) {
	t.Parallel()

	svc := ecsService(map[string]any{"cluster": "prod", "wait_timeout": 300})
	require.Equal(t, []string{"ecs-wait",
		"--cluster", "prod", "--service", "api", "--image", ecrImage, "--wait-timeout", "300",
		"--region", "us-east-1", "--profile", "prod",
	}, aws.GenerateECSWaitArgs(svc, ecrImage))
}

func TestECSStatusAndLogs(t *testing.T) {
	t.Parallel()

	svc := ecsService(map[string]any{"cluster": "prod"})
	require.Equal(t, []string{"aws", "ecs", "describe-services", "--cluster", "prod", "--services", "api",
		"--output", "json", "--region", "us-east-1", "--profile", "prod"}, aws.GenerateECSStatusCommand(svc))
	require.Equal(t, []string{"aws", "logs", "tail", "/ecs/api", "--region", "us-east-1", "--profile", "prod", "--follow"},
		aws.GenerateECSLogsCommand(svc, true))
}
