package script

import (
	"context"
	"testing"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScriptExecutor_ExecuteMessageHandler(t *testing.T) {
	// Setup
	cfg := &MockConfig{}
	engine := NewEngine(Dependencies{Config: cfg})

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"message_handler": "result := message.topic + '_processed'",
		},
	}

	engine.RegisterEmbeddedProvider(provider)
	err := engine.Initialize(context.Background(), true)
	require.NoError(t, err)

	// Create script config
	config := &ModuleScriptConfig{
		MessageHandlers: map[string]string{
			"test.topic": "message_handler",
		},
		EndpointScripts: make(map[string]string),
		DefaultLimits:   GetDefaultSecurityLimits(),
		AutoExtract:     false,
	}

	executor := NewScriptExecutor(engine, "test_module", config)

	// Create test message
	message := &pubsub.Message{
		Topic:    "test.topic",
		UserID:   "user123",
		Payload:  []byte("test payload"),
		Metadata: map[string]string{"timestamp": "2023-01-01T00:00:00Z"},
	}

	// Execute message handler
	output, err := executor.ExecuteMessageHandler(context.Background(), "test.topic", message, nil)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "test.topic_processed", output.Result)
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
	assert.Equal(t, 30, output.Result)
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
