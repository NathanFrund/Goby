package script

import (
	"context"
	"fmt"
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
	assert.Equal(t, int64(5), output.Result)
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
	assert.Equal(t, int64(30), output.Result)
}

func TestTengoEngine_CompilationError(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "invalid",
		Language:   LanguageTengo,
		Content:    `result := undefined_variable`, // This should fail compilation
	}

	_, err := engine.Compile(testScript)
	require.Error(t, err)

	var scriptErr *ScriptError
	assert.ErrorAs(t, err, &scriptErr)
	assert.Equal(t, ErrorTypeCompilation, scriptErr.Type)
	assert.Contains(t, err.Error(), "undefined_variable")
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

func TestTengoEngine_SetSecurityLimits(t *testing.T) {
	engine := NewTengoEngine()

	limits := SecurityLimits{
		MaxExecutionTime: 5 * time.Second,
		MaxMemoryBytes:   1024 * 1024, // 1MB
	}

	err := engine.SetSecurityLimits(limits)
	assert.NoError(t, err)

	// Verify limits are applied by checking they don't cause immediate failure
	testScript := &Script{
		ModuleName: "test",
		Name:       "simple",
		Language:   LanguageTengo,
		Content:    `result := "test"`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	output, err := engine.Execute(context.Background(), compiled, &ScriptInput{})
	require.NoError(t, err)
	assert.Equal(t, "test", output.Result)
}

func TestTengoEngine_ComplexScript(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "complex",
		Language:   LanguageTengo,
		Content: `
			// Test complex operations
			numbers := [1, 2, 3, 4, 5]
			sum := 0
			for i := 0; i < len(numbers); i++ {
				sum += numbers[i]
			}
			
			// Test string operations
			message := "Hello, " + name + "!"
			
			// Test conditional logic
			status := ""
			if sum > 10 {
				status = "high"
			} else {
				status = "low"
			}
			
			result := {
				sum: sum,
				message: message,
				status: status,
				count: len(numbers)
			}
		`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	input := &ScriptInput{
		Context: map[string]interface{}{
			"name": "World",
		},
	}

	output, err := engine.Execute(context.Background(), compiled, input)
	require.NoError(t, err)

	result, ok := output.Result.(map[string]interface{})
	require.True(t, ok, "Expected result to be a map")

	assert.Equal(t, int64(15), result["sum"])
	assert.Equal(t, "Hello, World!", result["message"])
	assert.Equal(t, "high", result["status"])
	assert.Equal(t, int64(5), result["count"])
}

func TestTengoEngine_ErrorHandling(t *testing.T) {
	engine := NewTengoEngine()

	testCases := []struct {
		name        string
		script      string
		expectError bool
		errorType   ErrorType
	}{
		{
			name:        "compilation_error_undefined",
			script:      `result := undefined_variable`,
			expectError: true,
			errorType:   ErrorTypeCompilation, // Caught during Execute when we compile with context
		},
		{
			name:        "syntax_error",
			script:      `result := {`, // Incomplete object
			expectError: true,
			errorType:   ErrorTypeCompilation,
		},
		{
			name:        "valid_script",
			script:      `result := 42`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testScript := &Script{
				ModuleName: "test",
				Name:       tc.name,
				Language:   LanguageTengo,
				Content:    tc.script,
			}

			compiled, compileErr := engine.Compile(testScript)

			if tc.expectError && tc.errorType == ErrorTypeCompilation {
				require.Error(t, compileErr)
				var scriptErr *ScriptError
				assert.ErrorAs(t, compileErr, &scriptErr)
				assert.Equal(t, tc.errorType, scriptErr.Type)
				return
			}

			require.NoError(t, compileErr)

			output, execErr := engine.Execute(context.Background(), compiled, &ScriptInput{})

			if tc.expectError && tc.errorType == ErrorTypeExecution {
				require.Error(t, execErr)
				var scriptErr *ScriptError
				assert.ErrorAs(t, execErr, &scriptErr)
				assert.Equal(t, tc.errorType, scriptErr.Type)
			} else {
				require.NoError(t, execErr)
				assert.NotNil(t, output)
			}
		})
	}
}

func TestTengoEngine_ContextCancellation(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "long_running",
		Language:   LanguageTengo,
		Content: `
			// Simulate some work
			sum := 0
			for i := 0; i < 1000000; i++ {
				sum += i
			}
			result := sum
		`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	_, err = engine.Execute(ctx, compiled, &ScriptInput{})

	// Should get a context cancellation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestTengoEngine_EmptyScript(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "empty",
		Language:   LanguageTengo,
		Content:    "",
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	output, err := engine.Execute(context.Background(), compiled, &ScriptInput{})
	require.NoError(t, err)
	assert.Nil(t, output.Result) // Empty script should return nil
}

func TestTengoEngine_LargeContext(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "large_context",
		Language:   LanguageTengo,
		Content: `
			total := 0
			for key, value in data {
				if is_int(value) {
					total += value
				}
			}
			result := total
		`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	// Create a large context
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = i
	}

	input := &ScriptInput{
		Context: map[string]interface{}{
			"data": largeData,
		},
	}

	output, err := engine.Execute(context.Background(), compiled, input)
	require.NoError(t, err)

	// Sum of 0 to 999 = 999 * 1000 / 2 = 499500
	assert.Equal(t, int64(499500), output.Result)
}

func TestTengoEngine_MetricsTracking(t *testing.T) {
	engine := NewTengoEngine()

	testScript := &Script{
		ModuleName: "test",
		Name:       "metrics",
		Language:   LanguageTengo,
		Content:    `result := 42`,
	}

	compiled, err := engine.Compile(testScript)
	require.NoError(t, err)

	output, err := engine.Execute(context.Background(), compiled, &ScriptInput{})
	require.NoError(t, err)

	// Verify metrics are populated
	assert.True(t, output.Metrics.Success)
	assert.Greater(t, output.Metrics.ExecutionTime, time.Duration(0))
	assert.Greater(t, output.Metrics.MemoryUsed, int64(0))
	// Note: ScriptName and ModuleName are not part of ExecutionMetrics
	// They are available in the ScriptError when errors occur
}
