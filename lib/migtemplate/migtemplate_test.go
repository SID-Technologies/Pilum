package migtemplate_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/lib/migtemplate"

	"github.com/stretchr/testify/require"
)

const (
	v1Image = "us-central1-docker.pkg.dev/statio-499700/statio/statio-egress-proxy:v1"
	v2Image = "us-central1-docker.pkg.dev/statio-499700/statio/statio-egress-proxy:v2"

	sourceLink = "https://www.googleapis.com/compute/v1/projects/statio-499700/global/instanceTemplates/egress-proxy-tpl"
)

// Trimmed from `gcloud compute instance-groups managed describe --format json`.
const migFixture = `{
  "name": "egress-proxy-mig",
  "instanceTemplate": "` + sourceLink + `",
  "versions": [{"instanceTemplate": "` + sourceLink + `"}],
  "status": {"isStable": true}
}`

// startupScript is the kind of script the setup script installs: it runs
// whatever image the container-image metadata names.
const startupScript = `#!/bin/bash
IMAGE=$(curl -sf -H Metadata-Flavor:Google http://metadata.google.internal/computeMetadata/v1/instance/attributes/container-image)
docker rm -f app || true
docker run -d --restart=always --name app --network host "$IMAGE"`

// templateFixture is a template as `instance-templates describe/list --format json` emits it.
func templateFixture(name, link, image string) string {
	script, _ := json.Marshal(startupScript)
	return `{
  "kind": "compute#instanceTemplate",
  "id": "123",
  "creationTimestamp": "2026-09-30T10:00:00.000-07:00",
  "name": "` + name + `",
  "description": "egress proxy",
  "selfLink": "` + link + `",
  "properties": {
    "machineType": "e2-small",
    "tags": {"items": ["egress-proxy"]},
    "networkInterfaces": [{
      "network": "https://www.googleapis.com/compute/v1/projects/statio-499700/global/networks/default",
      "subnetwork": "https://www.googleapis.com/compute/v1/projects/statio-499700/regions/us-central1/subnetworks/default",
      "fingerprint": "nic=",
      "accessConfigs": [{"type": "ONE_TO_ONE_NAT", "name": "External NAT"}]
    }],
    "serviceAccounts": [{"email": "proxy@statio-499700.iam.gserviceaccount.com", "scopes": ["https://www.googleapis.com/auth/cloud-platform"]}],
    "metadata": {
      "kind": "compute#metadata",
      "fingerprint": "abc=",
      "items": [
        {"key": "container-image", "value": "` + image + `"},
        {"key": "startup-script", "value": ` + string(script) + `},
        {"key": "google-logging-enabled", "value": "true"}
      ]
    }
  }
}`
}

// fakeRunner answers gcloud commands by substring.
func fakeRunner(responses map[string]string) migtemplate.Runner {
	return func(cmd []string) (string, error) {
		joined := strings.Join(cmd, " ")
		for key, resp := range responses {
			if strings.Contains(joined, key) {
				return resp, nil
			}
		}
		return "", fmt.Errorf("unexpected command: %s", joined)
	}
}

// fakeCompute records requests and answers with operations.
type fakeCompute struct {
	requests []*http.Request
	bodies   []string
	answers  []string
}

func (f *fakeCompute) Do(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	f.requests = append(f.requests, req)
	f.bodies = append(f.bodies, string(body))
	answer := f.answers[0]
	f.answers = f.answers[1:]
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(answer))}, nil
}

const opLink = "https://compute.googleapis.com/compute/v1/projects/statio-499700/global/operations/op-1"

func request() migtemplate.Request {
	return migtemplate.Request{
		Project:  "statio-499700",
		MIG:      "egress-proxy-mig",
		Zone:     "us-central1-a",
		Name:     "egress-proxy-tpl-v2",
		Image:    v2Image,
		ImageKey: gcp.DefaultImageMetadataKey,
	}
}

func TestCurrentTemplate(t *testing.T) {
	t.Parallel()

	got, err := migtemplate.CurrentTemplate(migFixture)
	require.NoError(t, err)
	require.Equal(t, sourceLink, got)

	got, err = migtemplate.CurrentTemplate(`{"instanceTemplate": "x"}`)
	require.NoError(t, err)
	require.Equal(t, "x", got)

	_, err = migtemplate.CurrentTemplate(`{"versions": [{"instanceTemplate": "a"}, {"instanceTemplate": "b"}]}`)
	require.ErrorContains(t, err, "2 versions")

	_, err = migtemplate.CurrentTemplate(`{}`)
	require.Error(t, err)
}

func TestTemplateLinks(t *testing.T) {
	t.Parallel()

	require.Empty(t, migtemplate.TemplateRegion(sourceLink))
	require.Equal(t, "us-central1", migtemplate.TemplateRegion(
		"https://www.googleapis.com/compute/v1/projects/p/regions/us-central1/instanceTemplates/t"))
	require.Equal(t, "egress-proxy-tpl", migtemplate.TemplateName(sourceLink))
	require.Equal(t, "plain", migtemplate.TemplateName("plain"))

	require.Equal(t, "https://compute.googleapis.com/compute/v1/projects/p/global/instanceTemplates",
		migtemplate.InsertURL("p", ""))
	require.Equal(t, "https://compute.googleapis.com/compute/v1/projects/p/regions/us-central1/instanceTemplates",
		migtemplate.InsertURL("p", "us-central1"))
}

func TestInsertBody(t *testing.T) {
	t.Parallel()

	raw, err := migtemplate.InsertBody(templateFixture("egress-proxy-tpl", sourceLink, v1Image), request())
	require.NoError(t, err)

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		ID          string `json:"id"`
		SelfLink    string `json:"selfLink"`
		Properties  struct {
			MachineType string `json:"machineType"`
			Tags        struct {
				Items []string `json:"items"`
			} `json:"tags"`
			ServiceAccounts []map[string]any `json:"serviceAccounts"`
			Metadata        struct {
				Fingerprint string `json:"fingerprint"`
				Items       []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"items"`
			} `json:"metadata"`
		} `json:"properties"`
	}
	err = json.Unmarshal(raw, &body)
	require.NoError(t, err)

	require.Equal(t, "egress-proxy-tpl-v2", body.Name)
	require.Equal(t, "egress proxy", body.Description)
	require.Empty(t, body.ID, "output-only fields are not sent")
	require.Empty(t, body.SelfLink)
	require.Empty(t, body.Properties.Metadata.Fingerprint)

	// Only the image changes; the script and the rest of the template are copied.
	require.Equal(t, "e2-small", body.Properties.MachineType)
	require.Equal(t, []string{"egress-proxy"}, body.Properties.Tags.Items)
	require.Len(t, body.Properties.ServiceAccounts, 1)
	require.Len(t, body.Properties.Metadata.Items, 3)
	require.Equal(t, "container-image", body.Properties.Metadata.Items[0].Key)
	require.Equal(t, v2Image, body.Properties.Metadata.Items[0].Value)
	require.Equal(t, startupScript, body.Properties.Metadata.Items[1].Value)
}

func TestInsertBodyCustomKey(t *testing.T) {
	t.Parallel()

	source := `{"properties": {"metadata": {"items": [{"key": "app-image", "value": "` + v1Image + `"}]}}}`
	req := request()
	req.ImageKey = "app-image"
	raw, err := migtemplate.InsertBody(source, req)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"value":"`+v2Image+`"`)
}

func TestInsertBodyRequiresImageKey(t *testing.T) {
	t.Parallel()

	_, err := migtemplate.InsertBody(`{"properties": {"metadata": {"items": [{"key": "startup-script", "value": "x"}]}}}`, request())
	require.ErrorContains(t, err, "no container-image metadata")

	_, err = migtemplate.InsertBody(`{"properties": {}}`, request())
	require.ErrorContains(t, err, "no container-image metadata")
}

func TestInsertBodyRejectsRetiredContainerAgent(t *testing.T) {
	t.Parallel()

	// A create-with-container template: GCP won't boot VMs from it any more.
	source := `{"properties": {"metadata": {"items": [
		{"key": "gce-container-declaration", "value": "spec:\n  containers:\n    - image: x\n"}]}}}`
	_, err := migtemplate.InsertBody(source, request())
	require.ErrorContains(t, err, "retired container startup agent")
}

func TestCreate(t *testing.T) {
	t.Parallel()

	run := fakeRunner(map[string]string{
		"instance-groups managed describe": migFixture,
		"instance-templates describe":      templateFixture("egress-proxy-tpl", sourceLink, v1Image),
		"instance-templates list":          `[]`,
		"print-access-token":               "tok\n",
	})
	compute := &fakeCompute{answers: []string{
		`{"selfLink": "` + opLink + `", "status": "RUNNING"}`,
		`{"selfLink": "` + opLink + `", "status": "DONE"}`,
	}}

	result, err := migtemplate.Create(request(), run, compute)
	require.NoError(t, err)
	require.True(t, result.Created)
	require.Equal(t, "egress-proxy-tpl", result.Source)

	require.Len(t, compute.requests, 2)
	require.Equal(t, "https://compute.googleapis.com/compute/v1/projects/statio-499700/global/instanceTemplates",
		compute.requests[0].URL.String())
	require.Equal(t, "Bearer tok", compute.requests[0].Header.Get("Authorization"))
	require.Contains(t, compute.bodies[0], `"name":"egress-proxy-tpl-v2"`)
	require.Equal(t, opLink+"/wait", compute.requests[1].URL.String())
}

func TestCreateOperationError(t *testing.T) {
	t.Parallel()

	run := fakeRunner(map[string]string{
		"instance-groups managed describe": migFixture,
		"instance-templates describe":      templateFixture("egress-proxy-tpl", sourceLink, v1Image),
		"instance-templates list":          `[]`,
		"print-access-token":               "tok",
	})
	compute := &fakeCompute{answers: []string{
		`{"selfLink": "` + opLink + `", "status": "DONE", "error": {"errors": [{"code": "QUOTA_EXCEEDED", "message": "too many templates"}]}}`,
	}}

	_, err := migtemplate.Create(request(), run, compute)
	require.ErrorContains(t, err, "QUOTA_EXCEEDED: too many templates")
}

func TestCreateReusesIdenticalTemplate(t *testing.T) {
	t.Parallel()

	existing := templateFixture("egress-proxy-tpl-v2",
		"https://www.googleapis.com/compute/v1/projects/statio-499700/global/instanceTemplates/egress-proxy-tpl-v2", v2Image)
	run := fakeRunner(map[string]string{
		"instance-groups managed describe": migFixture,
		"instance-templates describe":      templateFixture("egress-proxy-tpl", sourceLink, v1Image),
		"instance-templates list":          "[" + existing + "]",
	})

	result, err := migtemplate.Create(request(), run, &fakeCompute{})
	require.NoError(t, err)
	require.False(t, result.Created)
}

func TestCreateRejectsReusedTag(t *testing.T) {
	t.Parallel()

	// <base>-v2 exists but runs a different image: the tag was reused.
	existing := templateFixture("egress-proxy-tpl-v2",
		"https://www.googleapis.com/compute/v1/projects/statio-499700/global/instanceTemplates/egress-proxy-tpl-v2", v1Image)
	run := fakeRunner(map[string]string{
		"instance-groups managed describe": migFixture,
		"instance-templates describe":      templateFixture("egress-proxy-tpl", sourceLink, v1Image),
		"instance-templates list":          "[" + existing + "]",
	})

	_, err := migtemplate.Create(request(), run, &fakeCompute{})
	require.ErrorContains(t, err, "already exists and runs "+v1Image)
}

func TestCreateWhenMIGAlreadyOnTemplate(t *testing.T) {
	t.Parallel()

	link := "https://www.googleapis.com/compute/v1/projects/statio-499700/global/instanceTemplates/egress-proxy-tpl-v2"
	run := fakeRunner(map[string]string{
		"instance-groups managed describe": `{"versions": [{"instanceTemplate": "` + link + `"}]}`,
	})

	result, err := migtemplate.Create(request(), run, &fakeCompute{})
	require.NoError(t, err)
	require.False(t, result.Created)
	require.Equal(t, "egress-proxy-tpl-v2", result.Source)
}

func TestCreateScopeMismatch(t *testing.T) {
	t.Parallel()

	run := fakeRunner(map[string]string{
		"instance-groups managed describe": `{"versions": [{"instanceTemplate": ` +
			`"https://www.googleapis.com/compute/v1/projects/statio-499700/regions/us-central1/instanceTemplates/egress-proxy-tpl"}]}`,
	})

	_, err := migtemplate.Create(request(), run, &fakeCompute{})
	require.ErrorContains(t, err, "set mig.template_region")
}

func TestCreateValidatesRequest(t *testing.T) {
	t.Parallel()

	req := request()
	req.Zone = ""
	_, err := migtemplate.Create(req, fakeRunner(nil), &fakeCompute{})
	require.ErrorContains(t, err, "mig.zone")

	req = request()
	req.Image = ""
	_, err = migtemplate.Create(req, fakeRunner(nil), &fakeCompute{})
	require.Error(t, err)
}

// insertedProps runs InsertBody and returns the new template's properties.
func insertedProps(t *testing.T, req migtemplate.Request) map[string]any {
	t.Helper()

	raw, err := migtemplate.InsertBody(templateFixture("egress-proxy-tpl", sourceLink, v1Image), req)
	require.NoError(t, err)
	var body struct {
		Properties map[string]any `json:"properties"`
	}
	err = json.Unmarshal(raw, &body)
	require.NoError(t, err)
	return body.Properties
}

func metadataValue(props map[string]any, key string) (string, bool) {
	items := props["metadata"].(map[string]any)["items"].([]any)
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["key"] == key {
			return item["value"].(string), true
		}
	}
	return "", false
}

func TestInsertBodyKeepsTemplateWithoutOverrides(t *testing.T) {
	t.Parallel()

	props := insertedProps(t, request())
	require.Equal(t, "e2-small", props["machineType"])
	nic := props["networkInterfaces"].([]any)[0].(map[string]any)
	require.Contains(t, nic["network"], "/networks/default")
	require.Len(t, nic["accessConfigs"], 1)
	require.NotContains(t, nic, "fingerprint", "output-only")
	_, hasEnv := metadataValue(props, "container-env")
	require.False(t, hasEnv)
}

func TestInsertBodyVMOverrides(t *testing.T) {
	t.Parallel()

	req := request()
	req.Overrides = migtemplate.Overrides{
		MachineType:    "e2-medium",
		Network:        "statio-vpc",
		Subnet:         "proxy-subnet",
		SubnetRegion:   "us-central1",
		ExternalIP:     "false",
		NetworkTags:    []string{"egress-proxy", "allow-hc"},
		SetNetworkTags: true,
		ServiceAccount: "proxy@statio-499700.iam.gserviceaccount.com",
	}
	props := insertedProps(t, req)

	require.Equal(t, "e2-medium", props["machineType"])
	require.Equal(t, map[string]any{"items": []any{"egress-proxy", "allow-hc"}}, props["tags"])

	nic := props["networkInterfaces"].([]any)[0].(map[string]any)
	require.Equal(t, "projects/statio-499700/global/networks/statio-vpc", nic["network"])
	require.Equal(t, "projects/statio-499700/regions/us-central1/subnetworks/proxy-subnet", nic["subnetwork"])
	require.NotContains(t, nic, "accessConfigs", "external_ip: false removes the external IP")
	require.NotContains(t, nic, "fingerprint")

	accounts := props["serviceAccounts"].([]any)
	require.Len(t, accounts, 1)
	account := accounts[0].(map[string]any)
	require.Equal(t, "proxy@statio-499700.iam.gserviceaccount.com", account["email"])
	require.Equal(t, []any{"https://www.googleapis.com/auth/cloud-platform"}, account["scopes"], "keeps the template's scopes")
}

func TestInsertBodySubnetOnlyDropsOldNetwork(t *testing.T) {
	t.Parallel()

	req := request()
	req.Overrides = migtemplate.Overrides{
		Subnet: "projects/shared-vpc-host/regions/us-central1/subnetworks/proxy", // Shared VPC: a full path
	}
	props := insertedProps(t, req)

	nic := props["networkInterfaces"].([]any)[0].(map[string]any)
	require.Equal(t, "projects/shared-vpc-host/regions/us-central1/subnetworks/proxy", nic["subnetwork"])
	require.NotContains(t, nic, "network", "GCP infers the network from the subnet")
}

func TestInsertBodyAddsExternalIP(t *testing.T) {
	t.Parallel()

	source := `{"properties": {
	  "metadata": {"items": [{"key": "container-image", "value": "` + v1Image + `"}]},
	  "networkInterfaces": [{"network": "global/networks/default"}]}}`
	req := request()
	req.Overrides.ExternalIP = "true"
	raw, err := migtemplate.InsertBody(source, req)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"accessConfigs":[{"name":"External NAT","type":"ONE_TO_ONE_NAT"}]`)
}

func TestInsertBodyEnvAndSecrets(t *testing.T) {
	t.Parallel()

	req := request()
	req.Overrides = migtemplate.Overrides{
		Env:        []string{"LOG_LEVEL=info", "PORT=3128"},
		SetEnv:     true,
		Secrets:    []string{"API_KEY=projects/statio-499700/secrets/api-key/versions/latest"},
		SetSecrets: true,
	}
	props := insertedProps(t, req)

	env, ok := metadataValue(props, "container-env")
	require.True(t, ok)
	require.Equal(t, "LOG_LEVEL=info\nPORT=3128", env)

	secrets, ok := metadataValue(props, "container-secrets")
	require.True(t, ok)
	require.Equal(t, "API_KEY=projects/statio-499700/secrets/api-key/versions/latest", secrets)

	image, ok := metadataValue(props, "container-image")
	require.True(t, ok)
	require.Equal(t, v2Image, image)
}

func TestInsertBodyClearsEnvWhenSetEmpty(t *testing.T) {
	t.Parallel()

	source := `{"properties": {"metadata": {"items": [
	  {"key": "container-image", "value": "` + v1Image + `"},
	  {"key": "container-env", "value": "OLD=1"}]}}}`
	req := request()
	req.Overrides.SetEnv = true
	raw, err := migtemplate.InsertBody(source, req)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "container-env")
}

func TestCreateValidatesOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		o    migtemplate.Overrides
		want string
	}{
		{"newline in value", migtemplate.Overrides{Env: []string{"A=x\ny"}, SetEnv: true}, "newline"},
		{"bad name", migtemplate.Overrides{Env: []string{"1A=x"}, SetEnv: true}, "invalid env entry"},
		{"no equals", migtemplate.Overrides{Secrets: []string{"API_KEY"}, SetSecrets: true}, "invalid env entry"},
		{"external ip", migtemplate.Overrides{ExternalIP: "yes"}, "true or false"},
	}
	for _, tt := range tests {
		req := request()
		req.Overrides = tt.o
		_, err := migtemplate.Create(req, fakeRunner(nil), &fakeCompute{})
		require.ErrorContains(t, err, tt.want, tt.name)
	}
}
