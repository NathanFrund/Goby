package script

import (
	"context"
	"log/slog"

	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
)

// ScriptableModule extends the base Module interface for script support
type ScriptableModule interface {
	module.Module

	// GetScriptConfig returns script configuration for this module
	GetScriptConfig() *ModuleScriptConfig

	// GetExposedFunctions returns functions available to scripts
	GetExposedFunctions() map[string]interface{}
}

// ModuleScriptConfig defines script behavior for a module
type ModuleScriptConfig struct {
	// Scripts that should be executed on pub/sub messages
	MessageHandlers map[string]string // topic -> script name

	// Scripts available for HTTP endpoint execution
	EndpointScripts map[string]string // endpoint -> script name

	// Default security limits for this module's scripts
	DefaultLimits SecurityLimits

	// Whether to auto-extract embedded scripts on startup
	AutoExtract bool
}

// ScriptExecutor provides helper methods for modules to execute scripts
type ScriptExecutor struct {
	engine     ScriptEngine
	moduleName string
	config     *ModuleScriptConfig
}

// NewScriptExecutor creates a new script executor for a module
func NewScriptExecutor(engine ScriptEngine, moduleName string, config *ModuleScriptConfig) *ScriptExecutor {
	return &ScriptExecutor{
		engine:     engine,
		moduleName: moduleName,
		config:     config,
	}
}

// ExecuteMessageHandler executes a script in response to a pub/sub message
func (se *ScriptExecutor) ExecuteMessageHandler(ctx context.Context, topic string, message *pubsub.Message, exposedFunctions map[string]interface{}) (*ScriptOutput, error) {
	// Find the script for this topic
	scriptName, exists := se.config.MessageHandlers[topic]
	if !exists {
		slog.Debug("No script handler configured for topic", "module", se.moduleName, "topic", topic)
		return nil, nil
	}

	// Prepare script input
	input := &ScriptInput{
		Context: map[string]interface{}{
			"topic":      topic,
			"module":     se.moduleName,
			"timestamp":  message.Metadata["timestamp"],
		},
		Message:   message,
		Functions: exposedFunctions,
	}

	// Execute the script
	req := ExecutionRequest{
		ModuleName:     se.moduleName,
		ScriptName:     scriptName,
		Input:          input,
		SecurityLimits: se.config.DefaultLimits,
	}

	output, err := se.engine.Execute(ctx, req)
	if err != nil {
		slog.Error("Script execution failed for message handler",
			"module", se.moduleName,
			"topic", topic,
			"script", scriptName,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("Message handler script executed successfully",
		"module", se.moduleName,
		"topic", topic,
		"script", scriptName,
		"execution_time", output.Metrics.ExecutionTime,
	)

	return output, nil
}

// ExecuteEndpointScript executes a script for an HTTP endpoint
func (se *ScriptExecutor) ExecuteEndpointScript(ctx context.Context, endpoint string, httpRequest *HTTPRequestData, exposedFunctions map[string]interface{}) (*ScriptOutput, error) {
	// Find the script for this endpoint
	scriptName, exists := se.config.EndpointScripts[endpoint]
	if !exists {
		slog.Debug("No script configured for endpoint", "module", se.moduleName, "endpoint", endpoint)
		return nil, nil
	}

	// Prepare script input
	input := &ScriptInput{
		Context: map[string]interface{}{
			"endpoint": endpoint,
			"module":   se.moduleName,
		},
		HTTPRequest: httpRequest,
		Functions:   exposedFunctions,
	}

	// Execute the script
	req := ExecutionRequest{
		ModuleName:     se.moduleName,
		ScriptName:     scriptName,
		Input:          input,
		SecurityLimits: se.config.DefaultLimits,
	}

	output, err := se.engine.Execute(ctx, req)
	if err != nil {
		slog.Error("Script execution failed for endpoint handler",
			"module", se.moduleName,
			"endpoint", endpoint,
			"script", scriptName,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("Endpoint script executed successfully",
		"module", se.moduleName,
		"endpoint", endpoint,
		"script", scriptName,
		"execution_time", output.Metrics.ExecutionTime,
	)

	return output, nil
}

// ExecuteScript executes a named script with custom context
func (se *ScriptExecutor) ExecuteScript(ctx context.Context, scriptName string, scriptContext map[string]interface{}, exposedFunctions map[string]interface{}) (*ScriptOutput, error) {
	// Prepare script input
	input := &ScriptInput{
		Context:   scriptContext,
		Functions: exposedFunctions,
	}

	// Execute the script
	req := ExecutionRequest{
		ModuleName:     se.moduleName,
		ScriptName:     scriptName,
		Input:          input,
		SecurityLimits: se.config.DefaultLimits,
	}

	output, err := se.engine.Execute(ctx, req)
	if err != nil {
		slog.Error("Script execution failed",
			"module", se.moduleName,
			"script", scriptName,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("Script executed successfully",
		"module", se.moduleName,
		"script", scriptName,
		"execution_time", output.Metrics.ExecutionTime,
	)

	return output, nil
}

// GetDefaultModuleScriptConfig returns a default configuration for modules
func GetDefaultModuleScriptConfig() *ModuleScriptConfig {
	return &ModuleScriptConfig{
		MessageHandlers: make(map[string]string),
		EndpointScripts: make(map[string]string),
		DefaultLimits:   GetDefaultSecurityLimits(),
		AutoExtract:     false, // Don't auto-extract by default
	}
}

// ModuleScriptHelper provides utility functions for modules implementing ScriptableModule
type ModuleScriptHelper struct {
	engine   ScriptEngine
	executor *ScriptExecutor
}

// NewModuleScriptHelper creates a new helper for a module
func NewModuleScriptHelper(engine ScriptEngine, moduleName string, config *ModuleScriptConfig) *ModuleScriptHelper {
	return &ModuleScriptHelper{
		engine:   engine,
		executor: NewScriptExecutor(engine, moduleName, config),
	}
}

// RegisterEmbeddedScripts registers embedded scripts with the script engine
func (msh *ModuleScriptHelper) RegisterEmbeddedScripts(provider EmbeddedScriptProvider) {
	if engine, ok := msh.engine.(*Engine); ok {
		engine.RegisterEmbeddedProvider(provider)
		slog.Debug("Registered embedded scripts", "module", provider.GetModuleName())
	}
}

// GetExecutor returns the script executor for this module
func (msh *ModuleScriptHelper) GetExecutor() *ScriptExecutor {
	return msh.executor
}

// ExtractScriptsIfConfigured extracts embedded scripts if auto-extract is enabled
func (msh *ModuleScriptHelper) ExtractScriptsIfConfigured(config *ModuleScriptConfig, targetDir string) error {
	if config.AutoExtract {
		slog.Info("Auto-extracting embedded scripts", "target_dir", targetDir)
		return msh.engine.ExtractDefaultScripts(targetDir)
	}
	return nil
}