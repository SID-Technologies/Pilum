package aws

import (
	"slices"
	"strconv"

	"github.com/sid-technologies/pilum/lib/configutil"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// DefaultECSWaitTimeout is how long a deploy waits for the rollout, in seconds.
const DefaultECSWaitTimeout = 600

// ECSConfig is the ecs: block of an aws-ecs service.
type ECSConfig struct {
	Cluster string
	// Service is the ECS service name. Defaults to the Pilum service name.
	Service string
	// Container is the container in the task definition that runs the new
	// image. Empty picks the only container, or the one named like the
	// service.
	Container string
	// CPU and Memory are task-level sizes in ECS units ("512", "1024"), or
	// "" to keep the current task definition's.
	CPU    string
	Memory string
	// DesiredCount sets the service's task count on deploy. -1 leaves it
	// alone, which is what you want when autoscaling owns it.
	DesiredCount int
	// WaitTimeout is how long, in seconds, to wait for the rollout.
	WaitTimeout int
	// LogGroup is the CloudWatch log group `pilum logs` tails.
	LogGroup string
}

// ParseECSConfig reads the ecs: block of a service.
func ParseECSConfig(svc serviceinfo.ServiceInfo) ECSConfig {
	ecsMap := configutil.MapFromAny(svc.Config["ecs"])
	service := configutil.GetString(ecsMap, "service", svc.Name)
	return ECSConfig{
		Cluster:      configutil.GetString(ecsMap, "cluster", ""),
		Service:      service,
		Container:    configutil.GetString(ecsMap, "container", ""),
		CPU:          configutil.GetString(ecsMap, "cpu", ""),
		Memory:       configutil.GetString(ecsMap, "memory", ""),
		DesiredCount: configutil.GetInt(ecsMap, "desired_count", -1),
		WaitTimeout:  configutil.GetInt(ecsMap, "wait_timeout", DefaultECSWaitTimeout),
		LogGroup:     configutil.GetString(ecsMap, "log_group", "/ecs/"+service),
	}
}

// GenerateECSDeployArgs returns the arguments of the hidden `pilum
// ecs-deploy` command (everything after the executable). Env vars and
// secrets are only passed when the service sets them; otherwise the task
// definition keeps its own.
func GenerateECSDeployArgs(svc serviceinfo.ServiceInfo, image string) []string {
	cfg := ParseECSConfig(svc)
	args := []string{"ecs-deploy",
		"--cluster", cfg.Cluster,
		"--service", cfg.Service,
		"--image", image,
	}
	args = append(args, TargetOf(svc).Flags()...)

	optional := []struct{ flag, value string }{
		{"--container", cfg.Container},
		{"--cpu", cfg.CPU},
		{"--memory", cfg.Memory},
	}
	for _, o := range optional {
		if o.value != "" {
			args = append(args, o.flag, o.value)
		}
	}
	if cfg.DesiredCount >= 0 {
		args = append(args, "--desired-count", strconv.Itoa(cfg.DesiredCount))
	}

	env := make([]string, 0, len(svc.EnvVars))
	for _, e := range svc.EnvVars {
		env = append(env, e.Name+"="+e.Value)
	}
	slices.Sort(env)
	for _, e := range env {
		args = append(args, "--env", e)
	}

	secrets := make([]string, 0, len(svc.Secrets))
	for _, sec := range svc.Secrets {
		secrets = append(secrets, sec.Name+"="+sec.Value)
	}
	slices.Sort(secrets)
	for _, sec := range secrets {
		args = append(args, "--secret", sec)
	}

	return args
}

// GenerateECSWaitArgs returns the arguments of the hidden `pilum ecs-wait`
// command, which blocks until the service runs image.
func GenerateECSWaitArgs(svc serviceinfo.ServiceInfo, image string) []string {
	cfg := ParseECSConfig(svc)
	args := []string{"ecs-wait",
		"--cluster", cfg.Cluster,
		"--service", cfg.Service,
		"--image", image,
		"--wait-timeout", strconv.Itoa(cfg.WaitTimeout),
	}
	return append(args, TargetOf(svc).Flags()...)
}

// DescribeECSServiceCommand describes one ECS service as JSON.
func DescribeECSServiceCommand(target Target, cluster, service string) []string {
	cmd := []string{"aws", "ecs", "describe-services", "--cluster", cluster, "--services", service, "--output", "json"}
	return append(cmd, target.Flags()...)
}

// DescribeTaskDefinitionCommand describes a task definition, with its tags.
func DescribeTaskDefinitionCommand(target Target, taskDefinition string) []string {
	cmd := []string{"aws", "ecs", "describe-task-definition", "--task-definition", taskDefinition,
		"--include", "TAGS", "--output", "json"}
	return append(cmd, target.Flags()...)
}

// RegisterTaskDefinitionCommand registers a task definition from its JSON.
func RegisterTaskDefinitionCommand(target Target, inputJSON string) []string {
	cmd := []string{"aws", "ecs", "register-task-definition", "--cli-input-json", inputJSON, "--output", "json"}
	return append(cmd, target.Flags()...)
}

// UpdateECSServiceCommand points a service at a task definition, which
// starts a deployment. desiredCount < 0 leaves the count alone.
func UpdateECSServiceCommand(target Target, cluster, service, taskDefinition string, desiredCount int) []string {
	cmd := []string{"aws", "ecs", "update-service", "--cluster", cluster, "--service", service,
		"--task-definition", taskDefinition, "--output", "json"}
	if desiredCount >= 0 {
		cmd = append(cmd, "--desired-count", strconv.Itoa(desiredCount))
	}
	return append(cmd, target.Flags()...)
}

// GenerateECSStatusCommand describes the service for `pilum status`.
func GenerateECSStatusCommand(svc serviceinfo.ServiceInfo) []string {
	cfg := ParseECSConfig(svc)
	return DescribeECSServiceCommand(TargetOf(svc), cfg.Cluster, cfg.Service)
}

// GenerateECSLogsCommand tails the service's CloudWatch log group.
func GenerateECSLogsCommand(svc serviceinfo.ServiceInfo, follow bool) []string {
	cmd := withTarget([]string{"aws", "logs", "tail", ParseECSConfig(svc).LogGroup}, svc)
	if follow {
		cmd = append(cmd, "--follow")
	}
	return cmd
}
