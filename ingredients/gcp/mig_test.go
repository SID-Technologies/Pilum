package gcp_test

import (
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

func migService(mig map[string]any) serviceinfo.ServiceInfo {
	return serviceinfo.ServiceInfo{
		Name:     "statio-egress-proxy",
		Type:     "gcp-mig-container",
		Provider: "gcp",
		Project:  "statio-499700",
		Region:   "us-central1",
		Image:    "us-central1-docker.pkg.dev/statio-499700/statio/statio-egress-proxy",
		Config:   map[string]any{"mig": mig},
	}
}

func zonalMIG() map[string]any {
	return map[string]any{
		"name":          "egress-proxy-mig",
		"zone":          "us-central1-a",
		"template_base": "egress-proxy-tpl",
	}
}

func TestParseMIGConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg := gcp.ParseMIGConfig(map[string]any{"mig": zonalMIG()})
	require.Equal(t, "egress-proxy-mig", cfg.Name)
	require.Equal(t, "us-central1-a", cfg.Zone)
	require.Equal(t, "egress-proxy-tpl", cfg.TemplateBase)
	require.Equal(t, "1", cfg.MaxSurge)
	require.Equal(t, "0", cfg.MaxUnavailable)
	require.Equal(t, 600, cfg.WaitTimeout)
	require.Equal(t, "container-image", cfg.ImageMetadataKey)
	require.Equal(t, []string{"--zone", "us-central1-a"}, cfg.LocationFlags())

	regional := gcp.ParseMIGConfig(map[string]any{"mig": map[string]any{"name": "m", "region": "us-central1"}})
	require.Equal(t, "3", regional.MaxSurge, "regional surge must cover the zone count")
	require.Equal(t, []string{"--region", "us-central1"}, regional.LocationFlags())
}

func TestParseMIGConfigOverrides(t *testing.T) {
	t.Parallel()

	cfg := gcp.ParseMIGConfig(map[string]any{"mig": map[string]any{
		"name":               "m",
		"region":             "us-central1",
		"max_surge":          2, // YAML ints arrive as int
		"max_unavailable":    "10%",
		"wait_timeout":       900,
		"image_metadata_key": "app-image",
	}})
	require.Equal(t, "2", cfg.MaxSurge)
	require.Equal(t, "10%", cfg.MaxUnavailable)
	require.Equal(t, 900, cfg.WaitTimeout)
	require.Equal(t, "app-image", cfg.ImageMetadataKey)
}

func TestMIGTemplateName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tag, want string
	}{
		{"v1", "egress-proxy-tpl-v1"},
		{"v1.4.2", "egress-proxy-tpl-v1-4-2"},
		{"V1.4.2+Build_7", "egress-proxy-tpl-v1-4-2-build-7"},
		{"abc1234", "egress-proxy-tpl-abc1234"},
		{"-rc1-", "egress-proxy-tpl-rc1"},
	}
	for _, tt := range tests {
		got, err := gcp.MIGTemplateName("egress-proxy-tpl", tt.tag)
		require.NoError(t, err, tt.tag)
		require.Equal(t, tt.want, got, tt.tag)
	}
}

func TestMIGTemplateNameErrors(t *testing.T) {
	t.Parallel()

	_, err := gcp.MIGTemplateName("egress-proxy-tpl", "latest")
	require.ErrorContains(t, err, "unique --tag")

	_, err = gcp.MIGTemplateName("egress-proxy-tpl", "")
	require.ErrorContains(t, err, "unique --tag")

	_, err = gcp.MIGTemplateName("", "v1")
	require.ErrorContains(t, err, "template_base")

	// Too long is an error, never a truncation that could collide.
	_, err = gcp.MIGTemplateName("egress-proxy-tpl", "0123456789012345678901234567890123456789012345678")
	require.ErrorContains(t, err, "GCP allows 63")

	_, err = gcp.MIGTemplateName("1-bad-base", "v1")
	require.ErrorContains(t, err, "invalid")
}

func TestMIGTemplateRef(t *testing.T) {
	t.Parallel()

	require.Equal(t, "egress-proxy-tpl-v2", gcp.MIGTemplateRef(migService(zonalMIG()), "egress-proxy-tpl-v2"))

	mig := zonalMIG()
	mig["template_region"] = "us-central1"
	require.Equal(t,
		"projects/statio-499700/regions/us-central1/instanceTemplates/egress-proxy-tpl-v2",
		gcp.MIGTemplateRef(migService(mig), "egress-proxy-tpl-v2"))
}

func TestGenerateMIGRollingUpdateCommand(t *testing.T) {
	t.Parallel()

	cmd := gcp.GenerateMIGRollingUpdateCommand(migService(zonalMIG()), "egress-proxy-tpl-v2")
	require.Equal(t, []string{
		"gcloud", "compute", "instance-groups", "managed", "rolling-action", "start-update", "egress-proxy-mig",
		"--version", "template=egress-proxy-tpl-v2",
		"--max-surge", "1",
		"--max-unavailable", "0",
		"--zone", "us-central1-a",
		"--project", "statio-499700",
	}, cmd)
}

func TestGenerateMIGRollingUpdateCommandRegional(t *testing.T) {
	t.Parallel()

	svc := migService(map[string]any{
		"name":            "egress-proxy-mig",
		"region":          "us-central1",
		"template_base":   "egress-proxy-tpl",
		"template_region": "us-central1",
	})
	cmd := gcp.GenerateMIGRollingUpdateCommand(svc, "egress-proxy-tpl-v2")
	require.Equal(t, []string{
		"gcloud", "compute", "instance-groups", "managed", "rolling-action", "start-update", "egress-proxy-mig",
		"--version", "template=projects/statio-499700/regions/us-central1/instanceTemplates/egress-proxy-tpl-v2",
		"--max-surge", "3",
		"--max-unavailable", "0",
		"--region", "us-central1",
		"--project", "statio-499700",
	}, cmd)
}

func TestGenerateMIGWaitStableCommand(t *testing.T) {
	t.Parallel()

	cmd := gcp.GenerateMIGWaitStableCommand(migService(zonalMIG()))
	require.Equal(t, []string{
		"gcloud", "compute", "instance-groups", "managed", "wait-until", "egress-proxy-mig",
		"--stable",
		"--timeout", "600",
		"--zone", "us-central1-a",
		"--project", "statio-499700",
	}, cmd)
}

func TestGenerateMIGDescribeAndListCommands(t *testing.T) {
	t.Parallel()

	svc := migService(zonalMIG())
	require.Equal(t, []string{
		"gcloud", "compute", "instance-groups", "managed", "describe", "egress-proxy-mig",
		"--format", "json",
		"--zone", "us-central1-a",
		"--project", "statio-499700",
	}, gcp.GenerateMIGDescribeCommand(svc))

	require.Equal(t, []string{
		"gcloud", "compute", "instance-templates", "list",
		"--filter", "name=egress-proxy-tpl OR name~^egress-proxy-tpl-",
		"--format", "json",
		"--project", "statio-499700",
	}, gcp.GenerateListMIGTemplatesCommand(svc))
}

func TestParseMIGConfigVMOverrides(t *testing.T) {
	t.Parallel()

	cfg := gcp.ParseMIGConfig(map[string]any{"mig": map[string]any{
		"zone":            "us-central1-a",
		"machine_type":    "e2-medium",
		"network":         "statio-vpc",
		"subnet":          "proxy-subnet",
		"external_ip":     false,
		"network_tags":    []any{"egress-proxy"},
		"service_account": "proxy@p.iam.gserviceaccount.com",
	}})
	require.Equal(t, "e2-medium", cfg.MachineType)
	require.Equal(t, "statio-vpc", cfg.Network)
	require.Equal(t, "proxy-subnet", cfg.Subnet)
	require.NotNil(t, cfg.ExternalIP)
	require.False(t, *cfg.ExternalIP)
	require.True(t, cfg.SetNetworkTags)
	require.Equal(t, []string{"egress-proxy"}, cfg.NetworkTags)
	require.Equal(t, "us-central1", cfg.SubnetRegion())

	unset := gcp.ParseMIGConfig(map[string]any{"mig": map[string]any{"region": "europe-west1"}})
	require.Nil(t, unset.ExternalIP, "unset keeps the template's")
	require.False(t, unset.SetNetworkTags)
	require.Equal(t, "europe-west1", unset.SubnetRegion())

	empty := gcp.ParseMIGConfig(map[string]any{"mig": map[string]any{"network_tags": []any{}}})
	require.True(t, empty.SetNetworkTags, "an empty list clears the tags")
}

func TestGenerateMIGTemplateArgsMinimal(t *testing.T) {
	t.Parallel()

	args := gcp.GenerateMIGTemplateArgs(migService(zonalMIG()), "egress-proxy-tpl-v2", "r/proxy:v2")
	require.Equal(t, []string{
		"mig-template",
		"--project", "statio-499700",
		"--mig", "egress-proxy-mig",
		"--name", "egress-proxy-tpl-v2",
		"--image", "r/proxy:v2",
		"--image-key", "container-image",
		"--zone", "us-central1-a",
	}, args)
}

func TestGenerateMIGTemplateArgsFull(t *testing.T) {
	t.Parallel()

	mig := zonalMIG()
	mig["machine_type"] = "e2-medium"
	mig["network"] = "statio-vpc"
	mig["subnet"] = "proxy-subnet"
	mig["external_ip"] = false
	mig["network_tags"] = []any{"egress-proxy", "allow-hc"}
	mig["service_account"] = "proxy@statio-499700.iam.gserviceaccount.com"
	svc := migService(mig)
	svc.EnvVars = []serviceinfo.EnvVars{{Name: "PORT", Value: "3128"}, {Name: "LOG_LEVEL", Value: "info"}}
	svc.Secrets = []serviceinfo.Secrets{{Name: "API_KEY", Value: "api-key"}, {Name: "DB_URL", Value: "db-url:3"}}

	args := gcp.GenerateMIGTemplateArgs(svc, "egress-proxy-tpl-v2", "r/proxy:v2")
	require.Equal(t, []string{
		"mig-template",
		"--project", "statio-499700",
		"--mig", "egress-proxy-mig",
		"--name", "egress-proxy-tpl-v2",
		"--image", "r/proxy:v2",
		"--image-key", "container-image",
		"--zone", "us-central1-a",
		"--machine-type", "e2-medium",
		"--network", "statio-vpc",
		"--subnet", "proxy-subnet",
		"--service-account", "proxy@statio-499700.iam.gserviceaccount.com",
		"--subnet-region", "us-central1",
		"--external-ip=false",
		"--network-tags=egress-proxy,allow-hc",
		"--env", "LOG_LEVEL=info", // sorted, so the command is stable across runs
		"--env", "PORT=3128",
		"--secret", "API_KEY=projects/statio-499700/secrets/api-key/versions/latest",
		"--secret", "DB_URL=projects/statio-499700/secrets/db-url/versions/3",
	}, args)
}

func TestSecretVersionResource(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{"api-key", "projects/p/secrets/api-key/versions/latest"},
		{"api-key:3", "projects/p/secrets/api-key/versions/3"},
		{"api-key:", "projects/p/secrets/api-key/versions/latest"},
		{"projects/other/secrets/api-key/versions/2", "projects/other/secrets/api-key/versions/2"},
		{"projects/other/secrets/api-key", "projects/other/secrets/api-key/versions/latest"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, gcp.SecretVersionResource("p", tt.in), tt.in)
	}
}
