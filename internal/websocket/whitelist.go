package websocket

import "slices"

// clientWhitelist contains the set of actions that clients are allowed to publish
type clientWhitelist struct {
	allowedActions []string
}

// NewClientWhitelist creates a new whitelist with the given allowed actions
func NewClientWhitelist(allowedActions ...string) *clientWhitelist {
	return &clientWhitelist{
		allowedActions: slices.Clone(allowedActions),
	}
}

// IsAllowed checks if an action is in the whitelist
func (w *clientWhitelist) IsAllowed(action string) bool {
	return slices.Contains(w.allowedActions, action)
}

// DefaultClientWhitelist returns a whitelist with common client actions
func DefaultClientWhitelist() *clientWhitelist {
	return NewClientWhitelist(
		// Add default allowed actions here
		// Example: "chat.message", "game.move", "device.status_update"
	)
}
