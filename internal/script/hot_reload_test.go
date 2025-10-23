package script

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_HotReloading(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_hot_reload_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create initial script file
	scriptPath := "scripts/test_module/dynamic_script.tengo"
	err = os.WriteFile(scriptPath, []byte("result := 1"), 0644)
	require.NoError(t, err)

	// Create registry and load scripts
	registry := NewRegistry()
	err = registry.LoadScripts()
	require.NoError(t, err)

	// Start watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = registry.StartWatcher(ctx)
	require.NoError(t, err)

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Get initial script
	script, err := registry.GetScript("test_module", "dynamic_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 1", script.Content)

	// Modify the script file
	err = os.WriteFile(scriptPath, []byte("result := 2"), 0644)
	require.NoError(t, err)

	// Give watcher time to detect and process the change
	time.Sleep(200 * time.Millisecond)

	// Get updated script (should be reloaded automatically)
	script, err = registry.GetScript("test_module", "dynamic_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 2", script.Content)

	// Stop watcher
	registry.StopWatcher()
}

func TestRegistry_HotReloadingNewFile(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_hot_reload_new_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create registry and start watcher
	registry := NewRegistry()
	err = registry.LoadScripts()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = registry.StartWatcher(ctx)
	require.NoError(t, err)

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Initially, script should not exist
	_, err = registry.GetScript("test_module", "new_script")
	assert.Error(t, err)

	// Create new script file
	scriptPath := "scripts/test_module/new_script.tengo"
	err = os.WriteFile(scriptPath, []byte("result := 42"), 0644)
	require.NoError(t, err)

	// Give watcher time to detect and process the new file
	time.Sleep(200 * time.Millisecond)

	// New script should now be available
	script, err := registry.GetScript("test_module", "new_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 42", script.Content)
	assert.Equal(t, SourceExternal, script.Source)

	// Stop watcher
	registry.StopWatcher()
}

func TestRegistry_HotReloadingDeleteFile(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "script_hot_reload_delete_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create scripts directory
	err = os.MkdirAll("scripts/test_module", 0755)
	require.NoError(t, err)

	// Create external script that will override embedded
	scriptPath := "scripts/test_module/test_script.tengo"
	err = os.WriteFile(scriptPath, []byte("result := 999"), 0644)
	require.NoError(t, err)

	// Create registry with embedded script
	registry := NewRegistry()

	provider := &MockEmbeddedScriptProvider{
		moduleName: "test_module",
		scripts: map[string]string{
			"test_script": "result := 42",
		},
	}
	registry.RegisterEmbeddedProvider(provider)

	// Load scripts and start watcher
	err = registry.LoadScripts()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = registry.StartWatcher(ctx)
	require.NoError(t, err)

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// External script should be active
	script, err := registry.GetScript("test_module", "test_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 999", script.Content)
	assert.Equal(t, SourceExternal, script.Source)

	// Delete the external script file
	err = os.Remove(scriptPath)
	require.NoError(t, err)

	// Give watcher time to detect and process the deletion
	time.Sleep(200 * time.Millisecond)

	// Should fall back to embedded script
	script, err = registry.GetScript("test_module", "test_script")
	require.NoError(t, err)
	assert.Equal(t, "result := 42", script.Content)
	assert.Equal(t, SourceEmbedded, script.Source)

	// Stop watcher
	registry.StopWatcher()
}

func TestRegistry_ParseScriptPath(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name           string
		filePath       string
		expectedModule string
		expectedScript string
		expectError    bool
	}{
		{
			name:           "tengo script",
			filePath:       "scripts/wargame/damage_calculator.tengo",
			expectedModule: "wargame",
			expectedScript: "damage_calculator",
			expectError:    false,
		},
		{
			name:           "zygomys script",
			filePath:       "scripts/chat/processor.zygomys",
			expectedModule: "chat",
			expectedScript: "processor",
			expectError:    false,
		},
		{
			name:           "no extension",
			filePath:       "scripts/test/simple_script",
			expectedModule: "test",
			expectedScript: "simple_script",
			expectError:    false,
		},
		{
			name:        "invalid path",
			filePath:    "scripts/invalid",
			expectError: true,
		},
		{
			name:        "not in scripts directory",
			filePath:    "other/test/script.tengo",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			module, script, err := registry.parseScriptPath(tc.filePath)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedModule, module)
				assert.Equal(t, tc.expectedScript, script)
			}
		})
	}
}

func TestRegistry_IsScriptFile(t *testing.T) {
	registry := NewRegistry()

	testCases := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"tengo file", "script.tengo", true},
		{"zygomys file", "script.zygomys", true},
		{"no extension", "script", true},
		{"text file", "readme.txt", false},
		{"go file", "main.go", false},
		{"json file", "config.json", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := registry.isScriptFile(tc.filePath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRegistry_WatcherWithoutScriptsDirectory(t *testing.T) {
	// Create temporary directory without scripts folder
	tempDir, err := os.MkdirTemp("", "script_no_dir_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create registry and try to start watcher
	registry := NewRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should not error when scripts directory doesn't exist
	err = registry.StartWatcher(ctx)
	assert.NoError(t, err)
}
