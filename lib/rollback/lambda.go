package rollback

import (
	"strconv"

	"github.com/sid-technologies/pilum/ingredients/aws"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/lambdadeploy"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// planLambda moves the function's alias back to an earlier published
// version. Instant and nothing is rebuilt; it changes what callers of
// <function>:<alias> run, not callers of the bare function name ($LATEST).
// --to takes a version number or the tag it was deployed with.
func planLambda(plan *Plan, svc serviceinfo.ServiceInfo, to string, run Runner) error {
	cfg := aws.ParseLambdaConfig(svc)
	target := aws.TargetOf(svc)

	current, found, err := lambdadeploy.AliasVersion(target, cfg.Function, cfg.Alias, lambdadeploy.Runner(run))
	if err != nil {
		return errors.Wrap(err, "%s", svc.DisplayName())
	}
	if !found {
		return errors.New("%s: function %s has no alias %s to roll back; a Pilum deploy creates it", svc.DisplayName(), cfg.Function, cfg.Alias)
	}
	currentNumber, err := strconv.Atoi(current)
	if err != nil {
		return errors.New("%s: alias %s points at %s, not a published version", svc.DisplayName(), cfg.Alias, current)
	}

	listed, err := run(aws.ListVersionsCommand(target, cfg.Function))
	if err != nil {
		return errors.Wrap(err, "listing versions of %s", cfg.Function)
	}
	versions, err := lambdadeploy.ParseVersions(listed)
	if err != nil {
		return err
	}

	version, err := selectLambdaVersion(versions, currentNumber, to)
	if err != nil {
		return errors.Wrap(err, "%s", svc.DisplayName())
	}

	plan.From = cfg.Alias + " → " + current
	plan.To = cfg.Alias + " → " + strconv.Itoa(version.Number)
	if version.Tag() != "" {
		plan.To += " (" + version.Tag() + ")"
	}
	plan.Command = aws.UpdateAliasCommand(target, cfg.Function, cfg.Alias, strconv.Itoa(version.Number))
	return nil
}

// selectLambdaVersion picks the target from versions (newest first): the
// version numbered or tagged to, or the newest one before current.
func selectLambdaVersion(versions []lambdadeploy.Version, current int, to string) (lambdadeploy.Version, error) {
	if to == "" {
		for _, v := range versions {
			if v.Number < current {
				return v, nil
			}
		}
		return lambdadeploy.Version{}, errors.New("no published version before %d to roll back to", current)
	}

	number, err := strconv.Atoi(to)
	for _, v := range versions {
		if (err == nil && v.Number == number) || (err != nil && v.Tag() == to) {
			if v.Number == current {
				return lambdadeploy.Version{}, errors.New("version %d (%s) is already live", v.Number, to)
			}
			return v, nil
		}
	}
	if err == nil {
		return lambdadeploy.Version{}, errors.New("no published version %d", number)
	}
	return lambdadeploy.Version{}, errors.New("no version was deployed with tag %q", to)
}
