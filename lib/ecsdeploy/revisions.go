package ecsdeploy

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/errors"
)

// Revision is one task definition revision.
type Revision struct {
	ARN        string
	Number     int
	Images     []string // In container order
	Containers []string // Names, in the same order
	// Failed is true when a Pilum deploy of this revision failed or was
	// rolled back (tagged pilum:rollout=failed).
	Failed bool
}

// RolloutError is a deployment that failed or was rolled back, as opposed
// to one that ran out of time and may still finish. TaskDefinition is the
// revision that failed, or "" when Pilum couldn't tell which.
type RolloutError struct {
	TaskDefinition string
	msg            string
}

func (e *RolloutError) Error() string { return e.msg }

// Revisions lists a family's active revisions, newest first.
func Revisions(target aws.Target, family string, run Runner) ([]string, error) {
	listed, err := run(aws.ListTaskDefinitionsCommand(target, family))
	if err != nil {
		return nil, errors.Wrap(err, "listing revisions of %s", family)
	}
	return ParseRevisionList(listed, family)
}

// ParseRevisionList reads list-task-definitions output, keeping only the
// family's own revisions (the CLI filters by prefix) in the order given.
func ParseRevisionList(listJSON, family string) ([]string, error) {
	var listed struct {
		TaskDefinitionArns []string `json:"taskDefinitionArns"`
	}
	err := json.Unmarshal([]byte(listJSON), &listed)
	if err != nil {
		return nil, errors.Wrap(err, "invalid list-task-definitions JSON")
	}
	var arns []string
	for _, arn := range listed.TaskDefinitionArns {
		name, _, _ := strings.Cut(Family(arn), ":")
		if name == family {
			arns = append(arns, arn)
		}
	}
	return arns, nil
}

// DescribeRevision reads a revision's images and whether it's marked failed.
func DescribeRevision(target aws.Target, arn string, run Runner) (Revision, error) {
	described, err := run(aws.DescribeTaskDefinitionCommand(target, arn))
	if err != nil {
		return Revision{}, errors.Wrap(err, "describing task definition %s", Family(arn))
	}
	return ParseRevision(described)
}

// ParseRevision reads describe-task-definition output into a Revision.
func ParseRevision(describedJSON string) (Revision, error) {
	var described struct {
		TaskDefinition struct {
			TaskDefinitionArn    string `json:"taskDefinitionArn"`
			Revision             int    `json:"revision"`
			ContainerDefinitions []struct {
				Name  string `json:"name"`
				Image string `json:"image"`
			} `json:"containerDefinitions"`
		} `json:"taskDefinition"`
		Tags []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"tags"`
	}
	err := json.Unmarshal([]byte(describedJSON), &described)
	if err != nil {
		return Revision{}, errors.Wrap(err, "invalid task definition JSON")
	}
	td := described.TaskDefinition
	rev := Revision{ARN: td.TaskDefinitionArn, Number: td.Revision}
	if rev.Number == 0 {
		rev.Number = RevisionNumber(td.TaskDefinitionArn)
	}
	for _, c := range td.ContainerDefinitions {
		rev.Images = append(rev.Images, c.Image)
		rev.Containers = append(rev.Containers, c.Name)
	}
	for _, tag := range described.Tags {
		if tag.Key == aws.FailedRolloutTag && tag.Value == aws.FailedRolloutValue {
			rev.Failed = true
		}
	}
	return rev, nil
}

// RevisionNumber returns the revision number in a task definition ARN or
// family:revision, or 0.
func RevisionNumber(arn string) int {
	idx := strings.LastIndex(arn, ":")
	if idx == -1 {
		return 0
	}
	n, err := strconv.Atoi(arn[idx+1:])
	if err != nil {
		return 0
	}
	return n
}

// Image returns the image of the container a deploy changes, picked the
// way Deploy picks it: the one named, the only one, or the one named like
// the service. Falls back to the first container.
func (r Revision) Image(container, service string) string {
	if len(r.Images) == 0 {
		return ""
	}
	want := container
	if want == "" {
		want = service
	}
	for i, name := range r.Containers {
		if name == want && i < len(r.Images) {
			return r.Images[i]
		}
	}
	return r.Images[0]
}
