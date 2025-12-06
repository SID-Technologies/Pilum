package orchestrator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/output"
)

// Color aliases using semantic colors.
const (
	colorReset   = output.Reset
	colorError   = output.ErrorColor
	colorSuccess = output.SuccessColor
	colorWarning = output.WarningColor
	colorPrimary = output.Primary
	colorMuted   = output.Muted
	colorBold    = output.Bold
	colorInfo    = output.InfoColor
)

// Symbol aliases.
const (
	symbolRunning = output.SymbolInfo
	symbolSuccess = output.SymbolSuccess
	symbolFailure = output.SymbolFailure
	symbolSkipped = output.SymbolSkipped
	symbolDryRun  = output.SymbolDryRun
)

// OutputManager handles formatted CLI output for the orchestrator.
type OutputManager struct {
	mu           sync.Mutex
	maxNameLen   int
	useColors    bool
	serviceState map[string]string // tracks current state of each service
}

// NewOutputManager creates a new output manager.
func NewOutputManager() *OutputManager {
	return &OutputManager{
		useColors:    true,
		serviceState: make(map[string]string),
	}
}

// SetMaxNameLength sets the maximum service name length for alignment.
func (o *OutputManager) SetMaxNameLength(length int) {
	o.maxNameLen = length
}

// PrintHeader prints the main deployment header.
func (o *OutputManager) PrintHeader(message string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBold, message, colorReset)
	fmt.Println()
}

// PrintStepHeader prints a step header with separator.
func (o *OutputManager) PrintStepHeader(stepNum, totalSteps int, stepName string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	line := strings.Repeat("━", 50)
	fmt.Printf("\n%s%s Step %d/%d: %s %s%s\n", colorPrimary, line[:3], stepNum, totalSteps, stepName, line[:40], colorReset)
}

// PrintRunning prints a running status for a service.
func (o *OutputManager) PrintRunning(serviceName, stepName string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "running"
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s%s%s\n",
		colorWarning, symbolRunning, colorReset,
		padded,
		colorMuted, stepName, colorReset)
}

// PrintSuccess prints a success status for a service.
func (o *OutputManager) PrintSuccess(serviceName string, duration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "success"
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s(%s)%s\n",
		colorSuccess, symbolSuccess, colorReset,
		padded,
		colorMuted, formatDuration(duration), colorReset)
}

// PrintFailure prints a failure status for a service.
func (o *OutputManager) PrintFailure(serviceName string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "failed"
	padded := o.padName(serviceName)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	fmt.Printf("  %s%s%s %s %sfailed: %s%s\n",
		colorError, symbolFailure, colorReset,
		padded,
		colorError, errMsg, colorReset)
}

// PrintSkipped prints a skipped status for a service.
func (o *OutputManager) PrintSkipped(serviceName, reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "skipped"
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s(%s)%s\n",
		colorMuted, symbolSkipped, colorReset,
		padded,
		colorMuted, reason, colorReset)
}

// PrintDryRun prints a dry-run preview for a service.
func (o *OutputManager) PrintDryRun(serviceName, stepName string, command any) {
	o.mu.Lock()
	defer o.mu.Unlock()

	padded := o.padName(serviceName)
	cmdStr := formatCommand(command)
	fmt.Printf("  %s%s%s %s %s%s%s\n",
		colorInfo, symbolDryRun, colorReset,
		padded,
		colorMuted, stepName, colorReset)
	if cmdStr != "" {
		fmt.Printf("      %s→ %s%s\n", colorMuted, cmdStr, colorReset)
	}
}

// PrintInfo prints an info message.
func (o *OutputManager) PrintInfo(message string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	fmt.Printf("  %s%s%s\n", colorMuted, message, colorReset)
}

// PrintComplete prints the completion summary.
func (o *OutputManager) PrintComplete(results []TaskResult) {
	o.mu.Lock()
	defer o.mu.Unlock()

	successCount := 0
	failedCount := 0
	var failedServices []string
	var totalDuration time.Duration

	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failedCount++
			failedServices = append(failedServices, r.ServiceName)
		}
		totalDuration += r.Duration
	}

	fmt.Println()
	line := strings.Repeat("━", 50)
	fmt.Printf("%s%s Complete %s%s\n", colorPrimary, line[:3], line[:40], colorReset)

	if failedCount == 0 {
		fmt.Printf("  %s%s%s %d/%d services completed successfully\n",
			colorSuccess, symbolSuccess, colorReset,
			successCount, successCount+failedCount)
	} else {
		fmt.Printf("  %s%s%s %d/%d services completed, %d failed\n",
			colorError, symbolFailure, colorReset,
			successCount, successCount+failedCount, failedCount)
		fmt.Printf("     Failed: %s\n", strings.Join(failedServices, ", "))
	}

	fmt.Printf("     Total time: %s\n", formatDuration(totalDuration))
	fmt.Println()
}

// padName pads a service name for alignment.
func (o *OutputManager) padName(name string) string {
	if o.maxNameLen == 0 {
		o.maxNameLen = 20
	}
	if len(name) >= o.maxNameLen {
		return name
	}
	return name + strings.Repeat(" ", o.maxNameLen-len(name))
}

// formatDuration formats a duration nicely.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// formatCommand formats a command for display.
func formatCommand(cmd any) string {
	if cmd == nil {
		return ""
	}
	switch v := cmd.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, " ")
	case []any:
		parts := make([]string, len(v))
		for i, p := range v {
			parts[i] = fmt.Sprintf("%v", p)
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", cmd)
	}
}
