package script

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorReporter_ReportError(t *testing.T) {
	reporter := NewErrorReporter()

	// Create a test error
	scriptErr := NewScriptError(
		ErrorTypeExecution,
		"test_module",
		"test_script",
		"test error message",
		nil,
	)

	// Create execution context
	execCtx := &ExecutionContext{
		ID:         "test-exec-123",
		ModuleName: "test_module",
		ScriptName: "test_script",
		UserID:     "user123",
		RequestID:  "req456",
		StartTime:  time.Now(),
	}

	// Report the error
	report := reporter.ReportError(context.Background(), scriptErr, execCtx)

	// Verify report contents
	assert.Equal(t, scriptErr, report.Error)
	assert.Equal(t, SeverityMedium, report.Severity)
	assert.True(t, report.Recoverable)
	assert.True(t, report.FirstOccurrence)
	assert.Equal(t, 0, report.RetryCount)
	assert.NotEmpty(t, report.SuggestedAction)
	assert.False(t, report.CircuitBreakerTriggered)

	// Verify context
	assert.Equal(t, "test_module", report.Context.ModuleName)
	assert.Equal(t, "test_script", report.Context.ScriptName)
	assert.Equal(t, "test-exec-123", report.Context.ExecutionID)
	assert.Equal(t, "user123", report.Context.UserID)
	assert.Equal(t, "req456", report.Context.RequestID)
	assert.NotEmpty(t, report.Context.StackTrace)
	assert.NotEmpty(t, report.Context.SystemInfo.GoVersion)
}

func TestErrorReporter_DetermineSeverity(t *testing.T) {
	reporter := NewErrorReporter()

	testCases := []struct {
		errorType        ErrorType
		expectedSeverity ErrorSeverity
	}{
		{ErrorTypeSecurityViolation, SeverityCritical},
		{ErrorTypeMemoryLimit, SeverityHigh},
		{ErrorTypeTimeout, SeverityMedium},
		{ErrorTypeCompilation, SeverityMedium},
		{ErrorTypeInvalidSyntax, SeverityMedium},
		{ErrorTypeExecution, SeverityMedium},
		{ErrorTypeNotFound, SeverityLow},
	}

	for _, tc := range testCases {
		t.Run(string(tc.errorType), func(t *testing.T) {
			err := NewScriptError(tc.errorType, "test", "test", "test", nil)
			severity := reporter.determineSeverity(err)
			assert.Equal(t, tc.expectedSeverity, severity)
		})
	}
}

func TestErrorReporter_IsRecoverable(t *testing.T) {
	reporter := NewErrorReporter()

	testCases := []struct {
		errorType   ErrorType
		recoverable bool
	}{
		{ErrorTypeSecurityViolation, false},
		{ErrorTypeInvalidSyntax, false},
		{ErrorTypeNotFound, true},    // Can fallback
		{ErrorTypeCompilation, true}, // Can fallback
		{ErrorTypeExecution, true},   // Can retry
		{ErrorTypeTimeout, true},     // Can retry
		{ErrorTypeMemoryLimit, true}, // Can retry
	}

	for _, tc := range testCases {
		t.Run(string(tc.errorType), func(t *testing.T) {
			err := NewScriptError(tc.errorType, "test", "test", "test", nil)
			recoverable := reporter.isRecoverable(err)
			assert.Equal(t, tc.recoverable, recoverable)
		})
	}
}

func TestErrorReporter_CircuitBreaker(t *testing.T) {
	reporter := NewErrorReporter()

	// Set low threshold for testing
	policy := reporter.recoveryPolicy
	policy.CircuitBreakerThreshold = 3
	reporter.SetRecoveryPolicy(policy)

	scriptErr := NewScriptError(
		ErrorTypeExecution,
		"test_module",
		"test_script",
		"test error",
		nil,
	)

	// Report errors up to threshold
	for i := 0; i < 3; i++ {
		report := reporter.ReportError(context.Background(), scriptErr, nil)
		if i < 2 {
			assert.False(t, report.CircuitBreakerTriggered)
		} else {
			assert.True(t, report.CircuitBreakerTriggered)
		}
		assert.Equal(t, i, report.RetryCount)
	}
}

func TestErrorReporter_ErrorSummary(t *testing.T) {
	reporter := NewErrorReporter()

	// Report various errors
	errors := []*ScriptError{
		NewScriptError(ErrorTypeExecution, "module1", "script1", "error1", nil),
		NewScriptError(ErrorTypeExecution, "module1", "script1", "error2", nil),
		NewScriptError(ErrorTypeCompilation, "module1", "script2", "error3", nil),
		NewScriptError(ErrorTypeTimeout, "module2", "script3", "error4", nil),
	}

	for _, err := range errors {
		reporter.ReportError(context.Background(), err, nil)
	}

	// Get summary
	summary := reporter.GetErrorSummary()

	// Verify summary
	assert.Equal(t, 4, summary.TotalErrors)
	assert.Equal(t, 2, summary.ErrorsByType[ErrorTypeExecution])
	assert.Equal(t, 1, summary.ErrorsByType[ErrorTypeCompilation])
	assert.Equal(t, 1, summary.ErrorsByType[ErrorTypeTimeout])
	assert.Equal(t, 3, summary.ErrorsByModule["module1"])
	assert.Equal(t, 1, summary.ErrorsByModule["module2"])
	assert.NotNil(t, summary.MostCommonError)
	assert.False(t, summary.LastErrorTime.IsZero())
}

func TestErrorReporter_SuggestAction(t *testing.T) {
	reporter := NewErrorReporter()

	testCases := []struct {
		errorType ErrorType
		contains  string
	}{
		{ErrorTypeCompilation, "syntax"},
		{ErrorTypeExecution, "logic"},
		{ErrorTypeTimeout, "performance"},
		{ErrorTypeMemoryLimit, "memory"},
		{ErrorTypeSecurityViolation, "security"},
		{ErrorTypeNotFound, "exists"},
		{ErrorTypeInvalidSyntax, "syntax"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.errorType), func(t *testing.T) {
			err := NewScriptError(tc.errorType, "test", "test", "test", nil)
			suggestion := reporter.suggestAction(err)
			assert.Contains(t, suggestion, tc.contains)
		})
	}
}

func TestErrorReporter_ClearHistory(t *testing.T) {
	reporter := NewErrorReporter()

	// Report an error
	scriptErr := NewScriptError(ErrorTypeExecution, "test", "test", "test", nil)
	reporter.ReportError(context.Background(), scriptErr, nil)

	// Verify error was recorded
	summary := reporter.GetErrorSummary()
	assert.Equal(t, 1, summary.TotalErrors)

	// Clear history
	reporter.ClearErrorHistory()

	// Verify history was cleared
	summary = reporter.GetErrorSummary()
	assert.Equal(t, 0, summary.TotalErrors)
}

func TestEngine_ErrorReporting(t *testing.T) {
	// Create engine with mock config
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	// Try to execute non-existent script
	req := ExecutionRequest{
		ModuleName: "nonexistent",
		ScriptName: "nonexistent",
	}

	_, err := engine.Execute(context.Background(), req)
	require.Error(t, err)

	// Check that error was reported
	summary := engine.GetErrorSummary()
	assert.Equal(t, 1, summary.TotalErrors)
	assert.Equal(t, 1, summary.ErrorsByType[ErrorTypeNotFound])
}

func TestRecoveryPolicy_Configuration(t *testing.T) {
	reporter := NewErrorReporter()

	// Test default policy
	assert.True(t, reporter.recoveryPolicy.FallbackEnabled)
	assert.Equal(t, 5, reporter.recoveryPolicy.CircuitBreakerThreshold)

	// Update policy
	newPolicy := RecoveryPolicy{
		MaxRetries: map[ErrorType]int{
			ErrorTypeExecution: 5,
		},
		FallbackEnabled:         false,
		CircuitBreakerThreshold: 10,
		CooldownPeriod:          10 * time.Minute,
	}

	reporter.SetRecoveryPolicy(newPolicy)

	// Verify policy was updated
	assert.False(t, reporter.recoveryPolicy.FallbackEnabled)
	assert.Equal(t, 10, reporter.recoveryPolicy.CircuitBreakerThreshold)
	assert.Equal(t, 10*time.Minute, reporter.recoveryPolicy.CooldownPeriod)
	assert.Equal(t, 5, reporter.recoveryPolicy.MaxRetries[ErrorTypeExecution])
}
