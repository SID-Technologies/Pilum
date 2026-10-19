package ecsdeploy

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/errors"
)

// Service is the part of describe-services output Pilum reads.
type Service struct {
	ServiceName          string `json:"serviceName"`
	Status               string `json:"status"`
	TaskDefinition       string `json:"taskDefinition"`
	DesiredCount         int    `json:"desiredCount"`
	RunningCount         int    `json:"runningCount"`
	DeploymentController struct {
		Type string `json:"type"`
	} `json:"deploymentController"`
	Deployments []Deployment `json:"deployments"`
	Events      []struct {
		Message string `json:"message"`
	} `json:"events"`
}

// Deployment is one of a service's deployments. The PRIMARY one is the
// task definition the service is moving to (or runs, once it's done).
type Deployment struct {
	Status             string `json:"status"`
	TaskDefinition     string `json:"taskDefinition"`
	DesiredCount       int    `json:"desiredCount"`
	RunningCount       int    `json:"runningCount"`
	PendingCount       int    `json:"pendingCount"`
	FailedTasks        int    `json:"failedTasks"`
	RolloutState       string `json:"rolloutState"`
	RolloutStateReason string `json:"rolloutStateReason"`
}

// Primary returns the service's PRIMARY deployment.
func (s Service) Primary() (Deployment, bool) {
	idx := slices.IndexFunc(s.Deployments, func(d Deployment) bool { return d.Status == "PRIMARY" })
	if idx == -1 {
		return Deployment{}, false
	}
	return s.Deployments[idx], true
}

// ParseService reads one service from describe-services output. A missing
// or deleted service is an error: Pilum deploys to services, it doesn't
// create them.
func ParseService(describedJSON, name string) (Service, error) {
	var described struct {
		Services []Service `json:"services"`
		Failures []struct {
			Arn    string `json:"arn"`
			Reason string `json:"reason"`
		} `json:"failures"`
	}
	err := json.Unmarshal([]byte(describedJSON), &described)
	if err != nil {
		return Service{}, errors.Wrap(err, "invalid describe-services JSON")
	}
	if len(described.Failures) > 0 {
		return Service{}, errors.New("service %s: %s (create it first, e.g. with Terraform; Pilum deploys to existing services)",
			name, strings.ToLower(described.Failures[0].Reason))
	}
	if len(described.Services) == 0 {
		return Service{}, errors.New("service %s not found", name)
	}
	svc := described.Services[0]
	if svc.Status != "ACTIVE" {
		return Service{}, errors.New("service %s is %s", name, svc.Status)
	}
	return svc, nil
}

// DescribeService describes one ECS service.
func DescribeService(target aws.Target, cluster, service string, run Runner) (Service, error) {
	described, err := run(aws.DescribeECSServiceCommand(target, cluster, service))
	if err != nil {
		return Service{}, errors.Wrap(err, "describing service %s in cluster %s", service, cluster)
	}
	return ParseService(described, service)
}

// WaitRequest is the rollout to wait for.
type WaitRequest struct {
	aws.Target
	Cluster  string
	Service  string
	Image    string // The image the service should end up running
	Timeout  time.Duration
	Interval time.Duration
}

// Clock lets tests run a wait without sleeping.
type Clock struct {
	Now   func() time.Time
	Sleep func(time.Duration)
}

// Wait blocks until the service's deployment of req.Image has finished:
// one deployment left, all its tasks running. It fails as soon as the
// rollout fails or the service turns out to run something else, which is
// what a deployment circuit breaker rollback looks like. `aws ecs wait
// services-stable` reports that as success, since the service is stable,
// just on the old version.
func Wait(req WaitRequest, run Runner, clock Clock, progress func(string)) error {
	deadline := clock.Now().Add(req.Timeout)
	images := map[string]bool{}
	last := ""

	for {
		svc, err := DescribeService(req.Target, req.Cluster, req.Service, run)
		if err != nil {
			return err
		}
		primary, ok := svc.Primary()
		if !ok {
			return errors.New("service %s has no primary deployment", req.Service)
		}

		runsImage, err := definitionRuns(req.Target, primary.TaskDefinition, req.Image, images, run)
		if err != nil {
			return err
		}
		if !runsImage {
			return errors.New("service %s is deploying %s, which doesn't run %s; the deployment circuit breaker rolled back, "+
				"or another deploy replaced this one%s", req.Service, Family(primary.TaskDefinition), req.Image, recentEvents(svc))
		}
		if primary.RolloutState == "FAILED" {
			return errors.New("deployment of %s failed: %s%s", Family(primary.TaskDefinition), primary.RolloutStateReason, recentEvents(svc))
		}
		if len(svc.Deployments) == 1 && primary.RunningCount == primary.DesiredCount &&
			(primary.RolloutState == "" || primary.RolloutState == "COMPLETED") {
			return nil
		}

		status := fmt.Sprintf("%s: %d/%d tasks running", Family(primary.TaskDefinition), primary.RunningCount, primary.DesiredCount)
		if primary.FailedTasks > 0 {
			status += fmt.Sprintf(", %d failed", primary.FailedTasks)
		}
		if status != last {
			progress(status)
			last = status
		}

		if !clock.Now().Before(deadline) {
			return errors.New("service %s didn't finish deploying within %s (%s)%s", req.Service, req.Timeout, status, recentEvents(svc))
		}
		clock.Sleep(req.Interval)
	}
}

// definitionRuns reports whether a task definition has a container running
// image. Answers are cached: a revision never changes.
func definitionRuns(target aws.Target, taskDefinition, image string, cache map[string]bool, run Runner) (bool, error) {
	known, ok := cache[taskDefinition]
	if ok {
		return known, nil
	}
	described, err := run(aws.DescribeTaskDefinitionCommand(target, taskDefinition))
	if err != nil {
		return false, errors.Wrap(err, "describing task definition %s", taskDefinition)
	}
	images, err := ContainerImages(described)
	if err != nil {
		return false, err
	}
	runs := slices.Contains(images, image)
	cache[taskDefinition] = runs
	return runs, nil
}

// ContainerImages lists the images of a described task definition's containers.
func ContainerImages(describedJSON string) ([]string, error) {
	var described struct {
		TaskDefinition struct {
			ContainerDefinitions []struct {
				Name  string `json:"name"`
				Image string `json:"image"`
			} `json:"containerDefinitions"`
		} `json:"taskDefinition"`
	}
	err := json.Unmarshal([]byte(describedJSON), &described)
	if err != nil {
		return nil, errors.Wrap(err, "invalid task definition JSON")
	}
	images := make([]string, 0, len(described.TaskDefinition.ContainerDefinitions))
	for _, c := range described.TaskDefinition.ContainerDefinitions {
		images = append(images, c.Image)
	}
	return images, nil
}

// recentEvents formats the service's latest events (newest first), which
// is where ECS says why tasks didn't start or failed health checks.
func recentEvents(svc Service) string {
	const maxEvents = 3
	var lines []string
	for i := 0; i < len(svc.Events) && i < maxEvents; i++ {
		lines = append(lines, "  "+svc.Events[i].Message)
	}
	if len(lines) == 0 {
		return ""
	}
	return "\nrecent service events:\n" + strings.Join(lines, "\n")
}
