package telemetry

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// skipPrompt lists commands that never ask: asking during help, a version
// check or shell completion would be in the way, and the telemetry command
// is how people answer anyway.
var skipPrompt = map[string]bool{
	"help":       true,
	"completion": true,
	"telemetry":  true,
	"__complete": true,
	"version":    true,
}

// PromptContext is what decides whether to ask.
type PromptContext struct {
	Command     string // the subcommand's name, e.g. "deploy"
	Interactive bool   // stdin and stdout are terminals and no CI variable is set
	Quiet       bool   // --quiet or --json
	HelpOrVer   bool   // -h, --help or --version on the command line
}

// ShouldPrompt reports whether to ask: the first interactive run of any
// command, unless an env var already decided or the user already answered.
func ShouldPrompt(c PromptContext) bool {
	if !c.Interactive || c.Quiet || c.HelpOrVer || c.Command == "" || skipPrompt[c.Command] {
		return false
	}
	if _, src := Enabled(); src == SourceDoNotTrack || src == SourceEnv {
		return false
	}
	return !Asked()
}

const promptText = `Help improve Pilum? Share anonymous build telemetry: which recipe and step
ran, whether it succeeded, how long it took. Never service names, project
IDs, paths or error messages. Details: https://pilum.dev/docs/reference/telemetry/

Share telemetry? [y/N] `

// Ask shows the prompt, saves the answer and reports it. Anything but y or
// yes, including just pressing Enter, is no.
func Ask(in io.Reader, out io.Writer) bool {
	_, _ = fmt.Fprint(out, promptText)
	line, _ := bufio.NewReader(in).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	yes := answer == "y" || answer == "yes"

	if yes {
		if _, err := Enable(); err == nil {
			_, _ = fmt.Fprintln(out, "Thanks. Turn it off any time with `pilum telemetry disable`.")
		}
	} else {
		_, _ = Disable()
		_, _ = fmt.Fprintln(out, "Telemetry is off. Turn it on any time with `pilum telemetry enable`.")
	}
	_, _ = fmt.Fprintln(out)
	return yes
}
