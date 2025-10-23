package script

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// ExecutionContext manages the context and isolation for script execution
type ExecutionContext struct {
	// Execution metadata
	ID         string
	ModuleName string
	ScriptName string
	StartTime  time.Time
	UserID     string
	RequestID  string

	// Security and resource limits
	SecurityLimits SecurityLimits

	// Context data
	Variables map[string]interface{}
	Functions map[string]interface{}

	// Execution state
	mu       sync.RWMutex
	canceled bool
	result   interface{}
	logs     []string
}

// ContextManager manages script execution contexts and enforces isolation
type ContextManager struct {
	mu             sync.RWMutex
	activeContexts map[string]*ExecutionContext
	maxConcurrent  int
	defaultLimits  SecurityLimits
	contextCounter int64
}

// NewContextManager creates a new context manager
func NewContextManager(maxConcurrent int, defaultLimits SecurityLimits) *ContextManager {
	return &ContextManager{
		activeContexts: make(map[string]*ExecutionContext),
		maxConcurrent:  maxConcurrent,
		defaultLimits:  defaultLimits,
	}
}

// CreateExecutionContext creates a new isolated execution context
func (cm *ContextManager) CreateExecutionContext(req ExecutionRequest, userID, requestID string) (*ExecutionContext, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check concurrent execution limit
	if len(cm.activeContexts) >= cm.maxConcurrent {
		return nil, NewScriptError(
			ErrorTypeExecution,
			req.ModuleName,
			req.ScriptName,
			fmt.Sprintf("maximum concurrent executions reached: %d", cm.maxConcurrent),
			nil,
		)
	}

	// Generate unique context ID
	cm.contextCounter++
	contextID := fmt.Sprintf("%s_%s_%d_%d", req.ModuleName, req.ScriptName, time.Now().Unix(), cm.contextCounter)

	// Determine security limits
	limits := cm.defaultLimits
	if req.SecurityLimits.MaxExecutionTime > 0 {
		limits = req.SecurityLimits
	}

	// Create execution context
	execCtx := &ExecutionContext{
		ID:             contextID,
		ModuleName:     req.ModuleName,
		ScriptName:     req.ScriptName,
		StartTime:      time.Now(),
		UserID:         userID,
		RequestID:      requestID,
		SecurityLimits: limits,
		Variables:      make(map[string]interface{}),
		Functions:      make(map[string]interface{}),
		logs:           make([]string, 0),
	}

	// Populate context from request input
	if req.Input != nil {
		execCtx.populateFromInput(req.Input)
	}

	// Register active context
	cm.activeContexts[contextID] = execCtx

	slog.Debug("Created execution context",
		"context_id", contextID,
		"module", req.ModuleName,
		"script", req.ScriptName,
		"user_id", userID,
		"active_contexts", len(cm.activeContexts),
	)

	return execCtx, nil
}

// ReleaseExecutionContext removes a context from active tracking
func (cm *ContextManager) ReleaseExecutionContext(contextID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if ctx, exists := cm.activeContexts[contextID]; exists {
		delete(cm.activeContexts, contextID)

		executionTime := time.Since(ctx.StartTime)
		slog.Debug("Released execution context",
			"context_id", contextID,
			"module", ctx.ModuleName,
			"script", ctx.ScriptName,
			"execution_time", executionTime,
			"remaining_contexts", len(cm.activeContexts),
		)
	}
}

// GetActiveContexts returns information about currently active contexts
func (cm *ContextManager) GetActiveContexts() map[string]ContextInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	info := make(map[string]ContextInfo)
	for id, ctx := range cm.activeContexts {
		ctx.mu.RLock()
		info[id] = ContextInfo{
			ID:            ctx.ID,
			ModuleName:    ctx.ModuleName,
			ScriptName:    ctx.ScriptName,
			StartTime:     ctx.StartTime,
			UserID:        ctx.UserID,
			RequestID:     ctx.RequestID,
			ExecutionTime: time.Since(ctx.StartTime),
			Canceled:      ctx.canceled,
		}
		ctx.mu.RUnlock()
	}

	return info
}

// CancelContext cancels a specific execution context
func (cm *ContextManager) CancelContext(contextID string) bool {
	cm.mu.RLock()
	ctx, exists := cm.activeContexts[contextID]
	cm.mu.RUnlock()

	if !exists {
		return false
	}

	ctx.mu.Lock()
	ctx.canceled = true
	ctx.mu.Unlock()

	slog.Info("Canceled execution context", "context_id", contextID)
	return true
}

// populateFromInput populates the execution context from script input
func (ec *ExecutionContext) populateFromInput(input *ScriptInput) {
	// Add context variables
	if input.Context != nil {
		for key, value := range input.Context {
			ec.Variables[key] = value
		}
	}

	// Add message data if available
	if input.Message != nil {
		ec.Variables["message"] = map[string]interface{}{
			"topic":    input.Message.Topic,
			"user_id":  input.Message.UserID,
			"payload":  string(input.Message.Payload),
			"metadata": input.Message.Metadata,
		}
	}

	// Add HTTP request data if available
	if input.HTTPRequest != nil {
		ec.Variables["http_request"] = map[string]interface{}{
			"method":  input.HTTPRequest.Method,
			"path":    input.HTTPRequest.Path,
			"headers": input.HTTPRequest.Headers,
			"body":    string(input.HTTPRequest.Body),
			"query":   input.HTTPRequest.Query,
		}
	}

	// Add exposed functions
	if input.Functions != nil {
		for name, fn := range input.Functions {
			ec.Functions[name] = fn
		}
	}

	// Add standard context functions
	ec.addStandardFunctions()
}

// addStandardFunctions adds standard functions available to all scripts
func (ec *ExecutionContext) addStandardFunctions() {
	// Log function that captures logs in the context
	ec.Functions["log"] = func(message string) {
		ec.mu.Lock()
		ec.logs = append(ec.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message))
		ec.mu.Unlock()

		slog.Debug("Script log",
			"context_id", ec.ID,
			"module", ec.ModuleName,
			"script", ec.ScriptName,
			"message", message,
		)
	}

	// Context info function
	ec.Functions["get_context"] = func() map[string]interface{} {
		return map[string]interface{}{
			"module":     ec.ModuleName,
			"script":     ec.ScriptName,
			"user_id":    ec.UserID,
			"request_id": ec.RequestID,
			"start_time": ec.StartTime.Format(time.RFC3339),
		}
	}

	// Time functions
	ec.Functions["now"] = func() string {
		return time.Now().Format(time.RFC3339)
	}

	ec.Functions["timestamp"] = func() int64 {
		return time.Now().Unix()
	}
}

// IsCanceled checks if the context has been canceled
func (ec *ExecutionContext) IsCanceled() bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.canceled
}

// SetResult sets the execution result
func (ec *ExecutionContext) SetResult(result interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.result = result
}

// GetResult gets the execution result
func (ec *ExecutionContext) GetResult() interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.result
}

// GetLogs returns captured log messages
func (ec *ExecutionContext) GetLogs() []string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Return a copy to prevent modification
	logs := make([]string, len(ec.logs))
	copy(logs, ec.logs)
	return logs
}

// GetExecutionMetrics returns execution metrics for the context
func (ec *ExecutionContext) GetExecutionMetrics() ExecutionMetrics {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ExecutionMetrics{
		ExecutionTime: time.Since(ec.StartTime),
		MemoryUsed:    int64(memStats.Alloc),
		Success:       !ec.canceled && ec.result != nil,
		ErrorType:     "", // Will be set by caller if there's an error
	}
}

// ContextInfo provides information about an execution context
type ContextInfo struct {
	ID            string
	ModuleName    string
	ScriptName    string
	StartTime     time.Time
	UserID        string
	RequestID     string
	ExecutionTime time.Duration
	Canceled      bool
}

// SecurityValidator validates script execution against security policies
type SecurityValidator struct {
	allowedPackages  map[string]bool
	blockedFunctions map[string]bool
	maxVariableSize  int64
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(limits SecurityLimits) *SecurityValidator {
	allowedPkgs := make(map[string]bool)
	for _, pkg := range limits.AllowedPackages {
		allowedPkgs[pkg] = true
	}

	return &SecurityValidator{
		allowedPackages:  allowedPkgs,
		blockedFunctions: make(map[string]bool),
		maxVariableSize:  limits.MaxMemoryBytes / 10, // 10% of memory limit for variables
	}
}

// ValidateContext validates an execution context against security policies
func (sv *SecurityValidator) ValidateContext(ctx *ExecutionContext) error {
	// Validate variable sizes
	totalSize := int64(0)
	for key, value := range ctx.Variables {
		size := estimateSize(value)
		if size > sv.maxVariableSize {
			return NewScriptError(
				ErrorTypeSecurityViolation,
				ctx.ModuleName,
				ctx.ScriptName,
				fmt.Sprintf("variable %s exceeds size limit: %d bytes", key, size),
				nil,
			)
		}
		totalSize += size
	}

	if totalSize > ctx.SecurityLimits.MaxMemoryBytes/2 {
		return NewScriptError(
			ErrorTypeSecurityViolation,
			ctx.ModuleName,
			ctx.ScriptName,
			fmt.Sprintf("total variable size exceeds limit: %d bytes", totalSize),
			nil,
		)
	}

	return nil
}

// estimateSize estimates the memory size of a value
func estimateSize(value interface{}) int64 {
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case map[string]interface{}:
		size := int64(0)
		for key, val := range v {
			size += int64(len(key)) + estimateSize(val)
		}
		return size
	case []interface{}:
		size := int64(0)
		for _, val := range v {
			size += estimateSize(val)
		}
		return size
	default:
		return 64 // Estimate for other types
	}
}

// EnhancedExecutionRequest extends ExecutionRequest with context management
type EnhancedExecutionRequest struct {
	ExecutionRequest
	UserID    string
	RequestID string
	Context   context.Context
}

// ContextAwareEngine wraps the script engine with context management
type ContextAwareEngine struct {
	engine         ScriptEngine
	contextManager *ContextManager
	validator      *SecurityValidator
}

// NewContextAwareEngine creates a new context-aware script engine
func NewContextAwareEngine(engine ScriptEngine, maxConcurrent int) *ContextAwareEngine {
	defaultLimits := GetDefaultSecurityLimits()
	return &ContextAwareEngine{
		engine:         engine,
		contextManager: NewContextManager(maxConcurrent, defaultLimits),
		validator:      NewSecurityValidator(defaultLimits),
	}
}

// ExecuteWithContext executes a script with enhanced context management
func (cae *ContextAwareEngine) ExecuteWithContext(ctx context.Context, req EnhancedExecutionRequest) (*ScriptOutput, error) {
	// Create execution context
	execCtx, err := cae.contextManager.CreateExecutionContext(req.ExecutionRequest, req.UserID, req.RequestID)
	if err != nil {
		return nil, err
	}

	// Ensure context is released
	defer cae.contextManager.ReleaseExecutionContext(execCtx.ID)

	// Validate context
	if err := cae.validator.ValidateContext(execCtx); err != nil {
		return nil, err
	}

	// Create enhanced script input
	enhancedInput := &ScriptInput{
		Context:     execCtx.Variables,
		Functions:   execCtx.Functions,
		Message:     req.Input.Message,
		HTTPRequest: req.Input.HTTPRequest,
	}

	// Create enhanced execution request
	enhancedReq := ExecutionRequest{
		ModuleName:     req.ModuleName,
		ScriptName:     req.ScriptName,
		Input:          enhancedInput,
		Timeout:        req.Timeout,
		SecurityLimits: execCtx.SecurityLimits,
	}

	// Execute with timeout context
	execCtxWithTimeout, cancel := context.WithTimeout(ctx, execCtx.SecurityLimits.MaxExecutionTime)
	defer cancel()

	// Monitor for cancellation
	done := make(chan *ScriptOutput, 1)
	errChan := make(chan error, 1)

	go func() {
		output, err := cae.engine.Execute(execCtxWithTimeout, enhancedReq)
		if err != nil {
			errChan <- err
			return
		}

		// Enhance output with context information
		if output != nil {
			output.Logs = append(output.Logs, execCtx.GetLogs()...)
			output.Metrics = execCtx.GetExecutionMetrics()
		}

		done <- output
	}()

	// Wait for completion or cancellation
	select {
	case output := <-done:
		return output, nil
	case err := <-errChan:
		return nil, err
	case <-execCtxWithTimeout.Done():
		cae.contextManager.CancelContext(execCtx.ID)
		return nil, NewScriptError(
			ErrorTypeTimeout,
			req.ModuleName,
			req.ScriptName,
			"script execution timed out",
			execCtxWithTimeout.Err(),
		)
	}
}

// GetActiveContexts returns information about active execution contexts
func (cae *ContextAwareEngine) GetActiveContexts() map[string]ContextInfo {
	return cae.contextManager.GetActiveContexts()
}

// CancelExecution cancels a specific script execution
func (cae *ContextAwareEngine) CancelExecution(contextID string) bool {
	return cae.contextManager.CancelContext(contextID)
}
