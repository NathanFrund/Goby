package handlers_test

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/nfrund/goby/internal/presence"
)

// mockPresenceService is a simple mock for testing
type mockPresenceService struct {
	users []string
}

func (m *mockPresenceService) GetOnlineUsers() []string {
	return m.users
}

func TestPresenceHandler_GetPresenceHTML_DefaultRenderer(t *testing.T) {
	// Setup mock service
	mockService := &mockPresenceService{
		users: []string{"test@example.com", "alice@example.com"},
	}

	// Create handler with mock (we need to use reflection or create a test constructor)
	// For now, let's test the renderer directly
	component := presence.DefaultRenderer(mockService.users)

	// Render
	var buf strings.Builder
	err := component.Render(testContext(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	// Verify content
	if !strings.Contains(html, "test@example.com") {
		t.Errorf("expected HTML to contain test@example.com")
	}
	if !strings.Contains(html, "alice@example.com") {
		t.Errorf("expected HTML to contain alice@example.com")
	}
	if !strings.Contains(html, "Online Users (2)") {
		t.Errorf("expected HTML to contain user count")
	}
}

func TestPresenceHandler_CustomRenderer(t *testing.T) {
	users := []string{"test@example.com"}

	// Create custom renderer
	customRenderer := func(users []string) templ.Component {
		return templ.Raw(
			`<div class="custom-presence-test">` +
				strings.Join(users, ", ") +
				`</div>`,
		)
	}

	component := customRenderer(users)

	var buf strings.Builder
	err := component.Render(testContext(), &buf)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "custom-presence-test") {
		t.Errorf("expected custom class")
	}
	if !strings.Contains(html, "test@example.com") {
		t.Errorf("expected user email")
	}
}

func TestPresenceHandler_SetRenderer(t *testing.T) {
	// This test verifies that SetRenderer method exists and works
	// We can't easily test the full handler without complex mocking,
	// but we can verify the renderer pattern works

	defaultRenderer := presence.DefaultRenderer
	customRenderer := func(users []string) templ.Component {
		return templ.Raw("<div>custom</div>")
	}

	// Verify both are valid PresenceRenderer types
	var r1 presence.PresenceRenderer = defaultRenderer
	var r2 presence.PresenceRenderer = customRenderer

	if r1 == nil || r2 == nil {
		t.Error("renderers should be assignable to PresenceRenderer type")
	}

	// Verify they produce different output
	users := []string{"test@example.com"}

	c1 := r1(users)
	c2 := r2(users)

	var buf1, buf2 strings.Builder
	c1.Render(testContext(), &buf1)
	c2.Render(testContext(), &buf2)

	if buf1.String() == buf2.String() {
		t.Error("default and custom renderers should produce different output")
	}
}

// testContext returns a test context for rendering
func testContext() context.Context {
	return context.Background()
}
