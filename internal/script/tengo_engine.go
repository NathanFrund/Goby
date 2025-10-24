package script

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// TengoEngine implements the LanguageEngine interface for Tengo scripts
type TengoEngine struct {
	securityLimits SecurityLimits
}

// NewTengoEngine creates a new Tengo engine with default security limits
func NewTengoEngine() *TengoEngine {
	return &TengoEngine{
		securityLimits: GetDefaultSecurityLimits(),
	}
}

// SetSecurityLimits configures resource and security constraints
func (e *TengoEngine) SetSecurityLimits(limits SecurityLimits) error {
	e.securityLimits = limits
	return nil
}

// Compile prepares a script for execution
func (e *TengoEngine) Compile(script *Script) (*CompiledScript, error) {
	startTime := time.Now()

	// Create a new Tengo script
	tengoScript := tengo.NewScript([]byte(script.Content))

	// Set up allowed modules based on security limits
	modules := e.buildModuleMap()
	tengoScript.SetImports(modules)

	// Note: We don't do test compilation here because Tengo scripts often reference
	// variables that will be provided at execution time. This is normal scripting behavior.

	compilationTime := time.Since(startTime)
	slog.Debug("Tengo script compiled successfully",
		"module", script.ModuleName,
		"script", script.Name,
		"compilation_time", compilationTime,
	)

	return &CompiledScript{
		Script:   script,
		Compiled: tengoScript, // Store the prepared script for execution
	}, nil
}

// Execute runs a compiled script with context
func (e *TengoEngine) Execute(ctx context.Context, compiled *CompiledScript, input *ScriptInput) (*ScriptOutput, error) {
	startTime := time.Now()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	tengoScript, ok := compiled.Compiled.(*tengo.Script)
	if !ok {
		return nil, NewScriptError(
			ErrorTypeExecution,
			compiled.Script.ModuleName,
			compiled.Script.Name,
			"invalid compiled script type for Tengo engine",
			nil,
		)
	}

	// Set up execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.securityLimits.MaxExecutionTime)
	defer cancel()

	// Set up input variables before compilation
	if err := e.setInputVariables(tengoScript, input); err != nil {
		return nil, NewScriptError(
			ErrorTypeExecution,
			compiled.Script.ModuleName,
			compiled.Script.Name,
			"failed to set input variables",
			err,
		)
	}

	// Now compile the script with variables set
	tengoCompiled, err := tengoScript.Compile()
	if err != nil {
		return nil, NewScriptError(
			ErrorTypeCompilation,
			compiled.Script.ModuleName,
			compiled.Script.Name,
			"failed to compile Tengo script with variables",
			err,
		)
	}

	// Execute the script in a goroutine to handle timeouts and panics
	resultChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Convert panic to error
				resultChan <- fmt.Errorf("script panic: %v", r)
			}
		}()
		resultChan <- tengoCompiled.Run()
	}()

	// Wait for execution or timeout
	select {
	case err := <-resultChan:
		if err != nil {
			return nil, NewScriptError(
				ErrorTypeExecution,
				compiled.Script.ModuleName,
				compiled.Script.Name,
				"script execution failed",
				err,
			)
		}
	case <-execCtx.Done():
		return nil, NewScriptError(
			ErrorTypeTimeout,
			compiled.Script.ModuleName,
			compiled.Script.Name,
			"script execution timed out",
			execCtx.Err(),
		)
	}

	// Calculate execution metrics
	executionTime := time.Since(startTime)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	memoryUsed := int64(memAfter.Alloc - memBefore.Alloc)

	// Check memory limits
	if memoryUsed > e.securityLimits.MaxMemoryBytes {
		return nil, NewScriptError(
			ErrorTypeMemoryLimit,
			compiled.Script.ModuleName,
			compiled.Script.Name,
			fmt.Sprintf("script exceeded memory limit: %d bytes > %d bytes", memoryUsed, e.securityLimits.MaxMemoryBytes),
			nil,
		)
	}

	// Extract result
	result := e.extractResult(tengoCompiled)

	// Extract logs (if any were captured)
	logs := e.extractLogs(tengoCompiled)

	return &ScriptOutput{
		Result: result,
		Logs:   logs,
		Metrics: ExecutionMetrics{
			CompilationTime: 0, // Not tracked here, would be from Compile phase
			ExecutionTime:   executionTime,
			MemoryUsed:      memoryUsed,
			Success:         true,
			ErrorType:       "",
		},
		Error: nil,
	}, nil
}

// buildModuleMap creates the allowed modules map based on security limits
func (e *TengoEngine) buildModuleMap() *tengo.ModuleMap {
	modules := tengo.NewModuleMap()

	// Add basic standard library modules that are safe
	for _, pkg := range e.securityLimits.AllowedPackages {
		switch pkg {
		case "fmt":
			if module, exists := stdlib.BuiltinModules["fmt"]; exists {
				modules.AddBuiltinModule("fmt", module)
			}
		case "strings":
			if module, exists := stdlib.BuiltinModules["strings"]; exists {
				modules.AddBuiltinModule("strings", module)
			}
		case "math":
			if module, exists := stdlib.BuiltinModules["math"]; exists {
				modules.AddBuiltinModule("math", module)
			}
		case "rand":
			if module, exists := stdlib.BuiltinModules["rand"]; exists {
				modules.AddBuiltinModule("rand", module)
			}
		}
	}

	return modules
}

// setInputVariables sets up the input context for the script
func (e *TengoEngine) setInputVariables(script *tengo.Script, input *ScriptInput) error {
	if input == nil {
		return nil
	}

	// Set context variables
	if input.Context != nil {
		for key, value := range input.Context {
			if err := script.Add(key, value); err != nil {
				return fmt.Errorf("failed to set context variable %s: %w", key, err)
			}
		}
	}

	// Set message data if available (simplified)
	if input.Message != nil {
		// Set individual message fields as simple variables
		if err := script.Add("message_topic", input.Message.Topic); err != nil {
			return fmt.Errorf("failed to set message_topic variable: %w", err)
		}
		if err := script.Add("message_user_id", input.Message.UserID); err != nil {
			return fmt.Errorf("failed to set message_user_id variable: %w", err)
		}
		if err := script.Add("message_payload", string(input.Message.Payload)); err != nil {
			return fmt.Errorf("failed to set message_payload variable: %w", err)
		}

		// Create a simple message object with basic data
		messageMap := map[string]interface{}{
			"topic":   input.Message.Topic,
			"user_id": input.Message.UserID,
			"payload": string(input.Message.Payload),
		}
		if err := script.Add("message", messageMap); err != nil {
			// If complex object fails, just set a simple string
			script.Add("message", input.Message.Topic)
		}
	}

	// Set HTTP request data if available (simplified)
	if input.HTTPRequest != nil {
		// Set individual HTTP fields as simple variables
		if err := script.Add("http_method", input.HTTPRequest.Method); err != nil {
			return fmt.Errorf("failed to set http_method variable: %w", err)
		}
		if err := script.Add("http_path", input.HTTPRequest.Path); err != nil {
			return fmt.Errorf("failed to set http_path variable: %w", err)
		}
		if err := script.Add("http_body", string(input.HTTPRequest.Body)); err != nil {
			return fmt.Errorf("failed to set http_body variable: %w", err)
		}

		// Create a simple HTTP object with basic data
		httpMap := map[string]interface{}{
			"method": input.HTTPRequest.Method,
			"path":   input.HTTPRequest.Path,
			"body":   string(input.HTTPRequest.Body),
		}
		if err := script.Add("http_request", httpMap); err != nil {
			// If complex object fails, just set a simple string
			script.Add("http_request", input.HTTPRequest.Path)
		}
	}

	// Add custom logging function using Tengo's object system
	if err := e.addLoggingFunction(script); err != nil {
		return fmt.Errorf("failed to add logging function: %w", err)
	}

	return nil
}

// extractResult extracts the result from the executed script
func (e *TengoEngine) extractResult(compiled *tengo.Compiled) interface{} {
	// Try to get a "result" variable first
	if result := compiled.Get("result"); result != nil {
		return result.Value()
	}

	// If no result variable, try to get "return" or the last expression
	if returnVal := compiled.Get("return"); returnVal != nil {
		return returnVal.Value()
	}

	// Return nil if no explicit result
	return nil
}

// extractLogs extracts any log messages from the script execution
func (e *TengoEngine) extractLogs(compiled *tengo.Compiled) []string {
	// Try to get a "logs" variable that scripts might use to collect log messages
	if logsVar := compiled.Get("logs"); logsVar != nil {
		if logs, ok := logsVar.Value().([]interface{}); ok {
			stringLogs := make([]string, len(logs))
			for i, log := range logs {
				stringLogs[i] = fmt.Sprintf("%v", log)
			}
			return stringLogs
		}
	}

	return []string{}
}

// addLoggingFunction adds a custom logging function that integrates with Goby's logging
func (e *TengoEngine) addLoggingFunction(script *tengo.Script) error {
	// Create a custom logging function that integrates with Goby's logging
	logFunc := &tengo.UserFunction{
		Name: "log",
		Value: func(args ...tengo.Object) (tengo.Object, error) {
			if len(args) != 1 {
				return nil, tengo.ErrWrongNumArguments
			}

			// Convert the argument to string
			message := args[0].String()

			// Log to Goby's structured logger
			slog.Info("Script log", "message", message, "source", "tengo_script")

			// Return undefined (Tengo's equivalent of void)
			return tengo.UndefinedValue, nil
		},
	}

	// Add the log function to the script
	return script.Add("log", logFunc)
}
