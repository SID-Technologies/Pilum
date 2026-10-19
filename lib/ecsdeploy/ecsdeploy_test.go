package ecsdeploy_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/ecsdeploy"

	"github.com/stretchr/testify/require"
)

const (
	registry = "123456789012.dkr.ecr.us-east-1.amazonaws.com"
	v1Image  = registry + "/api:v1"
	v2Image  = registry + "/api:v2"

	tdV41 = "arn:aws:ecs:us-east-1:123456789012:task-definition/api:41"
	tdV42 = "arn:aws:ecs:us-east-1:123456789012:task-definition/api:42"
)

// Trimmed from `aws ecs describe-task-definition --include TAGS`.
func taskDefinitionFixture(arn, image string) string {
	return `{
  "taskDefinition": {
    "taskDefinitionArn": "` + arn + `",
    "family": "api",
    "revision": 41,
    "status": "ACTIVE",
    "taskRoleArn": "arn:aws:iam::123456789012:role/api-task",
    "executionRoleArn": "arn:aws:iam::123456789012:role/api-exec",
    "networkMode": "awsvpc",
    "requiresCompatibilities": ["FARGATE"],
    "compatibilities": ["EC2", "FARGATE"],
    "requiresAttributes": [{"name": "com.amazonaws.ecs.capability.logging-driver.awslogs"}],
    "cpu": "256",
    "memory": "512",
    "registeredAt": "2026-10-01T10:00:00.000000+00:00",
    "registeredBy": "arn:aws:iam::123456789012:role/terraform",
    "containerDefinitions": [
      {
        "name": "api",
        "image": "` + image + `",
        "essential": true,
        "portMappings": [{"containerPort": 8080, "protocol": "tcp"}],
        "environment": [{"name": "B", "value": "2"}, {"name": "A", "value": "1"}],
        "secrets": [{"name": "DB_URL", "valueFrom": "api/db-url"}],
        "logConfiguration": {"logDriver": "awslogs", "options": {"awslogs-group": "/ecs/api"}}
      }
    ]
  },
  "tags": [{"key": "team", "value": "core"}]
}`
}

// Trimmed from `aws ecs describe-services`.
func serviceFixture(deployments ...string) string {
	return `{
  "services": [{
    "serviceName": "api",
    "status": "ACTIVE",
    "taskDefinition": "` + tdV41 + `",
    "desiredCount": 2,
    "runningCount": 2,
    "deploymentController": {"type": "ECS"},
    "deployments": [` + strings.Join(deployments, ",") + `],
    "events": [{"message": "(service api) has reached a steady state."}]
  }],
  "failures": []
}`
}

func deployment(status, td string, running int, rollout string) string {
	return fmt.Sprintf(`{"status": %q, "taskDefinition": %q, "desiredCount": 2, "runningCount": %d, "rolloutState": %q, "rolloutStateReason": "tasks failed to start"}`,
		status, td, running, rollout)
}

// fakeAWS answers aws ecs commands by subcommand and records them.
type fakeAWS struct {
	responses map[string][]string // subcommand -> answers, last one repeats
	calls     [][]string
}

func (f *fakeAWS) run(cmd []string) (string, error) {
	f.calls = append(f.calls, cmd)
	key := cmd[2]
	if key == "describe-task-definition" {
		key += " " + cmd[4]
	}
	answers := f.responses[key]
	if len(answers) == 0 {
		return "", fmt.Errorf("unexpected command: %s", strings.Join(cmd, " "))
	}
	answer := answers[0]
	if len(answers) > 1 {
		f.responses[key] = answers[1:]
	}
	return answer, nil
}

func (f *fakeAWS) called(sub string) [][]string {
	var out [][]string
	for _, c := range f.calls {
		if c[2] == sub {
			out = append(out, c)
		}
	}
	return out
}

func request() ecsdeploy.Request {
	return ecsdeploy.Request{
		Target:       aws.Target{Region: "us-east-1"},
		Cluster:      "prod",
		Service:      "api",
		Image:        v2Image,
		DesiredCount: -1,
	}
}

func decode(t *testing.T, input string) map[string]any {
	t.Helper()
	var def map[string]any
	require.NoError(t, json.Unmarshal([]byte(input), &def))
	return def
}

func container(def map[string]any) map[string]any {
	return def["containerDefinitions"].([]any)[0].(map[string]any)
}

func TestNextDefinitionChangesOnlyTheImage(t *testing.T) {
	t.Parallel()

	input, changed, err := ecsdeploy.NextDefinition(taskDefinitionFixture(tdV41, v1Image), request())
	require.NoError(t, err)
	require.True(t, changed)

	def := decode(t, input)
	for _, field := range []string{"taskDefinitionArn", "revision", "status", "compatibilities", "requiresAttributes", "registeredAt", "registeredBy"} {
		require.NotContains(t, def, field)
	}
	require.Equal(t, "api", def["family"])
	require.Equal(t, "256", def["cpu"])
	require.Equal(t, "arn:aws:iam::123456789012:role/api-task", def["taskRoleArn"])
	require.Equal(t, []any{map[string]any{"key": "team", "value": "core"}}, def["tags"])

	c := container(def)
	require.Equal(t, v2Image, c["image"])
	// Kept as the current revision has them.
	require.Len(t, c["environment"], 2)
	require.Equal(t, []any{map[string]any{"name": "DB_URL", "valueFrom": "api/db-url"}}, c["secrets"])
	require.Contains(t, c, "portMappings")
	require.Contains(t, c, "logConfiguration")
}

func TestNextDefinitionReplacesWhatTheServiceSets(t *testing.T) {
	t.Parallel()

	req := request()
	req.SetEnv = true
	req.Env = []string{"LOG_LEVEL=debug", "A=1"}
	req.SetSecrets = true
	req.Secrets = []string{"DB_URL=arn:aws:secretsmanager:us-east-1:123456789012:secret:api/db-url"}
	req.CPU = "512"
	req.Memory = "1024"

	input, changed, err := ecsdeploy.NextDefinition(taskDefinitionFixture(tdV41, v1Image), req)
	require.NoError(t, err)
	require.True(t, changed)

	def := decode(t, input)
	require.Equal(t, "512", def["cpu"])
	require.Equal(t, "1024", def["memory"])
	c := container(def)
	require.Equal(t, []any{
		map[string]any{"name": "A", "value": "1"},
		map[string]any{"name": "LOG_LEVEL", "value": "debug"},
	}, c["environment"])
	require.Equal(t, []any{
		map[string]any{"name": "DB_URL", "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:api/db-url"},
	}, c["secrets"])
}

func TestNextDefinitionUnchanged(t *testing.T) {
	t.Parallel()

	// Same image, same env in another order: nothing to register.
	req := request()
	req.Image = v1Image
	req.SetEnv = true
	req.Env = []string{"A=1", "B=2"}
	_, changed, err := ecsdeploy.NextDefinition(taskDefinitionFixture(tdV41, v1Image), req)
	require.NoError(t, err)
	require.False(t, changed)
}

func TestNextDefinitionPicksContainer(t *testing.T) {
	t.Parallel()

	twoContainers := `{"taskDefinition": {"family": "api", "containerDefinitions": [
	  {"name": "envoy", "image": "envoy:1"}, {"name": "web", "image": "web:1"}]}}`

	_, _, err := ecsdeploy.NextDefinition(twoContainers, request())
	require.ErrorContains(t, err, "2 containers (envoy, web); set ecs.container")

	req := request()
	req.Container = "web"
	input, _, err := ecsdeploy.NextDefinition(twoContainers, req)
	require.NoError(t, err)
	containers := decode(t, input)["containerDefinitions"].([]any)
	require.Equal(t, "envoy:1", containers[0].(map[string]any)["image"])
	require.Equal(t, v2Image, containers[1].(map[string]any)["image"])

	req.Container = "worker"
	_, _, err = ecsdeploy.NextDefinition(twoContainers, req)
	require.ErrorContains(t, err, `no container named "worker"`)

	// Two containers, one named like the service.
	named := strings.Replace(twoContainers, `"web"`, `"api"`, 1)
	_, _, err = ecsdeploy.NextDefinition(named, request())
	require.NoError(t, err)
}

func TestNextDefinitionRejectsBadPairs(t *testing.T) {
	t.Parallel()

	req := request()
	req.SetEnv = true
	req.Env = []string{"NOEQUALS"}
	_, _, err := ecsdeploy.NextDefinition(taskDefinitionFixture(tdV41, v1Image), req)
	require.ErrorContains(t, err, `--env "NOEQUALS": want NAME=VALUE`)
}

func TestDeployRegistersAndUpdates(t *testing.T) {
	t.Parallel()

	fake := &fakeAWS{responses: map[string][]string{
		"describe-services":                 {serviceFixture(deployment("PRIMARY", tdV41, 2, "COMPLETED"))},
		"describe-task-definition " + tdV41: {taskDefinitionFixture(tdV41, v1Image)},
		"register-task-definition":          {`{"taskDefinition": {"taskDefinitionArn": "` + tdV42 + `"}}`},
		"update-service":                    {`{"service": {}}`},
	}}

	req := request()
	req.DesiredCount = 3
	result, err := ecsdeploy.Deploy(req, fake.run)
	require.NoError(t, err)
	require.Equal(t, ecsdeploy.Result{Previous: tdV41, TaskDefinition: tdV42, Registered: true}, result)

	update := fake.called("update-service")
	require.Len(t, update, 1)
	joined := strings.Join(update[0], " ")
	require.Contains(t, joined, "--cluster prod --service api --task-definition "+tdV42)
	require.Contains(t, joined, "--desired-count 3")
	require.Contains(t, joined, "--region us-east-1")
}

func TestDeployReusesMatchingRevision(t *testing.T) {
	t.Parallel()

	// A rerun after register succeeded: the service already runs api:41
	// with the new image, so nothing is registered.
	fake := &fakeAWS{responses: map[string][]string{
		"describe-services":                 {serviceFixture(deployment("PRIMARY", tdV41, 2, "COMPLETED"))},
		"describe-task-definition " + tdV41: {taskDefinitionFixture(tdV41, v2Image)},
		"update-service":                    {`{"service": {}}`},
	}}

	result, err := ecsdeploy.Deploy(request(), fake.run)
	require.NoError(t, err)
	require.False(t, result.Registered)
	require.Equal(t, tdV41, result.TaskDefinition)
	require.Empty(t, fake.called("register-task-definition"))
	require.Len(t, fake.called("update-service"), 1)
}

func TestDeployRefusesWhatItCannotDeploy(t *testing.T) {
	t.Parallel()

	codeDeploy := strings.Replace(serviceFixture(), `"type": "ECS"`, `"type": "CODE_DEPLOY"`, 1)
	fake := &fakeAWS{responses: map[string][]string{"describe-services": {codeDeploy}}}
	_, err := ecsdeploy.Deploy(request(), fake.run)
	require.ErrorContains(t, err, "CODE_DEPLOY deployment controller")

	missing := `{"services": [], "failures": [{"arn": "arn:aws:ecs:us-east-1:123456789012:service/prod/api", "reason": "MISSING"}]}`
	fake = &fakeAWS{responses: map[string][]string{"describe-services": {missing}}}
	_, err = ecsdeploy.Deploy(request(), fake.run)
	require.ErrorContains(t, err, "service api: missing (create it first")

	draining := strings.Replace(serviceFixture(), `"status": "ACTIVE"`, `"status": "DRAINING"`, 1)
	fake = &fakeAWS{responses: map[string][]string{"describe-services": {draining}}}
	_, err = ecsdeploy.Deploy(request(), fake.run)
	require.ErrorContains(t, err, "service api is DRAINING")
}

// fakeClock advances by the sleep, so a wait runs instantly.
func fakeClock() (ecsdeploy.Clock, *int) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	sleeps := 0
	return ecsdeploy.Clock{
		Now:   func() time.Time { return now },
		Sleep: func(d time.Duration) { now = now.Add(d); sleeps++ },
	}, &sleeps
}

func waitRequest() ecsdeploy.WaitRequest {
	return ecsdeploy.WaitRequest{
		Target:   aws.Target{Region: "us-east-1"},
		Cluster:  "prod",
		Service:  "api",
		Image:    v2Image,
		Timeout:  time.Minute,
		Interval: 10 * time.Second,
	}
}

func TestWaitUntilRolledOut(t *testing.T) {
	t.Parallel()

	fake := &fakeAWS{responses: map[string][]string{
		"describe-services": {
			serviceFixture(deployment("PRIMARY", tdV42, 0, "IN_PROGRESS"), deployment("ACTIVE", tdV41, 2, "COMPLETED")),
			serviceFixture(deployment("PRIMARY", tdV42, 2, "IN_PROGRESS"), deployment("ACTIVE", tdV41, 1, "COMPLETED")),
			serviceFixture(deployment("PRIMARY", tdV42, 2, "COMPLETED")),
		},
		"describe-task-definition " + tdV42: {taskDefinitionFixture(tdV42, v2Image)},
	}}
	clock, sleeps := fakeClock()
	var progress []string

	err := ecsdeploy.Wait(waitRequest(), fake.run, clock, func(s string) { progress = append(progress, s) })
	require.NoError(t, err)
	require.Equal(t, 2, *sleeps)
	require.Equal(t, []string{"api:42: 0/2 tasks running", "api:42: 2/2 tasks running"}, progress)
	// A revision is described once, however many polls.
	require.Len(t, fake.called("describe-task-definition"), 1)
}

func TestWaitCatchesCircuitBreakerRollback(t *testing.T) {
	t.Parallel()

	// The breaker rolled back: the service is stable, on the old image.
	fake := &fakeAWS{responses: map[string][]string{
		"describe-services":                 {serviceFixture(deployment("PRIMARY", tdV41, 2, "COMPLETED"))},
		"describe-task-definition " + tdV41: {taskDefinitionFixture(tdV41, v1Image)},
	}}
	clock, _ := fakeClock()
	err := ecsdeploy.Wait(waitRequest(), fake.run, clock, func(string) {})
	require.ErrorContains(t, err, "deploying api:41, which doesn't run "+v2Image)
	require.ErrorContains(t, err, "circuit breaker rolled back")
	require.ErrorContains(t, err, "has reached a steady state")
}

func TestWaitFailedRollout(t *testing.T) {
	t.Parallel()

	fake := &fakeAWS{responses: map[string][]string{
		"describe-services":                 {serviceFixture(deployment("PRIMARY", tdV42, 0, "FAILED"))},
		"describe-task-definition " + tdV42: {taskDefinitionFixture(tdV42, v2Image)},
	}}
	clock, _ := fakeClock()
	err := ecsdeploy.Wait(waitRequest(), fake.run, clock, func(string) {})
	require.ErrorContains(t, err, "deployment of api:42 failed: tasks failed to start")
}

func TestWaitTimesOut(t *testing.T) {
	t.Parallel()

	fake := &fakeAWS{responses: map[string][]string{
		"describe-services":                 {serviceFixture(deployment("PRIMARY", tdV42, 1, "IN_PROGRESS"))},
		"describe-task-definition " + tdV42: {taskDefinitionFixture(tdV42, v2Image)},
	}}
	clock, sleeps := fakeClock()
	err := ecsdeploy.Wait(waitRequest(), fake.run, clock, func(string) {})
	require.ErrorContains(t, err, "didn't finish deploying within 1m0s (api:42: 1/2 tasks running)")
	require.Equal(t, 6, *sleeps)
}

func TestFamily(t *testing.T) {
	t.Parallel()

	require.Equal(t, "api:42", ecsdeploy.Family(tdV42))
	require.Equal(t, "api:42", ecsdeploy.Family("api:42"))
}
