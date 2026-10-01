package output

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Spinner frames - a nice smooth animation.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// liveSpinner is the spinner block currently animating, if any. Output
// functions print above it instead of into it (see emit).
var liveSpinner atomic.Pointer[SpinnerManager]

// SpinnerManager manages multiple spinners for concurrent tasks.
//
// The live block is redrawn by moving the cursor up over it, so the manager
// must know exactly how many terminal rows it occupies. Two rules keep that
// count true: every spinner line is exactly one row (errors are flattened and
// lines are cut to the terminal width, since a wrapped or multi-line row makes
// every later frame drift down and repeat), and anything else printed while
// the block is live goes above it via printAbove.
type SpinnerManager struct {
	mu       sync.Mutex
	spinners map[string]*serviceSpinner
	order    []string // preserve insertion order
	stop     chan struct{}
	stopped  bool
	wg       sync.WaitGroup
	ciMode   bool // true when running in CI - disables animation
	drawn    int  // terminal rows the live block currently occupies
}

type serviceSpinner struct {
	name     string
	stepName string
	frame    int
	done     bool
	success  bool
	err      error
	duration time.Duration
}

// NewSpinnerManager creates a new spinner manager.
func NewSpinnerManager() *SpinnerManager {
	// Disable spinners in CI, verbose, quiet, or JSON mode
	disableSpinners := isCI() || IsVerbose() || IsQuiet() || IsJSON()
	return &SpinnerManager{
		spinners: make(map[string]*serviceSpinner),
		stop:     make(chan struct{}),
		ciMode:   disableSpinners,
	}
}

// Start begins the spinner animation loop.
func (sm *SpinnerManager) Start() {
	// In CI mode, don't start the animation loop
	if sm.ciMode {
		return
	}

	liveSpinner.Store(sm)

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-sm.stop:
				return
			case <-ticker.C:
				sm.render()
			}
		}
	}()
}

// Stop halts the spinner animation. The block stays live (output still goes
// above it) until RenderFinal draws the final state.
func (sm *SpinnerManager) Stop() {
	sm.mu.Lock()
	if sm.stopped {
		sm.mu.Unlock()
		return
	}
	sm.stopped = true
	sm.mu.Unlock()

	close(sm.stop)
	sm.wg.Wait()
}

// AddSpinner adds a new spinner for a service.
func (sm *SpinnerManager) AddSpinner(serviceName, stepName string, maxNameLen int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	padded := serviceName
	if len(serviceName) < maxNameLen {
		padded = serviceName + fmt.Sprintf("%*s", maxNameLen-len(serviceName), "")
	}

	s := &serviceSpinner{name: padded, stepName: stepName}
	sm.spinners[serviceName] = s
	sm.order = append(sm.order, serviceName)

	// In CI mode, print a static "running" indicator
	if sm.ciMode {
		fmt.Printf("  %s%s%s %s %s%s%s\n",
			colorWarning, symbolRunning, colorReset,
			padded,
			colorMuted, stepName, colorReset)
		return
	}

	fmt.Print(sm.line(s, false, terminalColumns()))
	sm.drawn++
}

// Complete marks a spinner as complete.
func (sm *SpinnerManager) Complete(serviceName string, success bool, duration time.Duration, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.spinners[serviceName]; ok {
		s.done = true
		s.success = success
		s.duration = duration
		s.err = err
	}
}

// render redraws the live block with the next animation frame.
func (sm *SpinnerManager) render() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.order) == 0 {
		return
	}
	for _, s := range sm.spinners {
		if !s.done {
			s.frame = (s.frame + 1) % len(spinnerFrames)
		}
	}
	fmt.Print(sm.clearSeq() + sm.frame(false))
	sm.drawn = len(sm.order)
}

// printAbove writes s above the live block: clear the block, print s, redraw.
func (sm *SpinnerManager) printAbove(w io.Writer, s string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	fmt.Print(sm.clearSeq())
	_, _ = fmt.Fprint(w, s)
	fmt.Print(sm.frame(false))
	sm.drawn = len(sm.order)
}

// RenderFinal prints the final state of all spinners (for when animation stops).
func (sm *SpinnerManager) RenderFinal() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.order) == 0 {
		liveSpinner.CompareAndSwap(sm, nil)
		return
	}

	// In CI mode, just print completion status (no cursor manipulation).
	// Nothing is redrawn afterwards, so the full multi-line error is fine here.
	if sm.ciMode {
		for _, key := range sm.order {
			s := sm.spinners[key]
			if s.success {
				fmt.Printf("  %s%s%s %s %s(%s)%s\n",
					colorSuccess, symbolSuccess, colorReset,
					s.name,
					colorMuted, FormatDuration(s.duration), colorReset)
			} else if s.done {
				errMsg := ""
				if s.err != nil {
					errMsg = s.err.Error()
				}
				fmt.Printf("  %s%s%s %s %sfailed: %s%s\n",
					colorError, symbolFailure, colorReset,
					s.name,
					colorError, errMsg, colorReset)
			} else {
				fmt.Printf("  %s%s%s %s %s(interrupted)%s\n",
					colorWarning, symbolRunning, colorReset,
					s.name,
					colorMuted, colorReset)
			}
		}
		return
	}

	fmt.Print(sm.clearSeq() + sm.frame(true))
	sm.drawn = 0 // final lines stay on screen; nothing will move over them
	liveSpinner.CompareAndSwap(sm, nil)
}

// Clear removes all spinners.
func (sm *SpinnerManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.spinners = make(map[string]*serviceSpinner)
	sm.order = nil
	sm.drawn = 0
	liveSpinner.CompareAndSwap(sm, nil)
}

// clearSeq moves the cursor to the top of the live block and clears from
// there to the end of the screen. Callers hold sm.mu.
func (sm *SpinnerManager) clearSeq() string {
	if sm.drawn == 0 {
		return ""
	}
	return fmt.Sprintf("\033[%dA\r\033[J", sm.drawn)
}

// frame renders every spinner as one row each. final marks unfinished
// spinners as interrupted. Callers hold sm.mu.
func (sm *SpinnerManager) frame(final bool) string {
	width := terminalColumns()
	var b strings.Builder
	for _, key := range sm.order {
		b.WriteString(sm.line(sm.spinners[key], final, width))
	}
	return b.String()
}

// line renders one spinner as exactly one terminal row.
func (*SpinnerManager) line(s *serviceSpinner, final bool, width int) string {
	var symbolColor, symbol, textColor, text string
	switch {
	case s.done && s.success:
		symbolColor, symbol = colorSuccess, symbolSuccess
		textColor, text = colorMuted, "("+FormatDuration(s.duration)+")"
	case s.done:
		errMsg := ""
		if s.err != nil {
			errMsg = s.err.Error()
		}
		symbolColor, symbol = colorError, symbolFailure
		textColor, text = colorError, "failed: "+errMsg
	case final:
		symbolColor, symbol = colorWarning, symbolRunning
		textColor, text = colorMuted, "(interrupted)"
	default:
		symbolColor, symbol = colorWarning, spinnerFrames[s.frame]
		textColor, text = colorMuted, s.stepName
	}

	// Visible layout: "  " + symbol + " " + name + " " + text. Keep one column
	// spare so the cursor never lands on the wrap boundary.
	budget := width - 1 - (2 + 1 + 1 + len([]rune(s.name)) + 1)
	text = fitOneRow(text, budget)

	return fmt.Sprintf("\033[2K  %s%s%s %s %s%s%s\n",
		symbolColor, symbol, colorReset, s.name, textColor, text, colorReset)
}

// fitOneRow flattens s to a single line and cuts it to at most budget runes.
func fitOneRow(s string, budget int) string {
	s = strings.Join(strings.Fields(s), " ")
	if budget <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= budget {
		return s
	}
	if budget == 1 {
		return "…"
	}
	return string(r[:budget-1]) + "…"
}

// terminalColumns is stdout's width, falling back to $COLUMNS, then 80.
func terminalColumns() int {
	if w := terminalWidth(); w > 0 {
		return w
	}
	if w, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && w > 0 {
		return w
	}
	return 80
}

// emit writes s to w, or above the live spinner block when one is animating,
// so messages printed mid-step (errors, warnings, debug) don't land inside
// the block and throw off its redraw.
func emit(w io.Writer, s string) {
	if sm := liveSpinner.Load(); sm != nil {
		sm.printAbove(w, s)
		return
	}
	_, _ = fmt.Fprint(w, s)
}
