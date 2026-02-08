package query_test

import (
	"testing"

	"github.com/sid-technologies/pilum/lib/query"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func TestGenerateStatusCommandGCP(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "api-gateway",
		Template: "gcp-cloud-run",
		Provider: "gcp",
		Region:   "us-central1",
	}

	cmd, err := query.GenerateStatusCommand(svc)

	require.NoError(t, err)
	require.Equal(t, "gcloud", cmd[0])
	require.Contains(t, cmd, "services")
}

func TestGenerateStatusCommandGCPJob(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "data-processor",
		Template: "gcp-cloud-run-job",
		Provider: "gcp",
		Region:   "us-central1",
	}

	cmd, err := query.GenerateStatusCommand(svc)

	require.NoError(t, err)
	require.Contains(t, cmd, "jobs")
}

func TestGenerateStatusCommandAzure(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "my-app",
		Template: "azure-container-apps",
		Provider: "azure",
		Project:  "my-rg",
	}

	cmd, err := query.GenerateStatusCommand(svc)

	require.NoError(t, err)
	require.Equal(t, "az", cmd[0])
}

func TestGenerateStatusCommandAWS(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "my-function",
		Template: "aws-lambda",
		Provider: "aws",
		Region:   "us-east-1",
	}

	cmd, err := query.GenerateStatusCommand(svc)

	require.NoError(t, err)
	require.Equal(t, "aws", cmd[0])
}

func TestGenerateStatusCommandCloudflare(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "my-site",
		Template: "cloudflare-pages",
		Provider: "cloudflare",
	}

	cmd, err := query.GenerateStatusCommand(svc)

	require.NoError(t, err)
	require.Equal(t, "npx", cmd[0])
}

func TestGenerateStatusCommandUnsupported(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "my-formula",
		Template: "homebrew",
		Provider: "homebrew",
	}

	_, err := query.GenerateStatusCommand(svc)

	require.Error(t, err)
}

func TestGenerateStatusCommandProviderFallback(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "my-service",
		Template: "gcp",
		Provider: "gcp",
		Region:   "us-central1",
	}

	cmd, err := query.GenerateStatusCommand(svc)

	require.NoError(t, err)
	require.Equal(t, "gcloud", cmd[0])
}

func TestGenerateLogsCommandGCP(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "api-gateway",
		Template: "gcp-cloud-run",
		Provider: "gcp",
		Region:   "us-central1",
	}

	cmd, err := query.GenerateLogsCommand(svc, 50, false)

	require.NoError(t, err)
	require.Equal(t, "gcloud", cmd[0])
	require.Contains(t, cmd, "logs")
}

func TestGenerateLogsCommandUnsupported(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "my-formula",
		Template: "homebrew",
		Provider: "homebrew",
	}

	_, err := query.GenerateLogsCommand(svc, 50, false)

	require.Error(t, err)
}

func TestParseGCPStatusResponse(t *testing.T) {
	t.Parallel()

	jsonOutput := `{
		"metadata": {
			"annotations": {
				"run.googleapis.com/lastModifiedTimestamp": "2025-01-15T10:30:00Z"
			}
		},
		"status": {
			"url": "https://api-gateway-abc123.run.app",
			"conditions": [
				{"type": "Ready", "status": "True"}
			]
		},
		"spec": {
			"template": {
				"spec": {
					"containers": [
						{"image": "gcr.io/my-project/api-gateway:v1.2.3"}
					]
				}
			}
		}
	}`

	svc := serviceinfo.ServiceInfo{
		Name:     "api-gateway",
		Template: "gcp-cloud-run",
		Provider: "gcp",
		Region:   "us-central1",
	}

	status := query.ParseStatusResponse(jsonOutput, svc)

	require.Equal(t, "api-gateway", status.Name)
	require.Equal(t, "gcp", status.Provider)
	require.Equal(t, "us-central1", status.Region)
	require.Equal(t, "Ready", status.Status)
	require.Equal(t, "https://api-gateway-abc123.run.app", status.URL)
	require.Equal(t, "gcr.io/my-project/api-gateway:v1.2.3", status.Image)
	require.Equal(t, "2025-01-15T10:30:00Z", status.Updated)
}

func TestParseGCPStatusResponseNotReady(t *testing.T) {
	t.Parallel()

	jsonOutput := `{
		"status": {
			"conditions": [
				{"type": "Ready", "status": "False"}
			]
		}
	}`

	svc := serviceinfo.ServiceInfo{
		Name:     "api-gateway",
		Template: "gcp-cloud-run",
		Provider: "gcp",
	}

	status := query.ParseStatusResponse(jsonOutput, svc)

	require.Equal(t, "NotReady", status.Status)
}

func TestParseAzureStatusResponse(t *testing.T) {
	t.Parallel()

	jsonOutput := `{
		"location": "eastus",
		"properties": {
			"provisioningState": "Succeeded",
			"configuration": {
				"ingress": {
					"fqdn": "my-app.azurecontainerapps.io"
				}
			},
			"template": {
				"containers": [
					{"image": "myregistry.azurecr.io/my-app:v1.0"}
				]
			}
		}
	}`

	svc := serviceinfo.ServiceInfo{
		Name:     "my-app",
		Template: "azure-container-apps",
		Provider: "azure",
	}

	status := query.ParseStatusResponse(jsonOutput, svc)

	require.Equal(t, "my-app", status.Name)
	require.Equal(t, "azure", status.Provider)
	require.Equal(t, "Succeeded", status.Status)
	require.Equal(t, "https://my-app.azurecontainerapps.io", status.URL)
	require.Equal(t, "myregistry.azurecr.io/my-app:v1.0", status.Image)
	require.Equal(t, "eastus", status.Region)
}

func TestParseAWSStatusResponse(t *testing.T) {
	t.Parallel()

	jsonOutput := `{
		"Configuration": {
			"State": "Active",
			"LastModified": "2025-01-15T10:30:00Z"
		}
	}`

	svc := serviceinfo.ServiceInfo{
		Name:     "my-function",
		Template: "aws-lambda",
		Provider: "aws",
		Region:   "us-east-1",
	}

	status := query.ParseStatusResponse(jsonOutput, svc)

	require.Equal(t, "my-function", status.Name)
	require.Equal(t, "aws", status.Provider)
	require.Equal(t, "Active", status.Status)
	require.Equal(t, "2025-01-15T10:30:00Z", status.Updated)
}

func TestParseStatusResponseInvalidJSON(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:     "api-gateway",
		Template: "gcp-cloud-run",
		Provider: "gcp",
		Region:   "us-central1",
	}

	status := query.ParseStatusResponse("not json", svc)

	require.Equal(t, "Unknown", status.Status)
}
