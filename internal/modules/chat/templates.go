package chat

import (
	"embed"
	"io"
	"log/slog"

	"github.com/nfrund/goby/internal/templates"
)

func init() {
	// Self-register the template registration function.
	templates.Register(RegisterTemplates)
}

//go:embed welcome-message.html
var templatesFS embed.FS

// RegisterTemplates registers the chat module's embedded templates with the renderer.
func RegisterTemplates(r *templates.Renderer) {
	_ = r.AddStandaloneFromFS(templatesFS, ".", "") // Register at root level

	// Sanity-check that the critical template was registered correctly.
	if err := r.Render(io.Discard, "welcome-message.html", nil, nil); err != nil {
		slog.Error("Chat template sanity check failed", "name", "welcome-message.html", "error", err)
	}
}
