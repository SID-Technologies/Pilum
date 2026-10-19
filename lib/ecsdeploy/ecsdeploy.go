// Package ecsdeploy rolls a new image onto an existing ECS service.
//
// Terraform (or whoever set the service up) owns the cluster, the load
// balancer, the IAM roles and the service itself. Pilum ships new versions:
// it copies the task definition the service runs now, changes the image
// and only the settings pilum.yaml sets, registers that as the next
// revision, and points the service at it.
//
// The aws CLI can't do the copy on its own (register-task-definition takes
// the whole definition, not a diff), so the current one is read as JSON,
// edited, and registered back. Copying keeps everything Pilum doesn't know
// about: ports, health checks, log configuration, mount points, sidecars.
package ecsdeploy

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/errors"
)

// Runner executes an aws command and returns its stdout.
type Runner func(cmd []string) (string, error)

// Request identifies the service and what its next revision changes.
type Request struct {
	aws.Target
	Cluster   string
	Service   string
	Container string // Empty picks the only container, or the one named like the service
	Image     string

	CPU          string // Task-level, "" keeps the current one
	Memory       string // Task-level, "" keeps the current one
	DesiredCount int    // -1 keeps the current count

	Env        []string // KEY=VALUE
	SetEnv     bool     // Replace the container's environment (otherwise kept as is)
	Secrets    []string // NAME=Secrets Manager or SSM parameter name or ARN
	SetSecrets bool     // Replace the container's secrets (otherwise kept as is)
}

// Result says what Deploy did.
type Result struct {
	Previous       string // Task definition the service ran before
	TaskDefinition string // Task definition the service runs now
	Registered     bool   // False when the current revision already matched
}

// readOnlyFields are in describe-task-definition's output but rejected by
// register-task-definition.
var readOnlyFields = []string{
	"taskDefinitionArn", "revision", "status", "requiresAttributes",
	"compatibilities", "registeredAt", "registeredBy", "deregisteredAt",
}

// Deploy registers the service's next task definition and starts a
// deployment of it. A rerun after a failure picks up where it stopped: a
// revision that already matches is reused rather than registered again.
func Deploy(req Request, run Runner) (Result, error) {
	if req.Cluster == "" || req.Service == "" || req.Image == "" {
		return Result{}, errors.New("cluster, service and image are required")
	}

	svc, err := DescribeService(req.Target, req.Cluster, req.Service, run)
	if err != nil {
		return Result{}, err
	}
	if svc.DeploymentController.Type != "" && svc.DeploymentController.Type != "ECS" {
		return Result{}, errors.New("service %s uses the %s deployment controller; Pilum deploys only ECS rolling and blue/green services",
			req.Service, svc.DeploymentController.Type)
	}
	result := Result{Previous: svc.TaskDefinition, TaskDefinition: svc.TaskDefinition}

	described, err := run(aws.DescribeTaskDefinitionCommand(req.Target, svc.TaskDefinition))
	if err != nil {
		return Result{}, errors.Wrap(err, "describing task definition %s", svc.TaskDefinition)
	}
	input, changed, err := NextDefinition(described, req)
	if err != nil {
		return Result{}, errors.Wrap(err, "task definition %s", svc.TaskDefinition)
	}

	if changed {
		registered, runErr := run(aws.RegisterTaskDefinitionCommand(req.Target, input))
		if runErr != nil {
			return Result{}, errors.Wrap(runErr, "registering the next revision of %s", Family(svc.TaskDefinition))
		}
		arn, parseErr := registeredARN(registered)
		if parseErr != nil {
			return Result{}, parseErr
		}
		result.TaskDefinition = arn
		result.Registered = true
	}

	// Also when nothing changed: a desired count to apply, or a rerun after
	// the register succeeded and the update didn't.
	_, err = run(aws.UpdateECSServiceCommand(req.Target, req.Cluster, req.Service, result.TaskDefinition, req.DesiredCount))
	if err != nil {
		return Result{}, errors.Wrap(err, "updating service %s", req.Service)
	}
	return result, nil
}

// NextDefinition turns describe-task-definition output into the input for
// the next revision. changed is false when the next revision would be the
// same as the current one.
func NextDefinition(describedJSON string, req Request) (string, bool, error) {
	var described struct {
		TaskDefinition map[string]any `json:"taskDefinition"`
		Tags           []any          `json:"tags"`
	}
	err := json.Unmarshal([]byte(describedJSON), &described)
	if err != nil {
		return "", false, errors.Wrap(err, "invalid task definition JSON")
	}
	if described.TaskDefinition == nil {
		return "", false, errors.New("describe-task-definition returned no task definition")
	}

	current := described.TaskDefinition
	for _, field := range readOnlyFields {
		delete(current, field)
	}
	next, err := deepCopy(current)
	if err != nil {
		return "", false, err
	}

	container, err := pickContainer(next, req.Container, req.Service)
	if err != nil {
		return "", false, err
	}
	container["image"] = req.Image
	if req.SetEnv {
		env, envErr := nameValues(req.Env, "value", "--env")
		if envErr != nil {
			return "", false, envErr
		}
		container["environment"] = env
	}
	if req.SetSecrets {
		secrets, secretErr := nameValues(req.Secrets, "valueFrom", "--secret")
		if secretErr != nil {
			return "", false, secretErr
		}
		container["secrets"] = secrets
	}
	if req.CPU != "" {
		next["cpu"] = req.CPU
	}
	if req.Memory != "" {
		next["memory"] = req.Memory
	}

	changed := !reflect.DeepEqual(normalize(current), normalize(next))

	// Tags come back next to the definition, not in it, and an empty list
	// is rejected.
	if len(described.Tags) > 0 {
		next["tags"] = described.Tags
	}
	input, err := json.Marshal(next)
	if err != nil {
		return "", false, errors.Wrap(err, "encoding task definition")
	}
	return string(input), changed, nil
}

// pickContainer finds the container that runs the new image: the one
// named, the only one, or the one named like the service.
func pickContainer(def map[string]any, name, service string) (map[string]any, error) {
	containers := containerList(def)
	names := make([]string, 0, len(containers))
	for _, c := range containers {
		names = append(names, entryName(c))
	}
	if len(containers) == 0 {
		return nil, errors.New("task definition has no containers")
	}

	want := name
	if want == "" {
		if len(containers) == 1 {
			return containers[0], nil
		}
		want = service
	}
	idx := slices.Index(names, want)
	if idx == -1 {
		if name == "" {
			return nil, errors.New("task definition has %d containers (%s); set ecs.container to the one that runs this image",
				len(containers), strings.Join(names, ", "))
		}
		return nil, errors.New("no container named %q in the task definition (has %s)", name, strings.Join(names, ", "))
	}
	return containers[idx], nil
}

// nameValues turns KEY=VALUE pairs into ECS's [{name, <valueKey>}] list,
// sorted by name so equal settings compare equal.
func nameValues(pairs []string, valueKey, flag string) ([]any, error) {
	sorted := slices.Clone(pairs)
	slices.Sort(sorted)
	out := make([]any, 0, len(sorted))
	for _, pair := range sorted {
		name, value, found := strings.Cut(pair, "=")
		if !found || name == "" {
			return nil, errors.New("%s %q: want NAME=VALUE", flag, pair)
		}
		out = append(out, map[string]any{"name": name, valueKey: value})
	}
	return out, nil
}

// normalize sorts each container's environment and secrets by name, which
// ECS doesn't keep in order, so reordering alone isn't a change.
func normalize(def map[string]any) map[string]any {
	copied, err := deepCopy(def)
	if err != nil {
		return def
	}
	for _, c := range containerList(copied) {
		for _, key := range []string{"environment", "secrets"} {
			entries, ok := c[key].([]any)
			if !ok || len(entries) == 0 {
				delete(c, key)
				continue
			}
			slices.SortFunc(entries, func(a, b any) int {
				return strings.Compare(entryName(a), entryName(b))
			})
		}
	}
	return copied
}

// containerList returns a task definition's container definitions.
func containerList(def map[string]any) []map[string]any {
	list, ok := def["containerDefinitions"].([]any)
	if !ok {
		return nil
	}
	containers := make([]map[string]any, 0, len(list))
	for _, item := range list {
		c, isMap := item.(map[string]any)
		if isMap {
			containers = append(containers, c)
		}
	}
	return containers
}

// entryName returns the name of a container or a name/value entry.
func entryName(entry any) string {
	m, ok := entry.(map[string]any)
	if !ok {
		return ""
	}
	name, ok := m["name"].(string)
	if !ok {
		return ""
	}
	return name
}

func deepCopy(m map[string]any) (map[string]any, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, errors.Wrap(err, "copying task definition")
	}
	var out map[string]any
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, errors.Wrap(err, "copying task definition")
	}
	return out, nil
}

func registeredARN(registerJSON string) (string, error) {
	var registered struct {
		TaskDefinition struct {
			TaskDefinitionArn string `json:"taskDefinitionArn"`
		} `json:"taskDefinition"`
	}
	err := json.Unmarshal([]byte(registerJSON), &registered)
	if err != nil {
		return "", errors.Wrap(err, "invalid register-task-definition JSON")
	}
	if registered.TaskDefinition.TaskDefinitionArn == "" {
		return "", errors.New("register-task-definition returned no ARN")
	}
	return registered.TaskDefinition.TaskDefinitionArn, nil
}

// Family returns "family:revision" from a task definition ARN, which is
// how ECS names a revision in its console and CLI.
func Family(arn string) string {
	_, after, found := strings.Cut(arn, ":task-definition/")
	if found {
		return after
	}
	return arn
}
