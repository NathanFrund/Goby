package websocket

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientWhitelist_IsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		actions  []string
		action   string
		expected bool
	}{
		{
			name:     "empty whitelist",
			actions:  []string{},
			action:   "chat.message",
			expected: false,
		},
		{
			name:     "action exists",
			actions:  []string{"chat.message", "game.move"},
			action:   "chat.message",
			expected: true,
		},
		{
			name:     "action does not exist",
			actions:  []string{"chat.message", "game.move"},
			action:   "unknown.action",
			expected: false,
		},
		{
			name:     "empty action",
			actions:  []string{"chat.message"},
			action:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := NewClientWhitelist(tt.actions...)
			result := wl.IsAllowed(tt.action)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientWhitelist_AddAction(t *testing.T) {
	tests := []struct {
		name          string
		initial       []string
		action        string
		expectedError  error
		expectedCount  int
		expectedExists bool
	}{
		{
			name:          "add new action",
			initial:       []string{"chat.message"},
			action:        "game.move",
			expectedError:  nil,
			expectedCount:  2,
			expectedExists: true,
		},
		{
			name:          "duplicate action",
			initial:       []string{"chat.message"},
			action:        "chat.message",
			expectedError:  ErrActionAlreadyExists,
			expectedCount:  1,
			expectedExists: true,
		},
		{
			name:          "empty action",
			initial:       []string{"chat.message"},
			action:        "",
			expectedError:  ErrInvalidAction,
			expectedCount:  1,
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := NewClientWhitelist(tt.initial...)
			err := wl.AddAction(tt.action)
			
			assert.ErrorIs(t, err, tt.expectedError)
			assert.Equal(t, tt.expectedExists, wl.IsAllowed(tt.action))
			
			// Verify the count of actions
			if tt.expectedCount >= 0 {
				// This is a bit of a hack to check the internal slice length
				// In a real test, we might want to add a method to get the count
				wl.mu.RLock()
				assert.Len(t, wl.allowedActions, tt.expectedCount)
				wl.mu.RUnlock()
			}
		})
	}
}

func TestClientWhitelist_ConcurrentAccess(t *testing.T) {
	wl := NewClientWhitelist()
	const numGoroutines = 100
	var wg sync.WaitGroup

	// Test concurrent AddAction
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			action := string(rune('a' + (idx % 26))) // Generate a-z actions
			_ = wl.AddAction(action)
		}(i)
	}

	// Test concurrent IsAllowed
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			action := string(rune('a' + (idx % 26)))
			_ = wl.IsAllowed(action)
		}(i)
	}

	wg.Wait()

	// Verify all actions were added (26 unique letters a-z)
	for i := 0; i < 26; i++ {
	action := string(rune('a' + i))
	assert.True(t, wl.IsAllowed(action), "action %s should be in whitelist", action)
	}
}
