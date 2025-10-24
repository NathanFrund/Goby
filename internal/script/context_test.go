package script

import (
	"context"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextManager_CreateExecutionContext(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	cm := NewContextManager(5, limits)

	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "test_script",
		Input: &ScriptInput{
			Context: map[string]interface{}{
				"test_var": "test_value",
			},
		},
	}

	execCtx, err := cm.CreateExecutionContext(req, "user123", "req456")
	require.NoError(t, err)
	assert.NotNil(t, execCtx)
	assert.Equal(t, "test_module", execCtx.ModuleName)
	assert.Equal(t, "test_script", execCtx.ScriptName)
	assert.Equal(t, "user123", execCtx.UserID)
	assert.Equal(t, "req456", execCtx.RequestID)
	assert.Contains(t, execCtx.Variables, "test_var")
	assert.Equal(t, "test_value", execCtx.Variables["test_var"])
}

func TestContextManager_ConcurrentLimit(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	cm := NewContextManager(2, limits) // Limit to 2 concurrent executions

	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "test_script",
	}

	// Create first context
	ctx1, err := cm.CreateExecutionContext(req, "user1", "req1")
	require.NoError(t, err)
	assert.NotNil(t, ctx1)

	// Create second context
	ctx2, err := cm.CreateExecutionContext(req, "user2", "req2")
	require.NoError(t, err)
	assert.NotNil(t, ctx2)

	// Third context should fail due to limit
	_, err = cm.CreateExecutionContext(req, "user3", "req3")
	require.Error(t, err)
	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeExecution, scriptErr.Type)

	// Release one context and try again
	cm.ReleaseExecutionContext(ctx1.ID)
	ctx3, err := cm.CreateExecutionContext(req, "user3", "req3")
	require.NoError(t, err)
	assert.NotNil(t, ctx3)
}

func TestExecutionContext_StandardFunctions(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	cm := NewContextManager(5, limits)

	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "test_script",
	}

	execCtx, err := cm.CreateExecutionContext(req, "user123", "req456")
	require.NoError(t, err)

	// Test log function
	logFn, exists := execCtx.Functions["log"]
	assert.True(t, exists)
	assert.NotNil(t, logFn)

	// Call log function
	if logFunc, ok := logFn.(func(string)); ok {
		logFunc("test message")
		logs := execCtx.GetLogs()
		assert.Len(t, logs, 1)
		assert.Contains(t, logs[0], "test message")
	}

	// Test context info function
	ctxFn, exists := execCtx.Functions["get_context"]
	assert.True(t, exists)
	assert.NotNil(t, ctxFn)

	// Test time functions
	nowFn, exists := execCtx.Functions["now"]
	assert.True(t, exists)
	assert.NotNil(t, nowFn)

	timestampFn, exists := execCtx.Functions["timestamp"]
	assert.True(t, exists)
	assert.NotNil(t, timestampFn)
}

func TestExecutionContext_MessageData(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	cm := NewContextManager(5, limits)

	message := &pubsub.Message{
		Topic:   "test.topic",
		UserID:  "user123",
		Payload: []byte("test payload"),
		Metadata: map[string]string{
			"timestamp": "2023-01-01T00:00:00Z",
		},
	}

	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "test_script",
		Input: &ScriptInput{
			Message: message,
		},
	}

	execCtx, err := cm.CreateExecutionContext(req, "user123", "req456")
	require.NoError(t, err)

	// Check message data was populated
	messageVar, exists := execCtx.Variables["message"]
	assert.True(t, exists)

	messageMap, ok := messageVar.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test.topic", messageMap["topic"])
	assert.Equal(t, "user123", messageMap["user_id"])
	assert.Equal(t, "test payload", messageMap["payload"])
}

func TestExecutionContext_HTTPRequestData(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	cm := NewContextManager(5, limits)

	httpReq := &HTTPRequestData{
		Method:  "POST",
		Path:    "/api/test",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"test": "data"}`),
		Query:   map[string]string{"param": "value"},
	}

	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "test_script",
		Input: &ScriptInput{
			HTTPRequest: httpReq,
		},
	}

	execCtx, err := cm.CreateExecutionContext(req, "user123", "req456")
	require.NoError(t, err)

	// Check HTTP request data was populated
	httpVar, exists := execCtx.Variables["http_request"]
	assert.True(t, exists)

	httpMap, ok := httpVar.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "POST", httpMap["method"])
	assert.Equal(t, "/api/test", httpMap["path"])
	assert.Equal(t, `{"test": "data"}`, httpMap["body"])
}

func TestSecurityValidator_ValidateContext(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	validator := NewSecurityValidator(limits)

	// Create a context with reasonable data
	execCtx := &ExecutionContext{
		ModuleName:     "test_module",
		ScriptName:     "test_script",
		SecurityLimits: limits,
		Variables: map[string]interface{}{
			"small_var": "small value",
			"number":    42,
		},
	}

	err := validator.ValidateContext(execCtx)
	assert.NoError(t, err)

	// Create a context with oversized variable
	execCtx.Variables["huge_var"] = make([]byte, limits.MaxMemoryBytes)
	err = validator.ValidateContext(execCtx)
	require.Error(t, err)
	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeSecurityViolation, scriptErr.Type)
}

func TestContextAwareEngine_ExecuteWithContext(t *testing.T) {
	// Setup base engine
	cfg := &MockConfig{}
	baseEngine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"test_script": "result := test_var + \"_processed\"",
		},
	}

	baseEngine.RegisterEmbeddedProvider(provider)
	err := baseEngine.Initialize(context.Background(), true)
	require.NoError(t, err)

	// Create context-aware engine
	contextEngine := NewContextAwareEngine(baseEngine, 5)

	// Execute with context
	req := EnhancedExecutionRequest{
		ExecutionRequest: ExecutionRequest{
			ModuleName: "test_module",
			ScriptName: "test_script",
			Input: &ScriptInput{
				Context: map[string]interface{}{
					"test_var": "hello",
				},
			},
		},
		UserID:    "user123",
		RequestID: "req456",
		Context:   context.Background(),
	}

	output, err := contextEngine.ExecuteWithContext(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "hello_processed", output.Result)
}

func TestContextAwareEngine_Timeout(t *testing.T) {
	// Setup base engine
	cfg := &MockConfig{}
	baseEngine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"infinite_script": `
				for true {
					// Infinite loop
				}
			`,
		},
	}

	baseEngine.RegisterEmbeddedProvider(provider)
	err := baseEngine.Initialize(context.Background(), true)
	require.NoError(t, err)

	// Create context-aware engine
	contextEngine := NewContextAwareEngine(baseEngine, 5)

	// Execute with very short timeout
	limits := GetDefaultSecurityLimits()
	limits.MaxExecutionTime = 10 * time.Millisecond

	req := EnhancedExecutionRequest{
		ExecutionRequest: ExecutionRequest{
			ModuleName:     "test_module",
			ScriptName:     "infinite_script",
			SecurityLimits: limits,
		},
		UserID:    "user123",
		RequestID: "req456",
		Context:   context.Background(),
	}

	_, err = contextEngine.ExecuteWithContext(context.Background(), req)
	require.Error(t, err)
	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeTimeout, scriptErr.Type)
}

func TestContextManager_GetActiveContexts(t *testing.T) {
	limits := GetDefaultSecurityLimits()
	cm := NewContextManager(5, limits)

	req := ExecutionRequest{
		ModuleName: "test_module",
		ScriptName: "test_script",
	}

	// Create a context
	execCtx, err := cm.CreateExecutionContext(req, "user123", "req456")
	require.NoError(t, err)

	// Get active contexts
	activeContexts := cm.GetActiveContexts()
	assert.Len(t, activeContexts, 1)
	assert.Contains(t, activeContexts, execCtx.ID)

	contextInfo := activeContexts[execCtx.ID]
	assert.Equal(t, "test_module", contextInfo.ModuleName)
	assert.Equal(t, "test_script", contextInfo.ScriptName)
	assert.Equal(t, "user123", contextInfo.UserID)
	assert.Equal(t, "req456", contextInfo.RequestID)

	// Release context
	cm.ReleaseExecutionContext(execCtx.ID)
	activeContexts = cm.GetActiveContexts()
	assert.Len(t, activeContexts, 0)
}

func TestEstimateSize(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected int64
	}{
		{"string", "hello", 5},
		{"bytes", []byte("hello"), 5},
		{"number", 42, 64},
		{"map", map[string]interface{}{"key": "value"}, 8}, // 3 + 5
		{"slice", []interface{}{"a", "b"}, 2},              // 1 + 1
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size := estimateSize(tc.value)
			assert.Equal(t, tc.expected, size)
		})
	}
}
