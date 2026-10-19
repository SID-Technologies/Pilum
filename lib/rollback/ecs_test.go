package rollback

import (
	"fmt"
	"strings"
	"testing"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

const ecsTD = "arn:aws:ecs:us-east-1:123456789012:task-definition/"

func ecsRevisionJSON(rev int, image string, failed bool) string {
	tags := `[]`
	if failed {
		tags = `[{"key": "pilum:rollout", "value": "failed"}]`
	}
	return fmt.Sprintf(`{"taskDefinition": {"taskDefinitionArn": "%sapi:%d", "revision": %d,
	  "containerDefinitions": [{"name": "envoy", "image": "envoy:1"}, {"name": "api", "image": %q}]}, "tags": %s}`,
		ecsTD, rev, rev, image, tags)
}

// ecsResponses is an ECS service on api:43 whose earlier revisions are
// 42 (failed rollout), 41 and 40, plus a longer family that shares the prefix.
func ecsResponses() map[string]string {
	return map[string]string{
		"describe-services": `{"services": [{"serviceName": "api", "status": "ACTIVE",
		  "taskDefinition": "` + ecsTD + `api:43", "deploymentController": {"type": "ECS"}}], "failures": []}`,
		"list-task-definitions": `{"taskDefinitionArns": ["` + ecsTD + `api-worker:9", "` + ecsTD + `api:43", "` +
			ecsTD + `api:42", "` + ecsTD + `api:41", "` + ecsTD + `api:40"]}`,
		"task-definition/api:43 ": ecsRevisionJSON(43, "repo/api:v4", false),
		"task-definition/api:42 ": ecsRevisionJSON(42, "repo/api:v3", true),
		"task-definition/api:41 ": ecsRevisionJSON(41, "repo/api:v2", false),
		"task-definition/api:40 ": ecsRevisionJSON(40, "repo/api:v1", false),
	}
}

func ecsRollbackService() serviceinfo.ServiceInfo {
	return *serviceinfo.NewServiceInfo(map[string]any{
		"name": "api", "provider": "aws", "type": "aws-ecs", "region": "us-east-1",
		"ecs": map[string]any{"cluster": "prod", "wait_timeout": 300},
	}, "api")
}

func TestPlanECSSkipsFailedRollouts(t *testing.T) {
	t.Parallel()

	kind, ok := KindFor(ecsRollbackService())
	require.True(t, ok)
	require.Equal(t, KindECS, kind)

	plan, err := NewPlan(ecsRollbackService(), "", fakeRunner(ecsResponses()))
	require.NoError(t, err)
	require.Equal(t, "api:43", plan.From)
	require.Equal(t, "api:41", plan.To)
	// The service's container, not the sidecar.
	require.Equal(t, "repo/api:v2", plan.ToImage)

	cmd := strings.Join(plan.Command, " ")
	require.Contains(t, cmd, "aws ecs update-service --cluster prod --service api --task-definition "+ecsTD+"api:41")
	require.NotContains(t, cmd, "--desired-count")
	require.Contains(t, plan.Wait, "ecs-wait")
	require.Contains(t, plan.Wait, "repo/api:v2")
	require.Equal(t, 300+ecsWaitGrace, plan.WaitTimeout)
}

func TestPlanECSTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		to   string
		want string
		err  string
	}{
		{to: "40", want: "api:40"},
		{to: "api:40", want: "api:40"},
		{to: ecsTD + "api:40", want: "api:40"},
		{to: "v1", want: "api:40"},
		// Asked for by name, a failed revision is allowed.
		{to: "42", want: "api:42"},
		{to: "43", err: "api:43 is already deployed"},
		{to: "v4", err: "v4 (api:43) is already deployed"},
		{to: "39", err: "no active revision api:39"},
		{to: "v9", err: `runs a "v9" image`},
	}
	for _, tt := range tests {
		plan, err := NewPlan(ecsRollbackService(), tt.to, fakeRunner(ecsResponses()))
		if tt.err != "" {
			require.ErrorContains(t, err, tt.err, tt.to)
			continue
		}
		require.NoError(t, err, tt.to)
		require.Equal(t, tt.want, plan.To, tt.to)
	}
}

func TestPlanECSNothingEarlier(t *testing.T) {
	t.Parallel()

	responses := ecsResponses()
	responses["list-task-definitions"] = `{"taskDefinitionArns": ["` + ecsTD + `api:43", "` + ecsTD + `api:42"]}`
	_, err := NewPlan(ecsRollbackService(), "", fakeRunner(responses))
	require.ErrorContains(t, err, "no earlier revision of api than api:43 to roll back to (failed rollouts are skipped)")
}
