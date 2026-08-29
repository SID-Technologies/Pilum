package gcp

import (
	"strings"
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

func deployCmd(cloudRun map[string]any) string {
	svc := serviceinfo.ServiceInfo{
		Name:    "statio-mcp-example",
		Region:  "us-central1",
		Project: "statio-499700",
		Config:  map[string]any{"cloud_run": cloudRun},
	}

	return strings.Join(GenerateGCPDeployCommand(svc, "img:latest"), " ")
}

// The bug this pins: `ingress` was read by NOBODY. Manifests declared
// `ingress: internal` and the deploy never passed the flag, so services marked
// internal were reachable from the internet and the manifest said otherwise.
func TestIngressIsPassedToGcloud(t *testing.T) {
	t.Parallel()

	got := deployCmd(map[string]any{"ingress": "internal"})
	if !strings.Contains(got, "--ingress internal") {
		t.Fatalf("ingress not passed to gcloud: %s", got)
	}
}

// An absent ingress must leave the service alone. Emitting a default would
// reopen a service someone closed by hand — the opposite failure, and a
// harder one to notice.
func TestAbsentIngressEmitsNoFlag(t *testing.T) {
	t.Parallel()

	if got := deployCmd(map[string]any{}); strings.Contains(got, "--ingress") {
		t.Fatalf("emitted --ingress with none configured: %s", got)
	}
}

func TestVPCConnectorIsPassedToGcloud(t *testing.T) {
	t.Parallel()

	got := deployCmd(map[string]any{"vpc_connector": "mcp-connector"})
	if !strings.Contains(got, "--vpc-connector mcp-connector") {
		t.Fatalf("connector not passed: %s", got)
	}
}

// The default that matters. gcloud's own default is private-ranges-only, which
// sends RFC1918 through the VPC and lets PUBLIC traffic bypass it — so a
// firewall hung off the connector never sees the egress it exists to block.
// Naming a connector without saying which traffic it carries must mean all.
func TestConnectorWithoutEgressDefaultsToAllTraffic(t *testing.T) {
	t.Parallel()

	got := deployCmd(map[string]any{"vpc_connector": "mcp-connector"})
	if !strings.Contains(got, "--vpc-egress all-traffic") {
		t.Fatalf("expected all-traffic default, got: %s", got)
	}
}

func TestExplicitVPCEgressWins(t *testing.T) {
	t.Parallel()

	got := deployCmd(map[string]any{
		"vpc_connector": "mcp-connector",
		"vpc_egress":    "private-ranges-only",
	})
	if !strings.Contains(got, "--vpc-egress private-ranges-only") {
		t.Fatalf("explicit egress ignored: %s", got)
	}
}

// vpc_egress alone does nothing — there is no connector to carry the traffic.
// Emitting the flag without one makes gcloud reject the deploy.
func TestEgressWithoutConnectorEmitsNothing(t *testing.T) {
	t.Parallel()

	if got := deployCmd(map[string]any{"vpc_egress": "all-traffic"}); strings.Contains(got, "--vpc-egress") {
		t.Fatalf("emitted --vpc-egress with no connector: %s", got)
	}
}

// Service-level flags must precede the first --container, or gcloud rejects
// the invocation in multi-container mode.
func TestNetworkFlagsComeBeforeContainers(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{
		Name:    "statio-mcp-example",
		Region:  "us-central1",
		Project: "statio-499700",
		Config: map[string]any{"cloud_run": map[string]any{
			"ingress": "internal", "vpc_connector": "mcp-connector",
		}},
		Sidecars: []serviceinfo.Sidecar{{Name: "proxy", Image: "sidecar:latest"}},
	}

	cmd := GenerateGCPDeployCommand(svc, "img:latest")

	first := -1
	for i, a := range cmd {
		if a == "--container" {
			first = i

			break
		}
	}

	if first == -1 {
		t.Skip("no --container emitted; ordering does not apply")
	}

	for i, a := range cmd {
		if (a == "--ingress" || a == "--vpc-connector" || a == "--vpc-egress") && i > first {
			t.Fatalf("%s appears after --container at %d: %v", a, first, cmd)
		}
	}
}
