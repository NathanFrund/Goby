package websocket

import (
	"errors"
	"log/slog"
	"slices"
	"sync"
)

var (
	// ErrActionAlreadyExists is returned when trying to add a duplicate action
	ErrActionAlreadyExists = errors.New("action already exists in whitelist")
	// ErrInvalidAction is returned when an empty action is provided
	ErrInvalidAction = errors.New("action cannot be empty")
)

// clientWhitelist contains the set of actions that clients are allowed to publish
type clientWhitelist struct {
	mu             sync.RWMutex
	allowedActions []string
}

// NewClientWhitelist creates a new whitelist with the given allowed actions
func NewClientWhitelist(allowedActions ...string) *clientWhitelist {
	// Filter out any empty actions
	validActions := make([]string, 0, len(allowedActions))
	for _, action := range allowedActions {
		if action != "" {
			validActions = append(validActions, action)
		}
	}

	return &clientWhitelist{
		allowedActions: validActions,
	}
}

// IsAllowed checks if an action is in the whitelist in a thread-safe manner
func (w *clientWhitelist) IsAllowed(action string) bool {
	if action == "" {
		return false
	}

	w.mu.RLock()
	defer w.mu.RUnlock()
	
	return slices.Contains(w.allowedActions, action)
}

// AddAction adds an action to the whitelist in a thread-safe manner
// Returns an error if the action is empty or already exists
func (w *clientWhitelist) AddAction(action string) error {
	if action == "" {
		slog.Warn("attempted to add empty action to whitelist")
		return ErrInvalidAction
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if slices.Contains(w.allowedActions, action) {
		slog.Debug("action already in whitelist", "action", action)
		return ErrActionAlreadyExists
	}

	w.allowedActions = append(w.allowedActions, action)
	slog.Info("added action to whitelist", "action", action)
	return nil
}

// DefaultClientWhitelist returns a whitelist with common client actions
func DefaultClientWhitelist() *clientWhitelist {
	return NewClientWhitelist(
		// Add default allowed actions here
		// Example: "chat.message", "game.move", "device.status_update"
	)
}
