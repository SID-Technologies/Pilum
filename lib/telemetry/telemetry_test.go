package telemetry

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sid-technologies/pilum/lib/exitcodes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolate points the config at a temp dir and clears every variable that
// changes behavior, so a developer's own settings never leak into a test.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := configDir
	configDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { configDir = old })
	for _, v := range []string{"DO_NOT_TRACK", "PILUM_TELEMETRY", "PILUM_TELEMETRY_ENDPOINT", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE", "CIRCLECI", "JENKINS_URL"} {
		t.Setenv(v, "")
	}
	take()
	return dir
}

func TestEnabledPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		doNotTrack string
		env        string
		config     string
		wantOn     bool
		wantSource Source
	}{
		{"default is off", "", "", "", false, SourceDefault},
		{"config on", "", "", stateEnabled, true, SourceConfig},
		{"config off", "", "", stateDisabled, false, SourceConfig},
		{"env on beats config off", "", "1", stateDisabled, true, SourceEnv},
		{"env off beats config on", "", "0", stateEnabled, false, SourceEnv},
		{"env true spelled out", "", "true", "", true, SourceEnv},
		{"unrecognized env falls through to config", "", "maybe", stateEnabled, true, SourceConfig},
		{"DO_NOT_TRACK beats env on", "1", "1", stateEnabled, false, SourceDoNotTrack},
		{"DO_NOT_TRACK=0 does nothing", "0", "", stateEnabled, true, SourceConfig},
		{"DO_NOT_TRACK=false does nothing", "false", "", stateEnabled, true, SourceConfig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			isolate(t)
			t.Setenv("DO_NOT_TRACK", c.doNotTrack)
			t.Setenv("PILUM_TELEMETRY", c.env)
			if c.config != "" {
				require.NoError(t, Save(Settings{Telemetry: c.config}))
			}
			on, src := Enabled()
			assert.Equal(t, c.wantOn, on)
			assert.Equal(t, c.wantSource, src)
		})
	}
}

func TestEnableDisable(t *testing.T) {
	isolate(t)
	s, err := Enable()
	require.NoError(t, err)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, s.InstallID)

	s, err = Disable()
	require.NoError(t, err)
	assert.Empty(t, s.InstallID, "disable deletes the install ID")
	assert.Equal(t, stateDisabled, Load().Telemetry)
}

func TestShouldPrompt(t *testing.T) {
	interactive := PromptContext{Command: "deploy", Interactive: true}
	cases := []struct {
		name  string
		ctx   PromptContext
		setup func(t *testing.T)
		want  bool
	}{
		{"first interactive run of any command", interactive, nil, true},
		{"read-only commands ask too", PromptContext{Command: "list", Interactive: true}, nil, true},
		{"not interactive (CI, pipes, /dev/null)", PromptContext{Command: "deploy"}, nil, false},
		{"--json or --quiet", PromptContext{Command: "deploy", Interactive: true, Quiet: true}, nil, false},
		{"--help or --version", PromptContext{Command: "deploy", Interactive: true, HelpOrVer: true}, nil, false},
		{"help", PromptContext{Command: "help", Interactive: true}, nil, false},
		{"completion", PromptContext{Command: "completion", Interactive: true}, nil, false},
		{"telemetry itself", PromptContext{Command: "telemetry", Interactive: true}, nil, false},
		{"already answered yes", interactive, func(t *testing.T) { _, _ = Enable() }, false},
		{"already answered no", interactive, func(t *testing.T) { _, _ = Disable() }, false},
		{"PILUM_TELEMETRY decided", interactive, func(t *testing.T) { t.Setenv("PILUM_TELEMETRY", "0") }, false},
		{"DO_NOT_TRACK decided", interactive, func(t *testing.T) { t.Setenv("DO_NOT_TRACK", "1") }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			isolate(t)
			if c.setup != nil {
				c.setup(t)
			}
			assert.Equal(t, c.want, ShouldPrompt(c.ctx))
		})
	}
}

func TestAsk(t *testing.T) {
	for input, want := range map[string]bool{"y\n": true, "YES\n": true, "\n": false, "n\n": false, "sure\n": false, "": false} {
		t.Run(strings.TrimSpace(input), func(t *testing.T) {
			isolate(t)
			var out strings.Builder
			assert.Equal(t, want, Ask(strings.NewReader(input), &out))
			assert.Contains(t, out.String(), "[y/N]")
			on, src := Enabled()
			assert.Equal(t, want, on)
			assert.Equal(t, SourceConfig, src, "either answer is saved, so it is never asked again")
		})
	}
}

// TestNoLeaks is the guarantee in the docs: nothing a user named or typed
// reaches the payload, whatever the run carries.
func TestNoLeaks(t *testing.T) {
	isolate(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	r := &Run{
		Command: "deploy",
		Flags:   map[string]bool{"dry_run": false, "only_changed": true, "secret-flag-zz": true},
		Recipes: []string{"gcp-cloud-run", "secret-svc-zz", "npm"},
		Started: time.Now().Add(-90 * time.Second),
		Steps: []Step{
			{Recipe: "gcp-cloud-run", Name: "deploy to cloud run", Duration: 4 * time.Second,
				Err: errors.New("token=abc123 at /home/x/proj: permission denied for project acme-prod-zz")},
			{Recipe: "secret-svc-zz", Name: "migrate secret-svc-zz", Duration: time.Second, Err: errors.New("exit status 1")},
			{Recipe: "gcp-cloud-run", Name: "my own step zz", Duration: time.Second, Err: errors.New("dial tcp: i/o timeout")},
			{Recipe: "npm", Name: "publish package", Success: true},
		},
	}
	runErr := exitcodes.WithCode(exitcodes.Deploy, errors.New("secret-svc-zz failed at /home/x/proj"))
	events := BuildEvents(r, "v0.11.0", runErr, time.Now())
	body, err := json.Marshal(Batch{Schema: 1, InstallID: newInstallID(), Events: events})
	require.NoError(t, err)

	text := string(body)
	for _, leak := range []string{"secret-svc-zz", "secret-flag-zz", "abc123", "/home/x/proj", "acme-prod-zz", "my own step zz", "token="} {
		assert.NotContains(t, text, leak)
	}

	require.Len(t, events, 4, "one run plus three failed steps")
	run := events[0].Props
	assert.Equal(t, "pilum_run", events[0].Event)
	assert.Equal(t, false, run["success"])
	assert.Equal(t, "deploy", run["exit_class"])
	assert.Equal(t, "github_actions", run["ci_provider"])
	assert.Equal(t, map[string]int64{"gcp-cloud-run": 1, "npm": 1, "custom": 1}, run["recipes"])
	assert.Equal(t, map[string]bool{"dry_run": false, "only_changed": true}, run["flags"])

	assert.Equal(t, "deploy to cloud run", events[1].Props["step"])
	assert.Equal(t, "permission", events[1].Props["error_class"])
	assert.Equal(t, "custom", events[2].Props["recipe"])
	assert.Equal(t, "custom", events[2].Props["step"])
	assert.Equal(t, "custom", events[3].Props["step"], "a step that isn't in the built-in recipe is custom")
}

func TestCustomRecipePathHidesNames(t *testing.T) {
	r := &Run{Command: "build", Recipes: []string{"gcp-cloud-run"}, CustomRecipes: true,
		Steps: []Step{{Recipe: "gcp-cloud-run", Name: "build binary", Err: errors.New("x")}}, Started: time.Now()}
	events := BuildEvents(r, "dev", nil, time.Now())
	assert.Equal(t, map[string]int64{"custom": 1}, events[0].Props["recipes"])
	assert.Equal(t, "custom", events[1].Props["recipe"])
	assert.Equal(t, "custom", events[1].Props["step"])
}

func TestVersionNormalised(t *testing.T) {
	for in, want := range map[string]string{"v0.11.0": "v0.11.0", "v1.2.3-rc.1": "v1.2.3-rc.1", "dev": "dev", "built-by-alice": "dev", "": "dev"} {
		assert.Equal(t, want, common(in)["version"], in)
	}
}

func TestOnlyActionableCommandsAreTracked(t *testing.T) {
	isolate(t)
	for _, cmd := range []string{"check", "list", "status", "dry-run", "delete-builds", "logs"} {
		Begin(cmd, nil)
		assert.Nil(t, take(), cmd)
	}
	Begin("deploy", nil)
	assert.NotNil(t, take())
}

// TestSchemaMatchesWorker keeps the CLI in step with Pilum-Metrics-Worker's
// schema/v1.json (copied into testdata). The Worker drops anything outside
// it, so a new recipe or enum value must be added to both.
func TestSchemaMatchesWorker(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "v1.json"))
	require.NoError(t, err)
	var schema struct {
		Schema int                 `json:"schema"`
		Enums  map[string][]string `json:"enums"`
		Flags  []string            `json:"flags"`
		Recipe map[string][]string `json:"recipes"`
		Limits map[string]int64    `json:"limits"`
	}
	require.NoError(t, json.Unmarshal(data, &schema))
	assert.Equal(t, schemaVersion, schema.Schema)

	sorted := func(m map[string]bool) []string {
		var out []string
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	sortedCopy := func(s []string) []string { c := append([]string(nil), s...); sort.Strings(c); return c }

	for cmd := range trackedCommands {
		assert.Contains(t, schema.Enums["command"], cmd)
	}
	for _, class := range exitClasses {
		assert.Contains(t, schema.Enums["exit_class"], class)
	}
	classes := map[string]bool{"unknown": true}
	for _, c := range errorClasses {
		classes[c.class] = true
	}
	assert.Equal(t, sortedCopy(schema.Enums["error_class"]), sorted(classes))
	assert.Equal(t, sortedCopy(schema.Enums["os"]), sorted(knownOS))
	assert.Equal(t, sortedCopy(schema.Enums["arch"]), sorted(knownArch))
	assert.Equal(t, []string{"dry_run", "force", "no_deps", "no_warm", "only_changed"}, sortedCopy(schema.Flags))
	assert.Equal(t, int64(maxServices), schema.Limits["services"])
	assert.Equal(t, int64(maxDurationMS), schema.Limits["duration_ms"])
	assert.Equal(t, int64(maxEvents), schema.Limits["max_events"])

	builtin := builtinSteps()
	require.NotEmpty(t, builtin)
	assert.Len(t, schema.Recipe, len(builtin), "every embedded recipe is in the Worker schema and vice versa")
	for name, steps := range schema.Recipe {
		want := map[string]bool{}
		for _, s := range steps {
			want[s] = true
		}
		assert.Equal(t, want, builtin[name], "steps of recipe %s", name)
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"ERROR: (gcloud.run.deploy) You do not currently have an active account selected. Run gcloud auth login": "auth",
		"googleapi: Error 403: Permission 'run.services.update' denied":                                          "permission",
		"AccessDenied: User is not authorized to perform: lambda:UpdateFunctionCode":                             "permission",
		"RESOURCE_EXHAUSTED: Quota exceeded for quota metric":                                                    "quota",
		"step timed out after 60s":  "timeout",
		"context deadline exceeded": "timeout",
		"ERROR: (gcloud.run.deploy) NOT_FOUND: Requested entity was not found":     "not_found",
		"dial tcp: lookup registry.example: no such host":                          "network",
		"ERROR: failed to solve: process \"/bin/sh -c go build\" did not complete": "build_failed",
		"npm ERR! code E401":     "auth",
		"something odd happened": "unknown",
	}
	for text, want := range cases {
		assert.Equal(t, want, Classify(errors.New(text)), text)
	}
	assert.Equal(t, "unknown", Classify(nil))
}

// server records what reaches the endpoint.
func server(t *testing.T, handler http.HandlerFunc) (*atomic.Int32, *[]Batch) {
	t.Helper()
	var hits atomic.Int32
	var got []Batch
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		var b Batch
		_ = json.NewDecoder(r.Body).Decode(&b)
		got = append(got, b)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PILUM_TELEMETRY_ENDPOINT", srv.URL)
	return &hits, &got
}

func accept(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]bool{"accept": on})
	}
}

func TestFlushSendsWhenOn(t *testing.T) {
	dir := isolate(t)
	hits, got := server(t, accept(true))
	t.Setenv("PILUM_TELEMETRY", "1")

	Begin("deploy", map[string]bool{"dry_run": true})
	SetServices([]string{"homebrew"}, false)
	Flush("v0.11.0", nil)

	require.Equal(t, int32(1), hits.Load())
	b := (*got)[0]
	assert.Equal(t, 1, b.Schema)
	assert.NotEmpty(t, b.InstallID)
	require.Len(t, b.Events, 1)
	assert.Equal(t, "ok", b.Events[0].Props["exit_class"])

	last, err := os.ReadFile(filepath.Join(dir, "last-telemetry.json"))
	require.NoError(t, err)
	assert.Contains(t, string(last), b.InstallID, "`pilum telemetry last` shows exactly what was sent")
}

func TestFlushSendsNothingWhenOff(t *testing.T) {
	isolate(t)
	hits, _ := server(t, accept(true))
	Begin("deploy", nil)
	Flush("v0.11.0", nil)
	assert.Equal(t, int32(0), hits.Load(), "off by default")

	t.Setenv("PILUM_TELEMETRY", "1")
	t.Setenv("DO_NOT_TRACK", "1")
	Begin("deploy", nil)
	Flush("v0.11.0", nil)
	assert.Equal(t, int32(0), hits.Load(), "DO_NOT_TRACK wins")
}

func TestAcceptFalsePausesForAWeek(t *testing.T) {
	isolate(t)
	hits, _ := server(t, accept(false))
	t.Setenv("PILUM_TELEMETRY", "1")

	Begin("deploy", nil)
	Flush("v0.11.0", nil)
	require.Equal(t, int32(1), hits.Load())
	until, err := time.Parse(time.RFC3339, Load().PausedUntil)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), until, time.Minute)

	Begin("deploy", nil)
	Flush("v0.11.0", nil)
	assert.Equal(t, int32(1), hits.Load(), "paused: nothing sent")
}

func TestFlushNeverWaitsLongerThanASecond(t *testing.T) {
	isolate(t)
	server(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusAccepted)
	})
	t.Setenv("PILUM_TELEMETRY", "1")

	Begin("deploy", nil)
	start := time.Now()
	Flush("v0.11.0", errors.New("deploy failed"))
	assert.Less(t, time.Since(start), 1100*time.Millisecond)
}

func TestFlushSurvivesAnUnreachableEndpoint(t *testing.T) {
	isolate(t)
	t.Setenv("PILUM_TELEMETRY", "1")
	t.Setenv("PILUM_TELEMETRY_ENDPOINT", "http://127.0.0.1:1/v1/events")
	Begin("deploy", nil)
	assert.NotPanics(t, func() { Flush("v0.11.0", nil) })
}
