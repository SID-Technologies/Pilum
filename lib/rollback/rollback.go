// Package rollback plans rollbacks of deployed services to an earlier version.
//
// The cloud provider is the source of truth for what was deployed, not the
// local history file: history only exists on the machine that ran the deploy
// (so not in CI), and it records tags rather than revisions or images.
package rollback

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/sid-technologies/pilum/ingredients/gcp"
	"github.com/sid-technologies/pilum/lib/errors"
	"github.com/sid-technologies/pilum/lib/migtemplate"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// Kind is how a service is rolled back.
type Kind string

const (
	// KindService moves traffic to an earlier revision. Instant, no rebuild,
	// and restores that revision's full config, not just its image.
	KindService Kind = "service"
	// KindJob points the job at an earlier image. Jobs have no revisions or
	// traffic, so the job keeps its current config.
	KindJob Kind = "job"
	// KindMIG rolls a managed instance group onto an earlier instance
	// template. Templates are immutable, so that version's full config comes
	// back, not just its image.
	KindMIG Kind = "mig"
)

// KindFor returns how svc is rolled back, or false if its target is unsupported.
// A bare "gcp" recipe key resolves to Cloud Run services, matching `pilum status`.
func KindFor(svc serviceinfo.ServiceInfo) (Kind, bool) {
	switch svc.RecipeKey() {
	case "gcp-cloud-run", "gcp-cloud-run-from-image", "gcp":
		return KindService, true
	case "gcp-cloud-run-job":
		return KindJob, true
	case "gcp-mig-container":
		return KindMIG, true
	}
	return "", false
}

// Plan is a rollback for one service instance (one region).
type Plan struct {
	Service string   `json:"service"`
	Region  string   `json:"region,omitempty"`
	Kind    Kind     `json:"kind"`
	From    string   `json:"from"` // serving revision (service), current image (job) or template (mig)
	To      string   `json:"to"`   // target revision (service), image (job) or template (mig)
	ToImage string   `json:"to_image,omitempty"`
	Command []string `json:"command"`

	// Wait, if set, runs after Command and must succeed for the rollback to
	// count. A MIG rollback is a rolling update that takes minutes, so it
	// waits for the group to be stable, with WaitTimeout seconds to do so.
	Wait        []string `json:"wait,omitempty"`
	WaitTimeout int      `json:"-"`
}

// Runner executes a read-only query command and returns its stdout.
type Runner func(cmd []string) (string, error)

// NewPlan works out what svc would roll back to. to is optional: a revision
// name for services, an image tag or full image reference for jobs. Empty
// means the version before the one currently deployed.
func NewPlan(svc serviceinfo.ServiceInfo, to string, run Runner) (Plan, error) {
	kind, ok := KindFor(svc)
	if !ok {
		return Plan{}, errors.New("rollback is not supported for %s (recipe %s)", svc.DisplayName(), svc.RecipeKey())
	}

	plan := Plan{Service: svc.DisplayName(), Region: svc.Region, Kind: kind}
	var err error
	switch kind {
	case KindJob:
		err = planJob(&plan, svc, to, run)
	case KindMIG:
		err = planMIG(&plan, svc, to, run)
	default:
		err = planService(&plan, svc, to, run)
	}
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func planService(plan *Plan, svc serviceinfo.ServiceInfo, to string, run Runner) error {
	described, err := run(gcp.GenerateStatusCommand(svc))
	if err != nil {
		return errors.Wrap(err, "describing %s", svc.DisplayName())
	}
	serving, err := ServingRevision(described)
	if err != nil {
		return errors.Wrap(err, "reading traffic for %s", svc.DisplayName())
	}

	listed, err := run(gcp.GenerateListRevisionsCommand(svc))
	if err != nil {
		return errors.Wrap(err, "listing revisions for %s", svc.DisplayName())
	}
	revisions, err := ParseRevisions(listed)
	if err != nil {
		return errors.Wrap(err, "parsing revisions for %s", svc.DisplayName())
	}

	target, err := SelectRevision(revisions, serving, to)
	if err != nil {
		return errors.Wrap(err, "%s", svc.DisplayName())
	}

	plan.From = serving
	plan.To = target.Name
	plan.ToImage = target.Image
	plan.Command = gcp.GenerateRollbackTrafficCommand(svc, target.Name)
	return nil
}

func planJob(plan *Plan, svc serviceinfo.ServiceInfo, to string, run Runner) error {
	described, err := run(gcp.GenerateJobStatusCommand(svc))
	if err != nil {
		return errors.Wrap(err, "describing job %s", svc.DisplayName())
	}
	current, err := JobImage(described)
	if err != nil {
		return errors.Wrap(err, "reading image for job %s", svc.DisplayName())
	}

	var target string
	if to != "" {
		target = ResolveImage(current, to)
	} else {
		listed, runErr := run(gcp.GenerateListJobExecutionsCommand(svc))
		if runErr != nil {
			return errors.Wrap(runErr, "listing executions for job %s", svc.DisplayName())
		}
		executions, parseErr := ParseExecutions(listed)
		if parseErr != nil {
			return errors.Wrap(parseErr, "parsing executions for job %s", svc.DisplayName())
		}
		target = PreviousImage(executions, current)
		if target == "" {
			return errors.New("%s: no execution ran an image other than the current one; pass --to <tag>", svc.DisplayName())
		}
	}

	if target == current {
		return errors.New("%s is already on %s", svc.DisplayName(), current)
	}

	plan.From = current
	plan.To = target
	plan.ToImage = target
	plan.Command = gcp.GenerateJobSetImageCommand(svc, target)
	return nil
}

func planMIG(plan *Plan, svc serviceinfo.ServiceInfo, to string, run Runner) error {
	cfg := gcp.ParseMIGConfig(svc.Config)

	described, err := run(gcp.GenerateMIGDescribeCommand(svc))
	if err != nil {
		return errors.Wrap(err, "describing MIG %s", cfg.Name)
	}
	current, err := migtemplate.CurrentTemplate(described)
	if err != nil {
		return errors.Wrap(err, "instance group %s", cfg.Name)
	}

	listed, err := run(gcp.GenerateListMIGTemplatesCommand(svc))
	if err != nil {
		return errors.Wrap(err, "listing templates for %s", svc.DisplayName())
	}
	templates, err := ParseTemplates(listed, cfg.TemplateRegion, cfg.ImageMetadataKey)
	if err != nil {
		return errors.Wrap(err, "parsing templates for %s", svc.DisplayName())
	}

	if to != "" {
		if to, err = templateTarget(cfg.TemplateBase, to); err != nil {
			return err
		}
	}
	target, err := SelectTemplate(templates, migtemplate.TemplateName(current), to)
	if err != nil {
		return errors.Wrap(err, "%s", svc.DisplayName())
	}

	plan.From = migtemplate.TemplateName(current)
	plan.To = target.Name
	plan.ToImage = target.Image
	plan.Command = gcp.GenerateMIGRollingUpdateCommand(svc, target.Name)
	plan.Wait = gcp.GenerateMIGWaitStableCommand(svc)
	plan.WaitTimeout = cfg.WaitTimeout + migWaitGrace
	return nil
}

// migWaitGrace lets gcloud's own wait timeout fire (with its clearer error)
// before the worker kills the process.
const migWaitGrace = 60

// templateTarget turns a --to value into a template name: a name in the
// family (the base, or <base>-...) is used as is, anything else is a tag.
func templateTarget(base, to string) (string, error) {
	if to == base || strings.HasPrefix(to, base+"-") {
		return to, nil
	}
	return gcp.MIGTemplateName(base, to)
}

// ParseTemplates parses `gcloud compute instance-templates list --format json`,
// keeping templates in the given scope (region, or "" for global). Image is
// the template's imageKey metadata value, empty if unset.
func ParseTemplates(listJSON, region, imageKey string) ([]Revision, error) {
	var raw []struct {
		Name              string    `json:"name"`
		SelfLink          string    `json:"selfLink"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Properties        struct {
			Metadata struct {
				Items []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"items"`
			} `json:"metadata"`
		} `json:"properties"`
	}
	err := json.Unmarshal([]byte(listJSON), &raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid templates JSON")
	}

	templates := make([]Revision, 0, len(raw))
	for _, t := range raw {
		if migtemplate.TemplateRegion(t.SelfLink) != region {
			continue
		}
		rev := Revision{Name: t.Name, Created: t.CreationTimestamp, Ready: true}
		for _, item := range t.Properties.Metadata.Items {
			if item.Key == imageKey {
				rev.Image = item.Value
			}
		}
		templates = append(templates, rev)
	}
	return templates, nil
}

// SelectTemplate picks the template to roll back to: to if set, otherwise the
// newest one created before current, so repeated rollbacks keep stepping back.
func SelectTemplate(templates []Revision, current, to string) (Revision, error) {
	if to != "" {
		if to == current {
			return Revision{}, errors.New("instance group already runs template %s", to)
		}
		for _, t := range templates {
			if t.Name == to {
				return t, nil
			}
		}
		return Revision{}, errors.New("template %s not found", to)
	}

	var currentCreated time.Time
	found := false
	for _, t := range templates {
		if t.Name == current {
			currentCreated, found = t.Created, true
			break
		}
	}
	if !found {
		return Revision{}, errors.New("current template %s is not in the mig.template_base family; pass --to <template>", current)
	}

	var best Revision
	for _, t := range templates {
		if t.Created.Before(currentCreated) && t.Created.After(best.Created) {
			best = t
		}
	}
	if best.Name == "" {
		return Revision{}, errors.New("no template older than %s to roll back to", current)
	}
	return best, nil
}

// Revision is one deployed version of a Cloud Run service or job execution.
type Revision struct {
	Name    string
	Created time.Time
	Image   string
	Ready   bool
}

// Cloud Run v1 (Knative-shaped) resources, as `gcloud --format json` emits them.
type (
	meta struct {
		Name              string    `json:"name"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
	}
	containers struct {
		Containers []struct {
			Image string `json:"image"`
		} `json:"containers"`
	}
	revisionJSON struct {
		Metadata meta       `json:"metadata"`
		Spec     containers `json:"spec"`
		Status   struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	}
	serviceJSON struct {
		Status struct {
			Traffic []struct {
				RevisionName string `json:"revisionName"`
				Percent      int    `json:"percent"`
			} `json:"traffic"`
		} `json:"status"`
	}
	executionJSON struct {
		Metadata meta `json:"metadata"`
		Spec     struct {
			Template struct {
				Spec containers `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
	jobJSON struct {
		Spec struct {
			Template struct {
				Spec struct {
					Template struct {
						Spec containers `json:"spec"`
					} `json:"template"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
	}
)

func (c containers) firstImage() string {
	if len(c.Containers) == 0 {
		return ""
	}
	return c.Containers[0].Image
}

// ServingRevision returns the revision receiving the most traffic.
func ServingRevision(serviceJSONOutput string) (string, error) {
	var svc serviceJSON
	if err := json.Unmarshal([]byte(serviceJSONOutput), &svc); err != nil {
		return "", errors.Wrap(err, "invalid service JSON")
	}
	best, bestPercent := "", -1
	for _, t := range svc.Status.Traffic {
		if t.RevisionName != "" && t.Percent > bestPercent {
			best, bestPercent = t.RevisionName, t.Percent
		}
	}
	if best == "" {
		return "", errors.New("service reports no revision serving traffic")
	}
	return best, nil
}

// ParseRevisions parses `gcloud run revisions list --format json`.
func ParseRevisions(listJSON string) ([]Revision, error) {
	var raw []revisionJSON
	err := json.Unmarshal([]byte(listJSON), &raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid revisions JSON")
	}
	revisions := make([]Revision, len(raw))
	for i, r := range raw {
		ready := false
		for _, c := range r.Status.Conditions {
			if c.Type == "Ready" {
				ready = c.Status == "True"
			}
		}
		revisions[i] = Revision{
			Name:    r.Metadata.Name,
			Created: r.Metadata.CreationTimestamp,
			Image:   r.Spec.firstImage(),
			Ready:   ready,
		}
	}
	return revisions, nil
}

// SelectRevision picks the rollback target. With to set, that revision must
// exist, be healthy, and not already be serving. Otherwise it is the newest
// healthy revision created before the serving one, so repeated rollbacks keep
// stepping back and a revision that never became Ready is never chosen.
func SelectRevision(revisions []Revision, serving, to string) (Revision, error) {
	if to != "" {
		if to == serving {
			return Revision{}, errors.New("revision %s is already serving", to)
		}
		for _, r := range revisions {
			if r.Name != to {
				continue
			}
			if !r.Ready {
				return Revision{}, errors.New("revision %s never became ready", to)
			}
			return r, nil
		}
		return Revision{}, errors.New("revision %s not found", to)
	}

	var servingCreated time.Time
	found := false
	for _, r := range revisions {
		if r.Name == serving {
			servingCreated, found = r.Created, true
			break
		}
	}
	if !found {
		return Revision{}, errors.New("serving revision %s not found in revision list", serving)
	}

	var best Revision
	for _, r := range revisions {
		if r.Ready && r.Created.Before(servingCreated) && r.Created.After(best.Created) {
			best = r
		}
	}
	if best.Name == "" {
		return Revision{}, errors.New("no earlier healthy revision than %s to roll back to", serving)
	}
	return best, nil
}

// JobImage returns the image a Cloud Run Job is currently configured with.
func JobImage(jobJSONOutput string) (string, error) {
	var job jobJSON
	if err := json.Unmarshal([]byte(jobJSONOutput), &job); err != nil {
		return "", errors.Wrap(err, "invalid job JSON")
	}
	image := job.Spec.Template.Spec.Template.Spec.firstImage()
	if image == "" {
		return "", errors.New("job has no container image")
	}
	return image, nil
}

// ParseExecutions parses `gcloud run jobs executions list --format json`.
func ParseExecutions(listJSON string) ([]Revision, error) {
	var raw []executionJSON
	err := json.Unmarshal([]byte(listJSON), &raw)
	if err != nil {
		return nil, errors.Wrap(err, "invalid executions JSON")
	}
	executions := make([]Revision, len(raw))
	for i, e := range raw {
		executions[i] = Revision{
			Name:    e.Metadata.Name,
			Created: e.Metadata.CreationTimestamp,
			Image:   e.Spec.Template.Spec.firstImage(),
		}
	}
	return executions, nil
}

// PreviousImage returns the image of the most recent execution that ran
// something other than current, or "" if there is none.
func PreviousImage(executions []Revision, current string) string {
	sorted := append([]Revision(nil), executions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Created.After(sorted[j].Created) })
	for _, e := range sorted {
		if e.Image != "" && e.Image != current {
			return e.Image
		}
	}
	return ""
}

// ResolveImage turns a --to value into a full image reference. A value with a
// "/" is already a full reference; anything else is a tag applied to the
// current image's repository.
func ResolveImage(current, to string) string {
	if strings.Contains(to, "/") {
		return to
	}
	repo := current
	if i := strings.Index(repo, "@"); i >= 0 {
		repo = repo[:i]
	}
	if i := strings.LastIndex(repo, ":"); i > strings.LastIndex(repo, "/") {
		repo = repo[:i]
	}
	return repo + ":" + to
}
