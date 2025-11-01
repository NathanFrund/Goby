package script

import (
	"context"
	"testing"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ContextKey is used for context value keys
type contextKey string

// ContextKeyInput is the key used to store script input in the context
const ContextKeyInput contextKey = "input"

func TestScriptExecutor_ExecuteMessageHandler(t *testing.T) {
	// Setup
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	// Create a simple script that returns a fixed result
	scriptContent := `result := "test.topic_processed"`

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"message_handler.tengo": scriptContent,
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background(), true)
	require.NoError(t, err)

	// Create script config
	config := &ModuleScriptConfig{
		MessageHandlers: map[string]string{
			"test.topic": "message_handler.tengo",
		},
		EndpointScripts: make(map[string]string),
		DefaultLimits:   GetDefaultSecurityLimits(),
		AutoExtract:     false,
	}

	executor := NewScriptExecutor(engine, "test_module", config)

	// Create test message
	msg := &pubsub.Message{
		Topic:   "test.topic",
		Payload: []byte(`{"data":"test message"}`),
	}

	// Create context with input
	ctx := context.Background()
	input := &ScriptInput{
		Context: map[string]interface{}{
			"topic": "test.topic",
			"data":  "test message",
		},
	}
	ctx = context.WithValue(ctx, ContextKeyInput, input)

	// Execute message handler
	output, err := executor.ExecuteMessageHandler(ctx, "test.topic", msg, nil)
	require.NoError(t, err)

	// The script returns a ScriptOutput, so we need to check the Result field
	if output != nil {
		result, ok := output.Result.(string)
		require.True(t, ok, "expected result to be a string")
		assert.Equal(t, "test.topic_processed", result)
	} else {
		t.Fatal("expected non-nil output from ExecuteMessageHandler")
	}
}

func TestScriptExecutor_ExecuteEndpointScript(t *testing.T) {
	// Setup
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"endpoint_handler": "result := http_request.method + '_' + http_request.path",
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background(), true)
	require.NoError(t, err)

	// Create script config
	config := &ModuleScriptConfig{
		MessageHandlers: make(map[string]string),
		EndpointScripts: map[string]string{
			"/api/test": "endpoint_handler",
		},
		DefaultLimits: GetDefaultSecurityLimits(),
		AutoExtract:   false,
	}

	executor := NewScriptExecutor(engine, "test_module", config)

	// Create test HTTP request
	httpRequest := &HTTPRequestData{
		Method:  "POST",
		Path:    "/api/test",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"test": "data"}`),
		Query:   map[string]string{"param": "value"},
	}

	// Execute endpoint script
	output, err := executor.ExecuteEndpointScript(context.Background(), "/api/test", httpRequest, nil)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "POST_/api/test", output.Result)
}

func TestScriptExecutor_ExecuteScript(t *testing.T) {
	// Setup
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"custom_script": "result := input_value * multiplier",
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background(), true)
	require.NoError(t, err)

	config := GetDefaultModuleScriptConfig()
	executor := NewScriptExecutor(engine, "test_module", config)

	// Execute custom script
	scriptContext := map[string]interface{}{
		"input_value": 10,
		"multiplier":  3,
	}

	output, err := executor.ExecuteScript(context.Background(), "custom_script", scriptContext, nil)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, int64(30), output.Result)
}

func TestScriptExecutor_NoHandlerConfigured(t *testing.T) {
	// Setup
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})
	err := engine.Initialize(context.Background(), true)
	require.NoError(t, err)

	config := GetDefaultModuleScriptConfig()
	executor := NewScriptExecutor(engine, "test_module", config)

	// Create test message
	message := &pubsub.Message{
		Topic:   "unconfigured.topic",
		UserID:  "user123",
		Payload: []byte("test payload"),
	}

	// Execute message handler for unconfigured topic
	output, err := executor.ExecuteMessageHandler(context.Background(), "unconfigured.topic", message, nil)
	require.NoError(t, err)
	assert.Nil(t, output) // Should return nil when no handler is configured
}

func TestGetDefaultModuleScriptConfig(t *testing.T) {
	config := GetDefaultModuleScriptConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.MessageHandlers)
	assert.NotNil(t, config.EndpointScripts)
	assert.Equal(t, GetDefaultSecurityLimits(), config.DefaultLimits)
	assert.False(t, config.AutoExtract)
}

func TestModuleScriptHelper_RegisterEmbeddedScripts(t *testing.T) {
	// Setup
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	config := GetDefaultModuleScriptConfig()
	helper := NewModuleScriptHelper(engine, "test_module", config)

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"test_script": "result := 42",
		},
	}

	// Register embedded scripts
	helper.RegisterEmbeddedScripts(provider)

	// Initialize engine to load scripts
	err := engine.Initialize(context.Background(), true)
	require.NoError(t, err)

	// Verify script was registered
	script, err := engine.GetScript("test_module", "test_script")
	require.NoError(t, err)
	assert.Equal(t, "test_module", script.ModuleName)
	assert.Equal(t, "test_script", script.Name)
}

func TestModuleScriptHelper_GetExecutor(t *testing.T) {
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	config := GetDefaultModuleScriptConfig()
	helper := NewModuleScriptHelper(engine, "test_module", config)

	executor := helper.GetExecutor()
	assert.NotNil(t, executor)
	assert.Equal(t, "test_module", executor.moduleName)
}
