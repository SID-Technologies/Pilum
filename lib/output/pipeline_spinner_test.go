package output

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

func TestFitOneRow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		in     string
		budget int
		want   string
	}{
		{"fits", "short", 10, "short"},
		{"flattens newlines", "command failed for x:\nERROR: daemon\n", 40, "command failed for x: ERROR: daemon"},
		{"truncates", "abcdefghij", 5, "abcd…"},
		{"counts runes not bytes", "ééééé", 5, "ééééé"},
		{"budget one", "abc", 1, "…"},
		{"no room", "abc", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, fitOneRow(tt.in, tt.budget))
		})
	}
}

// Every spinner line must occupy exactly one terminal row, or the cursor-up
// redraw drifts and repeats lines every frame.
func TestSpinnerFrameIsOneRowPerSpinner(t *testing.T) {
	t.Parallel()

	const width = 60
	sm := &SpinnerManager{spinners: map[string]*serviceSpinner{}}
	add := func(key string, s *serviceSpinner) {
		sm.spinners[key] = s
		sm.order = append(sm.order, key)
	}
	add("bad", &serviceSpinner{name: "bad    ", done: true, err: errors.New(
		"command failed for bad:\nERROR: Cannot connect to the Docker daemon at unix:///var/run/docker.sock.\nIs the docker daemon running?")})
	add("good", &serviceSpinner{name: "good   ", done: true, success: true, duration: time.Second})
	add("slow", &serviceSpinner{name: "slow   ", stepName: strings.Repeat("very long step name ", 10)})

	for _, final := range []bool{false, true} {
		var rows []string
		for _, key := range sm.order {
			rows = append(rows, sm.line(sm.spinners[key], final, width))
		}
		frame := strings.Join(rows, "")
		require.Equal(t, len(sm.order), strings.Count(frame, "\n"), "one newline per spinner")
		for _, row := range rows {
			visible := []rune(strings.TrimSuffix(ansi.ReplaceAllString(row, ""), "\n"))
			require.Less(t, len(visible), width, "row must not wrap: %q", string(visible))
		}
	}
}

// Messages printed while the block is live must go above it, and the block
// must be redrawn with the cursor-up count matching the rows it occupies.
//
//nolint:paralleltest // swaps os.Stdout
func TestEmitPrintsAboveLiveSpinner(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	sm := &SpinnerManager{spinners: map[string]*serviceSpinner{}, stop: make(chan struct{})}
	sm.AddSpinner("a", "build", 4)
	sm.AddSpinner("b", "build", 4)
	liveSpinner.Store(sm)
	t.Cleanup(func() { liveSpinner.Store(nil) })

	emit(os.Stdout, "hello\n")
	sm.RenderFinal()
	require.Nil(t, liveSpinner.Load(), "RenderFinal releases the live block")

	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	s := string(out)

	// Two initial rows, then: up 2, clear, message, redraw; then final redraw.
	require.Contains(t, s, "\x1b[2A\r\x1b[Jhello\n")
	require.Equal(t, 2, strings.Count(s, "\x1b[2A"), "each redraw moves up exactly the drawn rows")
	require.Less(t, strings.Index(s, "hello"), strings.LastIndex(s, " a "), "message lands above the block")
}
