package rollback

import (
	"os"
	"strconv"
	"strings"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/ecsdeploy"
	"github.com/sid-technologies/pilum/lib/errors"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// ecsLookback caps how many revisions an ECS rollback describes while
// looking for its target.
const ecsLookback = 20

// ecsWaitGrace lets ecs-wait's own timeout fire (with its clearer error)
// before the worker kills the process.
const ecsWaitGrace = 60

// planECS points the service back at an earlier task definition revision,
// which brings back that revision's whole config, not only its image. By
// default that's the newest revision before the current one whose rollout
// didn't fail. --to takes a revision number, family:revision, an ARN, or
// an image tag.
func planECS(plan *Plan, svc serviceinfo.ServiceInfo, to string, run Runner) error {
	cfg := aws.ParseECSConfig(svc)
	target := aws.TargetOf(svc)

	current, err := ecsdeploy.DescribeService(target, cfg.Cluster, cfg.Service, ecsdeploy.Runner(run))
	if err != nil {
		return err
	}
	family, _, _ := strings.Cut(ecsdeploy.Family(current.TaskDefinition), ":")
	currentNumber := ecsdeploy.RevisionNumber(current.TaskDefinition)

	arns, err := ecsdeploy.Revisions(target, family, ecsdeploy.Runner(run))
	if err != nil {
		return err
	}

	rev, err := selectECSRevision(arns, family, currentNumber, to, func(arn string) (ecsdeploy.Revision, error) {
		return ecsdeploy.DescribeRevision(target, arn, ecsdeploy.Runner(run))
	})
	if err != nil {
		return errors.Wrap(err, "%s", svc.DisplayName())
	}
	if len(rev.Images) == 0 {
		return errors.New("%s: revision %s has no containers", svc.DisplayName(), ecsdeploy.Family(rev.ARN))
	}

	plan.From = ecsdeploy.Family(current.TaskDefinition)
	plan.To = ecsdeploy.Family(rev.ARN)
	plan.ToImage = rev.Image(cfg.Container, svc.Name)
	plan.Command = aws.UpdateECSServiceCommand(target, cfg.Cluster, cfg.Service, rev.ARN, -1)
	plan.Wait = append([]string{pilumExecutable()}, aws.GenerateECSWaitArgs(svc, plan.ToImage)...)
	plan.WaitTimeout = cfg.WaitTimeout + ecsWaitGrace
	return nil
}

// selectECSRevision picks the rollback target from a family's revisions,
// newest first.
func selectECSRevision(arns []string, family string, current int, to string,
	describe func(string) (ecsdeploy.Revision, error),
) (ecsdeploy.Revision, error) {
	if to != "" {
		return explicitECSRevision(arns, family, current, to, describe)
	}

	described := 0
	for _, arn := range arns {
		if ecsdeploy.RevisionNumber(arn) >= current {
			continue
		}
		if described == ecsLookback {
			break
		}
		described++
		rev, err := describe(arn)
		if err != nil {
			return ecsdeploy.Revision{}, err
		}
		if !rev.Failed {
			return rev, nil
		}
	}
	return ecsdeploy.Revision{}, errors.New("no earlier revision of %s than %s:%d to roll back to (failed rollouts are skipped)",
		family, family, current)
}

// explicitECSRevision resolves --to: a revision number, family:revision, a
// task definition ARN, or else an image tag, matched against the newest
// revisions. An explicit target may be one marked failed: asked for by
// name, it's what the person wants.
func explicitECSRevision(arns []string, family string, current int, to string,
	describe func(string) (ecsdeploy.Revision, error),
) (ecsdeploy.Revision, error) {
	number := 0
	n, err := strconv.Atoi(to)
	switch {
	case err == nil:
		number = n
	case strings.HasPrefix(to, family+":") || strings.Contains(to, ":task-definition/"+family+":"):
		number = ecsdeploy.RevisionNumber(to)
	default:
		// An image tag, matched below.
	}

	if number != 0 {
		if number == current {
			return ecsdeploy.Revision{}, errors.New("%s:%d is already deployed", family, number)
		}
		for _, arn := range arns {
			if ecsdeploy.RevisionNumber(arn) == number {
				return describe(arn)
			}
		}
		return ecsdeploy.Revision{}, errors.New("no active revision %s:%d", family, number)
	}

	for i := 0; i < len(arns) && i < ecsLookback; i++ {
		rev, descErr := describe(arns[i])
		if descErr != nil {
			return ecsdeploy.Revision{}, descErr
		}
		for _, image := range rev.Images {
			if !strings.HasSuffix(image, ":"+to) {
				continue
			}
			if rev.Number == current {
				return ecsdeploy.Revision{}, errors.New("%s (%s:%d) is already deployed", to, family, current)
			}
			return rev, nil
		}
	}
	return ecsdeploy.Revision{}, errors.New("no revision among the latest %d of %s runs a %q image", ecsLookback, family, to)
}

// pilumExecutable is the running pilum binary, so the wait runs the same
// version even when it isn't the one on PATH.
func pilumExecutable() string {
	exe, err := os.Executable()
	if err == nil {
		return exe
	}
	return "pilum"
}
