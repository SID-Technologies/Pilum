package aws_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/aws"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateStatusCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "my-function",
		Region: "us-east-1",
	}

	cmd := aws.GenerateStatusCommand(svc)

	require.Equal(t, "aws", cmd[0])
	require.Equal(t, "lambda", cmd[1])
	require.Equal(t, "get-function", cmd[2])
	require.Contains(t, cmd, "--function-name")
	require.Contains(t, cmd, "my-function")
	require.Contains(t, cmd, "--region")
	require.Contains(t, cmd, "us-east-1")
	require.Contains(t, cmd, "--output")
	require.Contains(t, cmd, "json")
}

func TestGenerateStatusCommandNoRegion(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name: "my-function",
	}

	cmd := aws.GenerateStatusCommand(svc)

	require.NotContains(t, cmd, "--region")
}

func TestGenerateLogsCommand(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "my-function",
		Region: "us-east-1",
	}

	cmd := aws.GenerateLogsCommand(svc, false)

	require.Equal(t, "aws", cmd[0])
	require.Equal(t, "logs", cmd[1])
	require.Equal(t, "tail", cmd[2])
	require.Equal(t, "/aws/lambda/my-function", cmd[3])
	require.Contains(t, cmd, "--region")
	require.NotContains(t, cmd, "--follow")
}

func TestGenerateLogsCommandFollow(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:   "my-function",
		Region: "us-east-1",
	}

	cmd := aws.GenerateLogsCommand(svc, true)

	require.Contains(t, cmd, "--follow")
}
