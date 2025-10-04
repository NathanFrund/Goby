package view

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"maragu.dev/gomponents"
)

// --- GOMPONENTS -> TEMPL ADAPTER ---

// GomponentToTemplAdapter wraps a gomponents.Node to satisfy the templ.Component interface.
// This allows Gomponents content to be seamlessly rendered inside Templ layouts.
type GomponentToTemplAdapter struct {
	Node gomponents.Node
}

// Render implements the templ.Component interface by delegating the writing to the
// underlying gomponents.Node, allowing it to be used by templ's rendering pipeline.
func (a *GomponentToTemplAdapter) Render(ctx context.Context, w io.Writer) error {
	return a.Node.Render(w)
}

// AdaptGomponentToTempl is a helper function to create a GomponentToTemplAdapter.
// It converts a Gomponents Node (or any gomponents.Node) into a templ.Component.
func AdaptGomponentToTempl(node gomponents.Node) templ.Component {
	return &GomponentToTemplAdapter{Node: node}
}

// --- TEMPL -> GOMPONENTS ADAPTER ---

// TemplToGomponentAdapter wraps a templ.Component to satisfy the gomponents.Node interface.
// This allows a Templ component to be rendered inside a pure Gomponents view.
type TemplToGomponentAdapter struct {
	Component templ.Component
}

// Render implements the gomponents.Node interface by delegating the rendering to the
// underlying templ.Component.
// Note: Since Gomponents' Render method doesn't pass context, we use context.Background()
// for the Templ component call, which is a common pattern for this type of bridge.
func (a *TemplToGomponentAdapter) Render(w io.Writer) error {
	return a.Component.Render(context.Background(), w)
}

// AdaptTemplToGomponent is a helper function to create a TemplToGomponentAdapter.
// It converts a Templ Component into a Gomponents Node.
func AdaptTemplToGomponent(component templ.Component) gomponents.Node {
	return &TemplToGomponentAdapter{Component: component}
}
