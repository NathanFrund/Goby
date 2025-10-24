package script

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"
)

// ErrorReporter handles comprehensive error reporting and recovery
type ErrorReporter struct {
	errorCounts    map[string]int
	lastErrors     map[string]*ScriptError
	recoveryPolicy RecoveryPolicy
}

// RecoveryPolicy defines how the system should handle different types of errors
type RecoveryPolicy struct {
	// MaxRetries defines maximum retry attempts for different error types
	MaxRetries map[ErrorType]int

	// FallbackEnabled determines if fallback to embedded scripts is allowed
	FallbackEnabled bool

	// CircuitBreakerThreshold defines how many consecutive errors trigger circuit breaker
	CircuitBreakerThreshold int

	// CooldownPeriod defines how long to wait before retrying after circuit breaker
	CooldownPeriod time.Duration
}

// ErrorContext provides additional context for error analysis
type ErrorContext struct {
	ModuleName    string
	ScriptName    string
	ExecutionID   string
	UserID        string
	RequestID     string
	Timestamp     time.Time
	StackTrace    string
	SystemInfo    SystemInfo
	ScriptContent string // First 500 chars for debugging
}

// SystemInfo captures system state at time of error
type SystemInfo struct {
	GoVersion     string
	OS            string
	Arch          string
	NumGoroutines int
	MemoryUsage   int64
	CPUUsage      float64
}

// ErrorSummary provides aggregated error information
type ErrorSummary struct {
	TotalErrors     int
	ErrorsByType    map[ErrorType]int
	ErrorsByModule  map[string]int
	MostCommonError *ScriptError
	ErrorRate       float64 // errors per minute
	LastErrorTime   time.Time
}

// NewErrorReporter creates a new error reporter with default recovery policy
func NewErrorReporter() *ErrorReporter {
	return &ErrorReporter{
		errorCounts: make(map[string]int),
		lastErrors:  make(map[string]*ScriptError),
		recoveryPolicy: RecoveryPolicy{
			MaxRetries: map[ErrorType]int{
				ErrorTypeCompilation:       1, // Don't retry compilation errors
				ErrorTypeExecution:         2, // Retry execution errors twice
				ErrorTypeTimeout:           1, // Don't retry timeouts
				ErrorTypeMemoryLimit:       1, // Don't retry memory limit errors
				ErrorTypeSecurityViolation: 0, // Never retry security violations
				ErrorTypeNotFound:          1, // Retry not found once
				ErrorTypeInvalidSyntax:     0, // Never retry syntax errors
			},
			FallbackEnabled:         true,
			CircuitBreakerThreshold: 5,
			CooldownPeriod:          5 * time.Minute,
		},
	}
}

// ReportError reports and categorizes a script error with full context
func (er *ErrorReporter) ReportError(ctx context.Context, err *ScriptError, execCtx *ExecutionContext) *ErrorReport {
	// Create error context
	errorCtx := er.createErrorContext(err, execCtx)

	// Generate error key for tracking
	errorKey := fmt.Sprintf("%s/%s/%s", err.ModuleName, err.ScriptName, err.Type)

	// Update error counts
	er.errorCounts[errorKey]++
	er.lastErrors[errorKey] = err

	// Create comprehensive error report
	report := &ErrorReport{
		Error:           err,
		Context:         errorCtx,
		Severity:        er.determineSeverity(err),
		Recoverable:     er.isRecoverable(err),
		SuggestedAction: er.suggestAction(err),
		RetryCount:      er.errorCounts[errorKey] - 1,
		FirstOccurrence: er.errorCounts[errorKey] == 1,
	}

	// Log the error with appropriate level
	er.logError(report)

	// Check if circuit breaker should be triggered
	if er.shouldTriggerCircuitBreaker(errorKey) {
		report.CircuitBreakerTriggered = true
		slog.Error("Circuit breaker triggered for script",
			"module", err.ModuleName,
			"script", err.ScriptName,
			"error_count", er.errorCounts[errorKey],
		)
	}

	return report
}

// ErrorReport contains comprehensive information about a script error
type ErrorReport struct {
	Error                   *ScriptError
	Context                 *ErrorContext
	Severity                ErrorSeverity
	Recoverable             bool
	SuggestedAction         string
	RetryCount              int
	FirstOccurrence         bool
	CircuitBreakerTriggered bool
}

// ErrorSeverity categorizes the impact of errors
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical" // System-threatening errors
	SeverityHigh     ErrorSeverity = "high"     // Feature-breaking errors
	SeverityMedium   ErrorSeverity = "medium"   // Degraded functionality
	SeverityLow      ErrorSeverity = "low"      // Minor issues
)

// createErrorContext builds comprehensive error context
func (er *ErrorReporter) createErrorContext(err *ScriptError, execCtx *ExecutionContext) *ErrorContext {
	// Capture stack trace
	stackBuf := make([]byte, 4096)
	stackSize := runtime.Stack(stackBuf, false)
	stackTrace := string(stackBuf[:stackSize])

	// Get system info
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	sysInfo := SystemInfo{
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		NumGoroutines: runtime.NumGoroutine(),
		MemoryUsage:   int64(memStats.Alloc),
	}

	// Build error context
	errorCtx := &ErrorContext{
		ModuleName: err.ModuleName,
		ScriptName: err.ScriptName,
		Timestamp:  err.Timestamp,
		StackTrace: stackTrace,
		SystemInfo: sysInfo,
	}

	// Add execution context if available
	if execCtx != nil {
		errorCtx.ExecutionID = execCtx.ID
		errorCtx.UserID = execCtx.UserID
		errorCtx.RequestID = execCtx.RequestID

		// Add truncated script content for debugging
		if script, exists := execCtx.Variables["script_content"]; exists {
			if content, ok := script.(string); ok {
				if len(content) > 500 {
					errorCtx.ScriptContent = content[:500] + "..."
				} else {
					errorCtx.ScriptContent = content
				}
			}
		}
	}

	return errorCtx
}

// determineSeverity categorizes error severity based on type and context
func (er *ErrorReporter) determineSeverity(err *ScriptError) ErrorSeverity {
	switch err.Type {
	case ErrorTypeSecurityViolation:
		return SeverityCritical
	case ErrorTypeMemoryLimit:
		return SeverityHigh
	case ErrorTypeTimeout:
		return SeverityMedium
	case ErrorTypeCompilation, ErrorTypeInvalidSyntax:
		return SeverityMedium
	case ErrorTypeExecution:
		// Check if this is a recurring execution error
		errorKey := fmt.Sprintf("%s/%s/%s", err.ModuleName, err.ScriptName, err.Type)
		if er.errorCounts[errorKey] > 3 {
			return SeverityHigh
		}
		return SeverityMedium
	case ErrorTypeNotFound:
		return SeverityLow
	default:
		return SeverityMedium
	}
}

// isRecoverable determines if an error can be recovered from
func (er *ErrorReporter) isRecoverable(err *ScriptError) bool {
	switch err.Type {
	case ErrorTypeSecurityViolation, ErrorTypeInvalidSyntax:
		return false // These require manual intervention
	case ErrorTypeNotFound:
		return er.recoveryPolicy.FallbackEnabled // Can fallback to embedded
	case ErrorTypeCompilation:
		return er.recoveryPolicy.FallbackEnabled // Can fallback to embedded
	case ErrorTypeExecution, ErrorTypeTimeout, ErrorTypeMemoryLimit:
		return true // Can retry with different parameters
	default:
		return false
	}
}

// suggestAction provides actionable suggestions for error resolution
func (er *ErrorReporter) suggestAction(err *ScriptError) string {
	switch err.Type {
	case ErrorTypeCompilation:
		return "Check script syntax and fix compilation errors. Consider reverting to embedded script."
	case ErrorTypeExecution:
		return "Review script logic and input data. Check for runtime errors in script code."
	case ErrorTypeTimeout:
		return "Optimize script performance or increase timeout limits. Check for infinite loops."
	case ErrorTypeMemoryLimit:
		return "Reduce memory usage in script or increase memory limits. Check for memory leaks."
	case ErrorTypeSecurityViolation:
		return "Review script for security violations. Remove unauthorized operations."
	case ErrorTypeNotFound:
		return "Ensure script file exists or fallback to embedded script is available."
	case ErrorTypeInvalidSyntax:
		return "Fix syntax errors in script file. Validate script against language specification."
	default:
		return "Review error details and script implementation."
	}
}

// shouldTriggerCircuitBreaker determines if circuit breaker should be activated
func (er *ErrorReporter) shouldTriggerCircuitBreaker(errorKey string) bool {
	return er.errorCounts[errorKey] >= er.recoveryPolicy.CircuitBreakerThreshold
}

// logError logs the error with appropriate level and context
func (er *ErrorReporter) logError(report *ErrorReport) {
	baseFields := []interface{}{
		"module", report.Error.ModuleName,
		"script", report.Error.ScriptName,
		"error_type", report.Error.Type,
		"severity", report.Severity,
		"recoverable", report.Recoverable,
		"retry_count", report.RetryCount,
		"first_occurrence", report.FirstOccurrence,
	}

	// Add execution context if available
	if report.Context != nil {
		baseFields = append(baseFields,
			"execution_id", report.Context.ExecutionID,
			"user_id", report.Context.UserID,
			"request_id", report.Context.RequestID,
			"memory_usage", report.Context.SystemInfo.MemoryUsage,
			"goroutines", report.Context.SystemInfo.NumGoroutines,
		)
	}

	// Add error message and cause
	baseFields = append(baseFields, "error_message", report.Error.Message)
	if report.Error.Cause != nil {
		baseFields = append(baseFields, "underlying_error", report.Error.Cause.Error())
	}

	// Log with appropriate level based on severity
	switch report.Severity {
	case SeverityCritical:
		slog.Error("Critical script error", baseFields...)
	case SeverityHigh:
		slog.Error("High severity script error", baseFields...)
	case SeverityMedium:
		slog.Warn("Medium severity script error", baseFields...)
	case SeverityLow:
		slog.Info("Low severity script error", baseFields...)
	}

	// Log suggested action
	if report.SuggestedAction != "" {
		slog.Info("Error resolution suggestion",
			"module", report.Error.ModuleName,
			"script", report.Error.ScriptName,
			"suggestion", report.SuggestedAction,
		)
	}

	// Log stack trace for critical errors
	if report.Severity == SeverityCritical && report.Context != nil {
		slog.Debug("Stack trace for critical error",
			"module", report.Error.ModuleName,
			"script", report.Error.ScriptName,
			"stack_trace", report.Context.StackTrace,
		)
	}
}

// GetErrorSummary returns aggregated error statistics
func (er *ErrorReporter) GetErrorSummary() *ErrorSummary {
	summary := &ErrorSummary{
		ErrorsByType:   make(map[ErrorType]int),
		ErrorsByModule: make(map[string]int),
	}

	var mostCommonCount int
	var lastErrorTime time.Time

	for errorKey, count := range er.errorCounts {
		summary.TotalErrors += count

		// Parse error key to extract type and module
		parts := strings.Split(errorKey, "/")
		if len(parts) >= 3 {
			module := parts[0]
			errorType := ErrorType(parts[2])

			summary.ErrorsByType[errorType] += count
			summary.ErrorsByModule[module] += count
		}

		// Track most common error
		if count > mostCommonCount {
			mostCommonCount = count
			if lastErr, exists := er.lastErrors[errorKey]; exists {
				summary.MostCommonError = lastErr
			}
		}

		// Track latest error time
		if lastErr, exists := er.lastErrors[errorKey]; exists {
			if lastErr.Timestamp.After(lastErrorTime) {
				lastErrorTime = lastErr.Timestamp
			}
		}
	}

	summary.LastErrorTime = lastErrorTime

	// Calculate error rate (errors per minute over last hour)
	if !lastErrorTime.IsZero() {
		timeSinceLastError := time.Since(lastErrorTime)
		if timeSinceLastError < time.Hour {
			summary.ErrorRate = float64(summary.TotalErrors) / timeSinceLastError.Minutes()
		}
	}

	return summary
}

// ClearErrorHistory clears error tracking history
func (er *ErrorReporter) ClearErrorHistory() {
	er.errorCounts = make(map[string]int)
	er.lastErrors = make(map[string]*ScriptError)
	slog.Info("Error history cleared")
}

// SetRecoveryPolicy updates the error recovery policy
func (er *ErrorReporter) SetRecoveryPolicy(policy RecoveryPolicy) {
	er.recoveryPolicy = policy
	slog.Info("Error recovery policy updated",
		"fallback_enabled", policy.FallbackEnabled,
		"circuit_breaker_threshold", policy.CircuitBreakerThreshold,
		"cooldown_period", policy.CooldownPeriod,
	)
}
