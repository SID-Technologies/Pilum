package aws_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/aws"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

const ecrImage = "123456789012.dkr.ecr.us-east-1.amazonaws.com/team/api:v1.2.0"

func awsService(awsBlock map[string]any) serviceinfo.ServiceInfo {
	config := map[string]any{"name": "api", "provider": "aws", "region": "us-east-1"}
	if awsBlock != nil {
		config["aws"] = awsBlock
	}
	return *serviceinfo.NewServiceInfo(config, "services/api")
}

func TestParseConfig(t *testing.T) {
	t.Parallel()

	cfg := aws.ParseConfig(awsService(map[string]any{
		"account_id":        "012345678901",
		"profile":           "prod",
		"create_repository": true,
	}))
	require.Equal(t, aws.Config{AccountID: "012345678901", Profile: "prod", CreateRepository: true}, cfg)

	// registry_name is where AWS services kept the account ID before aws:.
	svc := awsService(nil)
	svc.RegistryName = "123456789012"
	require.Equal(t, "123456789012", aws.AccountID(svc))
	require.Equal(t, "123456789012.dkr.ecr.us-east-1.amazonaws.com", aws.RegistryHost(svc))

	svc.Region = ""
	require.Empty(t, aws.RegistryHost(svc))
}

func TestGenerateECRLoginCommand(t *testing.T) {
	t.Parallel()

	svc := awsService(map[string]any{"profile": "prod"})
	require.Equal(t,
		"'aws' 'ecr' 'get-login-password' '--region' 'us-east-1' '--profile' 'prod' | "+
			"docker login --username AWS --password-stdin '123456789012.dkr.ecr.us-east-1.amazonaws.com'",
		aws.GenerateECRLoginCommand(svc, ecrImage))

	// Login targets the registry the image is in, not the service's region.
	cmd := aws.GenerateECRLoginCommand(svc, "123456789012.dkr.ecr.eu-west-1.amazonaws.com/api:v1")
	require.Contains(t, cmd, "'--region' 'eu-west-1'")

	require.Empty(t, aws.GenerateECRLoginCommand(svc, "ghcr.io/org/api:v1"))
}

func TestGenerateCreateRepositoryCommand(t *testing.T) {
	t.Parallel()

	require.Empty(t, aws.GenerateCreateRepositoryCommand(awsService(nil), ecrImage))

	cmd := aws.GenerateCreateRepositoryCommand(awsService(map[string]any{"create_repository": true}), ecrImage)
	require.Equal(t,
		"'aws' 'ecr' 'describe-repositories' '--repository-names' 'team/api' '--region' 'us-east-1' >/dev/null 2>&1 || "+
			"'aws' 'ecr' 'create-repository' '--repository-name' 'team/api' "+
			"'--image-scanning-configuration' 'scanOnPush=true' '--region' 'us-east-1' >/dev/null",
		cmd)
}

func TestRepositoryName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "team/api", aws.RepositoryName(ecrImage))
	require.Equal(t, "api", aws.RepositoryName("123456789012.dkr.ecr.us-east-1.amazonaws.com/api@sha256:abc"))
	require.Equal(t, "api", aws.RepositoryName("api:v1"))
	require.Equal(t, "svc", aws.RepositoryName("localhost:5000/svc"))
}

func TestIsECRHost(t *testing.T) {
	t.Parallel()

	require.True(t, aws.IsECRHost("123456789012.dkr.ecr.us-east-1.amazonaws.com"))
	require.True(t, aws.IsECRHost("123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn"))
	require.False(t, aws.IsECRHost("public.ecr.aws"))
	require.False(t, aws.IsECRHost("us-docker.pkg.dev"))
	require.False(t, aws.IsECRHost(""))
}

func TestStatusAndLogsUseProfile(t *testing.T) {
	t.Parallel()

	svc := awsService(map[string]any{"profile": "prod"})
	require.Contains(t, aws.GenerateStatusCommand(svc), "prod")
	require.Contains(t, aws.GenerateLogsCommand(svc, true), "prod")
}
