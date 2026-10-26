package telemetry

import (
	"os"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/exitcodes"
	"github.com/sid-technologies/pilum/lib/recipe"
)

// Schema v1, mirrored from Pilum-Metrics-Worker's schema/v1.json (a copy is
// in testdata/, and TestSchemaMatchesWorker keeps them in step). The Worker
// drops anything outside these lists, so the CLI never builds it.
const schemaVersion = 1

var (
	trackedCommands = map[string]bool{"deploy": true, "build": true, "publish": true, "push": true, "rollback": true}
	knownOS         = map[string]bool{"linux": true, "darwin": true, "windows": true, "freebsd": true}
	knownArch       = map[string]bool{"amd64": true, "arm64": true, "386": true, "arm": true}
	versionPattern  = regexp.MustCompile(`^v\d{1,3}\.\d{1,4}\.\d{1,5}(-[0-9A-Za-z.-]{1,20})?$`)
)

const (
	maxServices   = 1000
	maxDurationMS = 86_400_000
)

// builtinSteps is the embedded recipes' step names by recipe, loaded once.
var builtinSteps = sync.OnceValue(func() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	infos, err := recipe.LoadEmbeddedRecipes()
	if err != nil {
		return out
	}
	for _, info := range infos {
		steps := map[string]bool{}
		for _, s := range info.Recipe.Steps {
			steps[s.Name] = true
		}
		out[info.Recipe.Name] = steps
	}
	return out
})

// Step is one finished recipe step. Err is classified, never sent.
type Step struct {
	Recipe   string
	Name     string
	Success  bool
	Duration time.Duration
	Err      error
}

// Run collects one actionable command while it runs.
type Run struct {
	Command       string
	Flags         map[string]bool
	Recipes       []string // the recipe of each service, in any order
	CustomRecipes bool     // recipes came from --recipe-path, so no name is trusted
	Steps         []Step
	Started       time.Time
}

var (
	mu      sync.Mutex
	current *Run
)

// Begin starts collecting for a tracked command. Other commands are ignored.
func Begin(command string, flags map[string]bool) {
	if !trackedCommands[command] {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	current = &Run{Command: command, Flags: flags, Started: time.Now()}
}

// SetServices records each service's recipe name.
func SetServices(recipes []string, custom bool) {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		current.Recipes = recipes
		current.CustomRecipes = custom
	}
}

// AddSteps records finished steps.
func AddSteps(steps ...Step) {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		current.Steps = append(current.Steps, steps...)
	}
}

// take returns the current run and clears it.
func take() *Run {
	mu.Lock()
	defer mu.Unlock()
	r := current
	current = nil
	return r
}

// Event is one event in Pilum's own wire format.
type Event struct {
	Event string         `json:"event"`
	TS    string         `json:"ts"`
	Props map[string]any `json:"props"`
}

// Batch is the body POSTed to the endpoint.
type Batch struct {
	Schema    int     `json:"schema"`
	InstallID string  `json:"install_id"`
	SentAt    string  `json:"sent_at"`
	Events    []Event `json:"events"`
}

var exitClasses = map[int]string{
	exitcodes.Success:     "ok",
	exitcodes.General:     "general",
	exitcodes.Config:      "config",
	exitcodes.NoServices:  "no_services",
	exitcodes.Deploy:      "deploy",
	exitcodes.IO:          "io",
	exitcodes.InvalidArgs: "invalid_args",
	exitcodes.Lock:        "lock",
}

func ciProvider() (bool, string) {
	switch {
	case os.Getenv("GITHUB_ACTIONS") == "true":
		return true, "github_actions"
	case os.Getenv("GITLAB_CI") != "":
		return true, "gitlab"
	case os.Getenv("BUILDKITE") != "":
		return true, "buildkite"
	case os.Getenv("CIRCLECI") != "":
		return true, "circleci"
	case os.Getenv("JENKINS_URL") != "":
		return true, "jenkins"
	case os.Getenv("CI") != "":
		return true, "other"
	}
	return false, "none"
}

// safeRecipe is the name if it's a built-in recipe, otherwise "custom".
func safeRecipe(name string, custom bool) string {
	if _, ok := builtinSteps()[name]; ok && !custom {
		return name
	}
	return "custom"
}

// safeStep is the name if it's a step of that built-in recipe, otherwise "custom".
func safeStep(rec, name string) string {
	if rec != "custom" && builtinSteps()[rec][name] {
		return name
	}
	return "custom"
}

func clamp(v, upper int64) int64 {
	if v < 0 {
		return 0
	}
	if v > upper {
		return upper
	}
	return v
}

// common is the properties every event carries.
func common(version string) map[string]any {
	ci, provider := ciProvider()
	p := map[string]any{"ci": ci, "ci_provider": provider}
	if versionPattern.MatchString(version) {
		p["version"] = version
	} else {
		p["version"] = "dev"
	}
	if knownOS[runtime.GOOS] {
		p["os"] = runtime.GOOS
	}
	if knownArch[runtime.GOARCH] {
		p["arch"] = runtime.GOARCH
	}
	return p
}

// BuildEvents turns a finished run into events. Only the allowlisted values
// above reach the output: service names, error text and the like are never
// copied, whatever the run carries.
func BuildEvents(r *Run, version string, runErr error, now time.Time) []Event {
	if r == nil {
		return nil
	}
	ts := now.UTC().Format(time.RFC3339)

	recipes := map[string]int64{}
	for _, name := range r.Recipes {
		recipes[safeRecipe(name, r.CustomRecipes)]++
	}
	for k, v := range recipes {
		recipes[k] = clamp(v, maxServices)
	}
	flags := map[string]bool{}
	for _, name := range []string{"dry_run", "only_changed", "no_deps", "no_warm", "force"} {
		if v, ok := r.Flags[name]; ok {
			flags[name] = v
		}
	}

	run := common(version)
	run["command"] = r.Command
	run["success"] = runErr == nil
	run["exit_class"] = exitClasses[exitcodes.CodeFrom(runErr)]
	if run["exit_class"] == "" {
		run["exit_class"] = "general"
	}
	run["duration_ms"] = clamp(now.Sub(r.Started).Milliseconds(), maxDurationMS)
	run["services"] = clamp(int64(len(r.Recipes)), maxServices)
	run["recipes"] = recipes
	run["flags"] = flags

	events := []Event{{Event: "pilum_run", TS: ts, Props: run}}
	for _, s := range r.Steps {
		if s.Success {
			continue
		}
		rec := safeRecipe(s.Recipe, r.CustomRecipes)
		p := common(version)
		p["recipe"] = rec
		p["step"] = safeStep(rec, s.Name)
		p["error_class"] = Classify(s.Err)
		p["duration_ms"] = clamp(s.Duration.Milliseconds(), maxDurationMS)
		events = append(events, Event{Event: "pilum_step_failed", TS: ts, Props: p})
	}
	return events
}
