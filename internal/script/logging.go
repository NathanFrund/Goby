package script

import (
	"context"
	"log/slog"
)

// ScriptLogger provides centralized logging for the script system
type ScriptLogger struct {
	baseFields []slog.Attr
}

// NewScriptLogger creates a new script logger with base fields
func NewScriptLogger() *ScriptLogger {
	return &ScriptLogger{
		baseFields: []slog.Attr{
			slog.String("component", "script_engine"),
		},
	}
}

// LogScriptExecution logs script execution events with consistent structure
func (sl *ScriptLogger) LogScriptExecution(level slog.Level, message string, moduleName, scriptName string, additionalFields ...slog.Attr) {
	fields := make([]slog.Attr, 0, len(sl.baseFields)+3+len(additionalFields))
	fields = append(fields, sl.baseFields...)
	fields = append(fields,
		slog.String("module", moduleName),
		slog.String("script", scriptName),
		slog.String("event_type", "script_execution"),
	)
	fields = append(fields, additionalFields...)

	slog.LogAttrs(context.TODO(), level, message, fields...)
}

// LogScriptError logs script errors with comprehensive context
func (sl *ScriptLogger) LogScriptError(err *ScriptError, additionalContext map[string]interface{}) {
	fields := make([]slog.Attr, 0, len(sl.baseFields)+8)
	fields = append(fields, sl.baseFields...)
	fields = append(fields,
		slog.String("module", err.ModuleName),
		slog.String("script", err.ScriptName),
		slog.String("error_type", string(err.Type)),
		slog.String("error_message", err.Message),
		slog.Time("error_timestamp", err.Timestamp),
		slog.String("event_type", "script_error"),
	)

	// Add cause if available
	if err.Cause != nil {
		fields = append(fields, slog.String("cause", err.Cause.Error()))
	}

	// Add additional context
	for key, value := range additionalContext {
		fields = append(fields, slog.Any(key, value))
	}

	slog.LogAttrs(context.TODO(), slog.LevelError, "Script execution error", fields...)
}

// LogScriptLifecycle logs script lifecycle events (loading, reloading, etc.)
func (sl *ScriptLogger) LogScriptLifecycle(level slog.Level, message string, moduleName, scriptName string, additionalFields ...slog.Attr) {
	fields := make([]slog.Attr, 0, len(sl.baseFields)+3+len(additionalFields))
	fields = append(fields, sl.baseFields...)
	fields = append(fields,
		slog.String("module", moduleName),
		slog.String("script", scriptName),
		slog.String("event_type", "script_lifecycle"),
	)
	fields = append(fields, additionalFields...)

	slog.LogAttrs(context.TODO(), level, message, fields...)
}

// LogSystemEvent logs system-level script events
func (sl *ScriptLogger) LogSystemEvent(level slog.Level, message string, additionalFields ...slog.Attr) {
	fields := make([]slog.Attr, 0, len(sl.baseFields)+1+len(additionalFields))
	fields = append(fields, sl.baseFields...)
	fields = append(fields, slog.String("event_type", "script_system"))
	fields = append(fields, additionalFields...)

	slog.LogAttrs(context.TODO(), level, message, fields...)
}

// LogPerformanceMetrics logs script performance metrics
func (sl *ScriptLogger) LogPerformanceMetrics(moduleName, scriptName string, metrics ExecutionMetrics) {
	fields := make([]slog.Attr, 0, len(sl.baseFields)+8)
	fields = append(fields, sl.baseFields...)
	fields = append(fields,
		slog.String("module", moduleName),
		slog.String("script", scriptName),
		slog.String("event_type", "script_performance"),
		slog.Duration("execution_time", metrics.ExecutionTime),
		slog.Int64("memory_used", metrics.MemoryUsed),
		slog.Bool("success", metrics.Success),
	)

	if metrics.ErrorType != "" {
		fields = append(fields, slog.String("error_type", string(metrics.ErrorType)))
	}

	level := slog.LevelDebug
	if !metrics.Success {
		level = slog.LevelWarn
	}

	slog.LogAttrs(context.TODO(), level, "Script execution metrics", fields...)
}

// LogHotReload logs hot-reload events
func (sl *ScriptLogger) LogHotReload(action string, moduleName, scriptName, filePath string, success bool, err error) {
	fields := make([]slog.Attr, 0, len(sl.baseFields)+6)
	fields = append(fields, sl.baseFields...)
	fields = append(fields,
		slog.String("module", moduleName),
		slog.String("script", scriptName),
		slog.String("file_path", filePath),
		slog.String("action", action),
		slog.String("event_type", "hot_reload"),
		slog.Bool("success", success),
	)

	if err != nil {
		fields = append(fields, slog.String("error", err.Error()))
	}

	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}

	message := "Script hot-reload " + action
	slog.LogAttrs(context.TODO(), level, message, fields...)
}

// Global script logger instance
var scriptLogger = NewScriptLogger()

// Convenience functions for common logging operations

// LogExecution logs a script execution event
func LogExecution(level slog.Level, message string, moduleName, scriptName string, additionalFields ...slog.Attr) {
	scriptLogger.LogScriptExecution(level, message, moduleName, scriptName, additionalFields...)
}

// LogError logs a script error
func LogError(err *ScriptError, additionalContext map[string]interface{}) {
	scriptLogger.LogScriptError(err, additionalContext)
}

// LogLifecycle logs a script lifecycle event
func LogLifecycle(level slog.Level, message string, moduleName, scriptName string, additionalFields ...slog.Attr) {
	scriptLogger.LogScriptLifecycle(level, message, moduleName, scriptName, additionalFields...)
}

// LogSystem logs a system-level event
func LogSystem(level slog.Level, message string, additionalFields ...slog.Attr) {
	scriptLogger.LogSystemEvent(level, message, additionalFields...)
}

// LogPerformance logs script performance metrics
func LogPerformance(moduleName, scriptName string, metrics ExecutionMetrics) {
	scriptLogger.LogPerformanceMetrics(moduleName, scriptName, metrics)
}

// LogHotReloadEvent logs hot-reload events
func LogHotReloadEvent(action string, moduleName, scriptName, filePath string, success bool, err error) {
	scriptLogger.LogHotReload(action, moduleName, scriptName, filePath, success, err)
}
