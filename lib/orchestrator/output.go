package orchestrator

import (
	"encoding/json"
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
	if output.IsQuiet() || output.IsJSON() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBold, message, colorReset)
	fmt.Println()
}

// PrintStepHeader prints a step header with separator.
func (o *OutputManager) PrintStepHeader(stepNum, totalSteps int, stepName string) {
	if output.IsQuiet() || output.IsJSON() {
		return
	}
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
	if output.IsQuiet() || output.IsJSON() {
		return
	}
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
	if output.IsQuiet() || output.IsJSON() {
		return
	}
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
	if output.IsQuiet() || output.IsJSON() {
		return
	}
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
	if output.IsQuiet() || output.IsJSON() {
		return
	}
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s(%s)%s\n",
		colorMuted, symbolSkipped, colorReset,
		padded,
		colorMuted, reason, colorReset)
}

// PrintDryRun prints a dry-run preview for a service.
func (o *OutputManager) PrintDryRun(serviceName, stepName string, command any) {
	if output.IsQuiet() || output.IsJSON() {
		return
	}
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
	if output.IsQuiet() || output.IsJSON() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	fmt.Printf("  %s%s%s\n", colorMuted, message, colorReset)
}

// JSONResult represents the JSON output format.
type JSONResult struct {
	Success      bool           `json:"success"`
	TotalTime    string         `json:"total_time"`
	SuccessCount int            `json:"success_count"`
	FailedCount  int            `json:"failed_count"`
	Results      []JSONTaskInfo `json:"results"`
}

// JSONTaskInfo represents a single task result in JSON format.
type JSONTaskInfo struct {
	Service  string `json:"service"`
	Step     string `json:"step"`
	Success  bool   `json:"success"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
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

	// JSON mode: output structured JSON
	if output.IsJSON() {
		jsonResults := make([]JSONTaskInfo, len(results))
		for i, r := range results {
			errStr := ""
			if r.Error != nil {
				errStr = r.Error.Error()
			}
			jsonResults[i] = JSONTaskInfo{
				Service:  r.ServiceName,
				Step:     r.StepName,
				Success:  r.Success,
				Duration: formatDuration(r.Duration),
				Error:    errStr,
			}
		}
		jsonOutput := JSONResult{
			Success:      failedCount == 0,
			TotalTime:    formatDuration(totalDuration),
			SuccessCount: successCount,
			FailedCount:  failedCount,
			Results:      jsonResults,
		}
		data, _ := json.MarshalIndent(jsonOutput, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Quiet mode: just print a summary line
	if output.IsQuiet() {
		if failedCount == 0 {
			fmt.Printf("OK: %d/%d services completed in %s\n",
				successCount, successCount+failedCount, formatDuration(totalDuration))
		} else {
			fmt.Printf("FAILED: %d/%d services failed (%s)\n",
				failedCount, successCount+failedCount, strings.Join(failedServices, ", "))
		}
		return
	}

	// Normal/verbose mode: print full summary
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
