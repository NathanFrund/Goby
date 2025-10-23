package script

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTengoEngine_Compile(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "simple",
		Language:   LanguageTengo,
		Content:    `result := 2 + 3`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)
	assert.NotNil(t, compiled)
	assert.Equal(t, testScript, compiled.Script)
}

func TestTengoEngine_Execute(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "simple",
		Language:   LanguageTengo,
		Content:    `result := 2 + 3`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	input := &ScriptInput{
		Context: map[string]interface{}{
			"multiplier": 2,
		},
	}

	output, err := engine.Execute(context.Background(), compiled, input)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, 5, output.Result)
	assert.True(t, output.Metrics.Success)
	assert.Greater(t, output.Metrics.ExecutionTime, time.Duration(0))
}

func TestTengoEngine_ExecuteWithContext(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "with_context",
		Language:   LanguageTengo,
		Content:    `result := base_value * multiplier`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	input := &ScriptInput{
		Context: map[string]interface{}{
			"base_value": 10,
			"multiplier": 3,
		},
	}

	output, err := engine.Execute(context.Background(), compiled, input)
	require.NoError(t, err)
	assert.Equal(t, 30, output.Result)
}

func TestTengoEngine_CompilationError(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "invalid",
		Language:   LanguageTengo,
		Content:    `invalid syntax here !!!`,
	}

	_, err := engine.Compile(testScript)
	require.Error(t, err)

	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeCompilation, scriptErr.Type)
}

func TestTengoEngine_Timeout(t *testing.T) {
	engine := NewTengoEngine()

	// Set a very short timeout
	limits := GetDefaultSecurityLimits()
	limits.MaxExecutionTime = 1 * time.Millisecond
	engine.SetSecurityLimits(limits)

	testScript := &Script{
		ModuleName: "test",
		Name:       "infinite_loop",
		Language:   LanguageTengo,
		Content: `
			for true {
				// Infinite loop to trigger timeout
			}
		`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	_, err = engine.Execute(context.Background(), compiled, &ScriptInput{})
	require.Error(t, err)

	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeTimeout, scriptErr.Type)
}