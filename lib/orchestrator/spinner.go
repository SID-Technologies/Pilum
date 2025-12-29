package orchestrator

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/output"
)

// Spinner frames - a nice smooth animation.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// isCI returns true if running in a CI environment.
func isCI() bool {
	// Check common CI environment variables
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_URL", "BUILDKITE"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// SpinnerManager manages multiple spinners for concurrent tasks.
type SpinnerManager struct {
	mu       sync.Mutex
	spinners map[string]*serviceSpinner
	order    []string // preserve insertion order
	stop     chan struct{}
	stopped  bool
	wg       sync.WaitGroup
	ciMode   bool // true when running in CI - disables animation
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
	disableSpinners := isCI() || output.IsVerbose() || output.IsQuiet() || output.IsJSON()
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

// Stop halts the spinner animation.
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

	sm.spinners[serviceName] = &serviceSpinner{
		name:     padded,
		stepName: stepName,
		frame:    0,
	}
	sm.order = append(sm.order, serviceName)

	// In CI mode, print a static "running" indicator
	if sm.ciMode {
		fmt.Printf("  %s%s%s %s %s%s%s\n",
			colorWarning, symbolRunning, colorReset,
			padded,
			colorMuted, stepName, colorReset)
		return
	}

	// Print initial line with spinner
	fmt.Printf("  %s%s%s %s %s%s%s\n",
		colorWarning, spinnerFrames[0], colorReset,
		padded,
		colorMuted, stepName, colorReset)
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

// render updates all spinner displays.
func (sm *SpinnerManager) render() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Move cursor up for each spinner and redraw
	count := len(sm.order)
	if count == 0 {
		return
	}

	// Move up
	fmt.Printf("\033[%dA", count)

	for _, key := range sm.order {
		s := sm.spinners[key]
		if s.done {
			if s.success {
				fmt.Printf("\033[2K  %s%s%s %s %s(%s)%s\n",
					colorSuccess, symbolSuccess, colorReset,
					s.name,
					colorMuted, formatDuration(s.duration), colorReset)
			} else {
				errMsg := ""
				if s.err != nil {
					errMsg = s.err.Error()
				}
				fmt.Printf("\033[2K  %s%s%s %s %sfailed: %s%s\n",
					colorError, symbolFailure, colorReset,
					s.name,
					colorError, errMsg, colorReset)
			}
		} else {
			s.frame = (s.frame + 1) % len(spinnerFrames)
			fmt.Printf("\033[2K  %s%s%s %s %s%s%s\n",
				colorWarning, spinnerFrames[s.frame], colorReset,
				s.name,
				colorMuted, s.stepName, colorReset)
		}
	}
}

// RenderFinal prints the final state of all spinners (for when animation stops).
func (sm *SpinnerManager) RenderFinal() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := len(sm.order)
	if count == 0 {
		return
	}

	// In CI mode, just print completion status (no cursor manipulation)
	if sm.ciMode {
		for _, key := range sm.order {
			s := sm.spinners[key]
			if s.success {
				fmt.Printf("  %s%s%s %s %s(%s)%s\n",
					colorSuccess, symbolSuccess, colorReset,
					s.name,
					colorMuted, formatDuration(s.duration), colorReset)
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

	// Move up and clear (interactive mode)
	fmt.Printf("\033[%dA", count)

	for _, key := range sm.order {
		s := sm.spinners[key]
		if s.success {
			fmt.Printf("\033[2K  %s%s%s %s %s(%s)%s\n",
				colorSuccess, symbolSuccess, colorReset,
				s.name,
				colorMuted, formatDuration(s.duration), colorReset)
		} else if s.done {
			errMsg := ""
			if s.err != nil {
				errMsg = s.err.Error()
			}
			fmt.Printf("\033[2K  %s%s%s %s %sfailed: %s%s\n",
				colorError, symbolFailure, colorReset,
				s.name,
				colorError, errMsg, colorReset)
		} else {
			// Still running when stopped - mark as interrupted
			fmt.Printf("\033[2K  %s%s%s %s %s(interrupted)%s\n",
				colorWarning, symbolRunning, colorReset,
				s.name,
				colorMuted, colorReset)
		}
	}
}

// Clear removes all spinners.
func (sm *SpinnerManager) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.spinners = make(map[string]*serviceSpinner)
	sm.order = nil
}
