package presence

import "github.com/a-h/templ"

// PresenceRenderer is a function that renders presence data as HTML.
// Modules can provide their own implementations to customize the UI.
// The function receives a list of online user identifiers and returns
// a templ.Component that will be rendered by the framework.
type PresenceRenderer func(users []string) templ.Component

// DefaultRenderer provides a simple, unstyled presence list.
// This is the framework's default renderer that will be used
// unless a module injects a custom renderer.
func DefaultRenderer(users []string) templ.Component {
	return defaultPresenceList(users)
}
