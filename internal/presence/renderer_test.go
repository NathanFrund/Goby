package presence_test

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/nfrund/goby/internal/presence"
)

func TestDefaultRenderer(t *testing.T) {
	tests := []struct {
		name        string
		users       []string
		contains    []string
		notContains []string
	}{
		{
			name:        "empty user list",
			users:       []string{},
			contains:    []string{"No users online", "Online Users (0)"},
			notContains: []string{"<li>"},
		},
		{
			name:        "single user",
			users:       []string{"alice@example.com"},
			contains:    []string{"alice@example.com", "Online Users (1)", "<li>"},
			notContains: []string{"No users online"},
		},
		{
			name:        "multiple users",
			users:       []string{"alice@example.com", "bob@example.com"},
			contains:    []string{"alice@example.com", "bob@example.com", "Online Users (2)"},
			notContains: []string{"No users online"},
		},
		{
			name:        "user with special characters",
			users:       []string{"user+test@example.com"},
			contains:    []string{"user+test@example.com"},
			notContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := presence.DefaultRenderer(tt.users)

			// Render to string
			var buf strings.Builder
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			html := buf.String()

			// Check that expected strings are present
			for _, expected := range tt.contains {
				if !strings.Contains(html, expected) {
					t.Errorf("expected HTML to contain %q, got:\n%s", expected, html)
				}
			}

			// Check that unexpected strings are not present
			for _, unexpected := range tt.notContains {
				if strings.Contains(html, unexpected) {
					t.Errorf("expected HTML to NOT contain %q, got:\n%s", unexpected, html)
				}
			}
		})
	}
}

func TestDefaultRenderer_RendersValidHTML(t *testing.T) {
	users := []string{"test1@example.com", "test2@example.com"}
	component := presence.DefaultRenderer(users)

	var buf strings.Builder
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	// Verify basic HTML structure
	requiredElements := []string{
		`<div id="online-users"`,
		`<h3>`,
		`</h3>`,
		`<ul>`,
		`</ul>`,
		`<li>`,
		`</li>`,
		`</div>`,
	}

	for _, element := range requiredElements {
		if !strings.Contains(html, element) {
			t.Errorf("expected HTML to contain element %q", element)
		}
	}
}

func TestCustomRenderer(t *testing.T) {
	// Create a custom renderer that returns a simple component
	customRenderer := func(users []string) templ.Component {
		// Use a simple raw component for testing
		return templ.Raw(
			`<div class="custom-presence">` +
				strings.Join(users, ", ") +
				`</div>`,
		)
	}

	users := []string{"test@example.com"}
	component := customRenderer(users)

	var buf strings.Builder
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "test@example.com") {
		t.Errorf("custom renderer should render user email, got: %s", html)
	}
	if !strings.Contains(html, "custom-presence") {
		t.Errorf("custom renderer should use custom class, got: %s", html)
	}
}

func TestPresenceRenderer_Interface(t *testing.T) {
	// Verify that PresenceRenderer is a function type that can be assigned
	var renderer presence.PresenceRenderer

	renderer = presence.DefaultRenderer
	if renderer == nil {
		t.Error("DefaultRenderer should be assignable to PresenceRenderer")
	}

	// Verify it can be called
	users := []string{"test@example.com"}
	component := renderer(users)
	if component == nil {
		t.Error("renderer should return a non-nil component")
	}
}
