package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sid-technologies/pilum/lib/types"
)

// Color aliases using semantic colors.
const (
	colorReset   = Reset
	colorError   = ErrorColor
	colorSuccess = SuccessColor
	colorWarning = WarningColor
	colorPrimary = Primary
	colorMuted   = Muted
	colorBold    = Bold
	colorInfo    = InfoColor
)

// Symbol aliases.
const (
	symbolRunning = SymbolInfo
	symbolSuccess = SymbolSuccess
	symbolFailure = SymbolFailure
	symbolSkipped = SymbolSkipped
	symbolDryRun  = SymbolDryRun
)

// PipelineOutput handles formatted CLI output for the orchestrator.
type PipelineOutput struct {
	mu           sync.Mutex
	MaxNameLen   int
	useColors    bool
	serviceState map[string]string // tracks current state of each service
}

// NewPipelineOutput creates a new pipeline output manager.
func NewPipelineOutput() *PipelineOutput {
	return &PipelineOutput{
		useColors:    true,
		serviceState: make(map[string]string),
	}
}

// SetMaxNameLength sets the maximum service name length for alignment.
func (o *PipelineOutput) SetMaxNameLength(length int) {
	o.MaxNameLen = length
}

// PrintHeader prints the main deployment header.
func (o *PipelineOutput) PrintHeader(message string) {
	if IsQuiet() || IsJSON() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBold, message, colorReset)
	fmt.Println()
}

// PrintStepHeader prints a step header with separator.
func (o *PipelineOutput) PrintStepHeader(stepNum, totalSteps int, stepName string) {
	if IsQuiet() || IsJSON() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	line := strings.Repeat("━", 50)
	fmt.Printf("\n%s%s Step %d/%d: %s %s%s\n", colorPrimary, line[:3], stepNum, totalSteps, stepName, line[:40], colorReset)
}

// PrintRunning prints a running status for a service.
func (o *PipelineOutput) PrintRunning(serviceName, stepName string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "running"
	if IsQuiet() || IsJSON() {
		return
	}
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s%s%s\n",
		colorWarning, symbolRunning, colorReset,
		padded,
		colorMuted, stepName, colorReset)
}

// PrintSuccess prints a success status for a service.
func (o *PipelineOutput) PrintSuccess(serviceName string, duration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "success"
	if IsQuiet() || IsJSON() {
		return
	}
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s(%s)%s\n",
		colorSuccess, symbolSuccess, colorReset,
		padded,
		colorMuted, FormatDuration(duration), colorReset)
}

// PrintFailure prints a failure status for a service.
func (o *PipelineOutput) PrintFailure(serviceName string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "failed"
	if IsQuiet() || IsJSON() {
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
func (o *PipelineOutput) PrintSkipped(serviceName, reason string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "skipped"
	if IsQuiet() || IsJSON() {
		return
	}
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s(%s)%s\n",
		colorMuted, symbolSkipped, colorReset,
		padded,
		colorMuted, reason, colorReset)
}

// PrintDryRun prints a dry-run preview for a service.
func (o *PipelineOutput) PrintDryRun(serviceName, stepName string, command any) {
	if IsQuiet() || IsJSON() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	padded := o.padName(serviceName)
	cmdStr := FormatCommand(command)
	fmt.Printf("  %s%s%s %s %s%s%s\n",
		colorInfo, symbolDryRun, colorReset,
		padded,
		colorMuted, stepName, colorReset)
	if cmdStr != "" {
		fmt.Printf("      %s→ %s%s\n", colorMuted, cmdStr, colorReset)
	}
}

// PrintWaveHeader prints a wave sub-header within a step.
func (o *PipelineOutput) PrintWaveHeader(waveNum, totalWaves int) {
	if IsQuiet() || IsJSON() {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	fmt.Printf("  %sWave %d/%d:%s\n", colorMuted, waveNum, totalWaves, colorReset)
}

// PrintSkippedDependency prints a message when a service is skipped due to a failed dependency.
func (o *PipelineOutput) PrintSkippedDependency(serviceName, failedDep string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.serviceState[serviceName] = "skipped"
	if IsQuiet() || IsJSON() {
		return
	}
	padded := o.padName(serviceName)
	fmt.Printf("  %s%s%s %s %s(dependency %s failed)%s\n",
		colorWarning, symbolSkipped, colorReset,
		padded,
		colorMuted, failedDep, colorReset)
}

// PrintInfo prints an info message.
func (o *PipelineOutput) PrintInfo(message string) {
	if IsQuiet() || IsJSON() {
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
func (o *PipelineOutput) PrintComplete(results []types.TaskResult) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Track unique services and their final status (failed if any step failed)
	serviceStatus := make(map[string]bool) // true = all steps succeeded
	var totalDuration time.Duration

	for _, r := range results {
		totalDuration += r.Duration
		if !r.Success {
			serviceStatus[r.ServiceName] = false
		} else if _, exists := serviceStatus[r.ServiceName]; !exists {
			serviceStatus[r.ServiceName] = true
		}
	}

	successCount := 0
	failedCount := 0
	var failedServices []string
	for name, ok := range serviceStatus {
		if ok {
			successCount++
		} else {
			failedCount++
			failedServices = append(failedServices, name)
		}
	}

	// JSON mode: output structured JSON
	if IsJSON() {
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
				Duration: FormatDuration(r.Duration),
				Error:    errStr,
			}
		}
		jsonOutput := JSONResult{
			Success:      failedCount == 0,
			TotalTime:    FormatDuration(totalDuration),
			SuccessCount: successCount,
			FailedCount:  failedCount,
			Results:      jsonResults,
		}
		data, _ := json.MarshalIndent(jsonOutput, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Quiet mode: just print a summary line
	if IsQuiet() {
		if failedCount == 0 {
			fmt.Printf("OK: %d/%d services completed in %s\n",
				successCount, successCount+failedCount, FormatDuration(totalDuration))
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

	fmt.Printf("     Total time: %s\n", FormatDuration(totalDuration))
	fmt.Println()
}

// padName pads a service name for alignment.
func (o *PipelineOutput) padName(name string) string {
	if o.MaxNameLen == 0 {
		o.MaxNameLen = 20
	}
	if len(name) >= o.MaxNameLen {
		return name
	}
	return name + strings.Repeat(" ", o.MaxNameLen-len(name))
}

// FormatDuration formats a duration nicely.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// FormatCommand formats a command for display.
func FormatCommand(cmd any) string {
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
