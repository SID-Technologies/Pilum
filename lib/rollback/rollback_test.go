package rollback

import (
	"fmt"
	"strings"
	"testing"
	"time"

	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"

	"github.com/stretchr/testify/require"
)

// Trimmed from `gcloud run services describe --format json`.
const serviceFixture = `{
  "metadata": {"name": "api"},
  "status": {
    "traffic": [
      {"revisionName": "api-00003-ccc", "percent": 100, "latestRevision": true}
    ]
  }
}`

// Trimmed from `gcloud run revisions list --format json` (newest first, as gcloud emits).
const revisionsFixture = `[
  {"metadata": {"name": "api-00003-ccc", "creationTimestamp": "2026-09-03T10:00:00.000000Z"},
   "spec": {"containers": [{"image": "us-docker.pkg.dev/p/r/api:v3"}]},
   "status": {"conditions": [{"type": "Ready", "status": "True"}]}},
  {"metadata": {"name": "api-00002-bbb", "creationTimestamp": "2026-09-02T10:00:00.000000Z"},
   "spec": {"containers": [{"image": "us-docker.pkg.dev/p/r/api:v2"}]},
   "status": {"conditions": [{"type": "Ready", "status": "False"}]}},
  {"metadata": {"name": "api-00001-aaa", "creationTimestamp": "2026-09-01T10:00:00.000000Z"},
   "spec": {"containers": [{"image": "us-docker.pkg.dev/p/r/api:v1"}]},
   "status": {"conditions": [{"type": "Ready", "status": "True"}]}}
]`

// Trimmed from `gcloud run jobs describe --format json`.
const jobFixture = `{
  "metadata": {"name": "nightly"},
  "spec": {"template": {"spec": {"template": {"spec": {
    "containers": [{"image": "us-docker.pkg.dev/p/r/nightly:v3"}]
  }}}}}
}`

// Trimmed from `gcloud run jobs executions list --format json`.
const executionsFixture = `[
  {"metadata": {"name": "nightly-c", "creationTimestamp": "2026-09-03T02:00:00Z"},
   "spec": {"template": {"spec": {"containers": [{"image": "us-docker.pkg.dev/p/r/nightly:v3"}]}}}},
  {"metadata": {"name": "nightly-a", "creationTimestamp": "2026-09-01T02:00:00Z"},
   "spec": {"template": {"spec": {"containers": [{"image": "us-docker.pkg.dev/p/r/nightly:v1"}]}}}},
  {"metadata": {"name": "nightly-b", "creationTimestamp": "2026-09-02T02:00:00Z"},
   "spec": {"template": {"spec": {"containers": [{"image": "us-docker.pkg.dev/p/r/nightly:v2"}]}}}}
]`

// fakeRunner answers gcloud queries by subcommand.
func fakeRunner(responses map[string]string) Runner {
	return func(cmd []string) (string, error) {
		joined := strings.Join(cmd, " ")
		for key, resp := range responses {
			if strings.Contains(joined, key) {
				return resp, nil
			}
		}
		return "", fmt.Errorf("unexpected command: %s", joined)
	}
}

func TestKindFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ  string
		want Kind
		ok   bool
	}{
		{"gcp-cloud-run", KindService, true},
		{"gcp-cloud-run-from-image", KindService, true},
		{"gcp-cloud-run-job", KindJob, true},
		{"aws-lambda", "", false},
		{"azure-container-apps", "", false},
	}
	for _, tt := range tests {
		kind, ok := KindFor(serviceinfo.ServiceInfo{Name: "x", Provider: "p", Type: tt.typ})
		require.Equal(t, tt.ok, ok, tt.typ)
		require.Equal(t, tt.want, kind, tt.typ)
	}

	// Bare provider falls back to Cloud Run services, like `pilum status`.
	kind, ok := KindFor(serviceinfo.ServiceInfo{Name: "x", Provider: "gcp"})
	require.True(t, ok)
	require.Equal(t, KindService, kind)
}

func TestServingRevision(t *testing.T) {
	t.Parallel()

	rev, err := ServingRevision(serviceFixture)
	require.NoError(t, err)
	require.Equal(t, "api-00003-ccc", rev)

	// Split traffic: the revision with the largest share is "serving".
	rev, err = ServingRevision(`{"status": {"traffic": [
		{"revisionName": "a", "percent": 10},
		{"revisionName": "b", "percent": 90}]}}`)
	require.NoError(t, err)
	require.Equal(t, "b", rev)

	_, err = ServingRevision(`{"status": {}}`)
	require.Error(t, err)
}

func TestSelectRevision(t *testing.T) {
	t.Parallel()

	revisions, err := ParseRevisions(revisionsFixture)
	require.NoError(t, err)
	require.Len(t, revisions, 3)
	require.Equal(t, "us-docker.pkg.dev/p/r/api:v3", revisions[0].Image)
	require.Equal(t, time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC), revisions[0].Created)

	t.Run("previous skips revisions that never became ready", func(t *testing.T) {
		t.Parallel()
		got, err := SelectRevision(revisions, "api-00003-ccc", "")
		require.NoError(t, err)
		require.Equal(t, "api-00001-aaa", got.Name)
	})

	t.Run("nothing older", func(t *testing.T) {
		t.Parallel()
		_, err := SelectRevision(revisions, "api-00001-aaa", "")
		require.ErrorContains(t, err, "no earlier healthy revision")
	})

	t.Run("serving revision missing from list", func(t *testing.T) {
		t.Parallel()
		_, err := SelectRevision(revisions, "api-00009-zzz", "")
		require.Error(t, err)
	})

	t.Run("explicit target", func(t *testing.T) {
		t.Parallel()
		got, err := SelectRevision(revisions, "api-00003-ccc", "api-00001-aaa")
		require.NoError(t, err)
		require.Equal(t, "api-00001-aaa", got.Name)
	})

	t.Run("explicit target errors", func(t *testing.T) {
		t.Parallel()
		_, err := SelectRevision(revisions, "api-00003-ccc", "api-00003-ccc")
		require.ErrorContains(t, err, "already serving")
		_, err = SelectRevision(revisions, "api-00003-ccc", "api-00002-bbb")
		require.ErrorContains(t, err, "never became ready")
		_, err = SelectRevision(revisions, "api-00003-ccc", "nope")
		require.ErrorContains(t, err, "not found")
	})
}

func TestPreviousImage(t *testing.T) {
	t.Parallel()

	executions, err := ParseExecutions(executionsFixture)
	require.NoError(t, err)

	// Out-of-order input; newest non-current is v2, not v1.
	require.Equal(t, "us-docker.pkg.dev/p/r/nightly:v2", PreviousImage(executions, "us-docker.pkg.dev/p/r/nightly:v3"))
	require.Empty(t, PreviousImage(executions[:1], "us-docker.pkg.dev/p/r/nightly:v3"))
}

func TestResolveImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		current, to, want string
	}{
		{"us-docker.pkg.dev/p/r/app:v3", "v1", "us-docker.pkg.dev/p/r/app:v1"},
		{"us-docker.pkg.dev/p/r/app@sha256:abc", "v1", "us-docker.pkg.dev/p/r/app:v1"},
		{"localhost:5000/app:v3", "v1", "localhost:5000/app:v1"},
		{"localhost:5000/app", "v1", "localhost:5000/app:v1"},
		{"us-docker.pkg.dev/p/r/app:v3", "other.io/x/app:v9", "other.io/x/app:v9"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, ResolveImage(tt.current, tt.to), tt.current+" + "+tt.to)
	}
}

func TestNewPlanService(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{Name: "api", Provider: "gcp", Type: "gcp-cloud-run", Region: "us-central1", Project: "proj"}
	run := fakeRunner(map[string]string{
		"services describe": serviceFixture,
		"revisions list":    revisionsFixture,
	})

	plan, err := NewPlan(svc, "", run)
	require.NoError(t, err)
	require.Equal(t, KindService, plan.Kind)
	require.Equal(t, "api-00003-ccc", plan.From)
	require.Equal(t, "api-00001-aaa", plan.To)
	require.Equal(t, "us-docker.pkg.dev/p/r/api:v1", plan.ToImage)
	require.Equal(t, []string{
		"gcloud", "run", "services", "update-traffic", "api",
		"--to-revisions=api-00001-aaa=100",
		"--region", "us-central1", "--project", "proj",
	}, plan.Command)
}

func TestNewPlanJob(t *testing.T) {
	t.Parallel()

	svc := serviceinfo.ServiceInfo{Name: "nightly", Provider: "gcp", Type: "gcp-cloud-run-job", Region: "us-central1"}
	run := fakeRunner(map[string]string{
		"jobs describe":        jobFixture,
		"jobs executions list": executionsFixture,
	})

	plan, err := NewPlan(svc, "", run)
	require.NoError(t, err)
	require.Equal(t, "us-docker.pkg.dev/p/r/nightly:v3", plan.From)
	require.Equal(t, "us-docker.pkg.dev/p/r/nightly:v2", plan.To)
	require.Equal(t, []string{
		"gcloud", "run", "jobs", "update", "nightly",
		"--image", "us-docker.pkg.dev/p/r/nightly:v2",
		"--region", "us-central1",
	}, plan.Command)

	plan, err = NewPlan(svc, "v1", run)
	require.NoError(t, err)
	require.Equal(t, "us-docker.pkg.dev/p/r/nightly:v1", plan.To)

	_, err = NewPlan(svc, "v3", run)
	require.ErrorContains(t, err, "already on")
}

func TestNewPlanUnsupported(t *testing.T) {
	t.Parallel()

	_, err := NewPlan(serviceinfo.ServiceInfo{Name: "fn", Provider: "aws", Type: "aws-lambda"}, "", fakeRunner(nil))
	require.ErrorContains(t, err, "not supported")
}
