package script

import (
	"time"

	"github.com/nfrund/goby/internal/pubsub"
)

// ScriptLanguage represents supported scripting languages
type ScriptLanguage string

const (
	LanguageTengo   ScriptLanguage = "tengo"
	LanguageZygomys ScriptLanguage = "zygomys"
)

// ScriptSource indicates where a script was loaded from
type ScriptSource string

const (
	SourceEmbedded ScriptSource = "embedded"
	SourceExternal ScriptSource = "external"
)

// ErrorType categorizes different types of script errors
type ErrorType string

const (
	ErrorTypeCompilation       ErrorType = "compilation"
	ErrorTypeExecution         ErrorType = "execution"
	ErrorTypeTimeout           ErrorType = "timeout"
	ErrorTypeMemoryLimit       ErrorType = "memory_limit"
	ErrorTypeSecurityViolation ErrorType = "security_violation"
	ErrorTypeNotFound          ErrorType = "not_found"
	ErrorTypeInvalidSyntax     ErrorType = "invalid_syntax"
)

// Script represents a script file with metadata
type Script struct {
	ModuleName       string
	Name             string
	Language         ScriptLanguage
	Content          string
	Source           ScriptSource
	LastModified     time.Time
	Checksum         string
	OriginalLanguage ScriptLanguage // tracks the embedded script's language for fallback
}

// ExecutionRequest contains all data needed to execute a script
type ExecutionRequest struct {
	ModuleName     string
	ScriptName     string
	Input          *ScriptInput
	Timeout        time.Duration
	SecurityLimits SecurityLimits
}

// ScriptInput provides context and data to the executing script
type ScriptInput struct {
	// Context data from the calling module
	Context map[string]interface{}

	// Message data if triggered by pub/sub
	Message *pubsub.Message

	// HTTP request data if triggered by handler
	HTTPRequest *HTTPRequestData

	// Available functions exposed to the script
	Functions map[string]interface{}
}

// HTTPRequestData contains HTTP request information for scripts
type HTTPRequestData struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
	Query   map[string]string
}

// ScriptOutput contains the results of script execution
type ScriptOutput struct {
	Result  interface{}
	Logs    []string
	Metrics ExecutionMetrics
	Error   error
}

// ExecutionMetrics tracks performance and execution data
type ExecutionMetrics struct {
	CompilationTime time.Duration
	ExecutionTime   time.Duration
	MemoryUsed      int64
	Success         bool
	ErrorType       ErrorType
}

// SecurityLimits defines resource constraints for script execution
type SecurityLimits struct {
	MaxExecutionTime time.Duration
	MaxMemoryBytes   int64
	AllowedPackages  []string
	ExposedFunctions map[string]interface{}
}

// CompiledScript represents a compiled script ready for execution
type CompiledScript struct {
	Script   *Script
	Compiled interface{} // language-specific compiled representation
}

// ScriptError represents script-related errors with context
type ScriptError struct {
	Type       ErrorType
	ModuleName string
	ScriptName string
	Message    string
	Cause      error
	Timestamp  time.Time
}

func (e *ScriptError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *ScriptError) Unwrap() error {
	return e.Cause
}

// NewScriptError creates a new ScriptError with the given parameters
func NewScriptError(errorType ErrorType, moduleName, scriptName, message string, cause error) *ScriptError {
	return &ScriptError{
		Type:       errorType,
		ModuleName: moduleName,
		ScriptName: scriptName,
		Message:    message,
		Cause:      cause,
		Timestamp:  time.Now(),
	}
}