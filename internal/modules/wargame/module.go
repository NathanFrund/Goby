package wargame

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
)

// WargameModule implements the module.Module interface.
type WargameModule struct{}

// Name returns the unique name for the module.
func (m *WargameModule) Name() string {
	return "wargame"
}

// RegisterTemplates registers the module's embedded templates with the renderer.
func (m *WargameModule) RegisterTemplates(renderer *templates.Renderer) {
	// This logic is moved from engine.go's RegisterTemplates function.
	if err := renderer.AddStandaloneFromFS(templatesFS, "templates/components", m.Name()); err != nil {
		slog.Error("Failed to register wargame embedded components", "error", err)
	}
}

// Register creates the Wargame Engine and registers it in the service locator.
func (m *WargameModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
	// This logic is moved from internal/server/modules.go
	htmlHubVal, ok := sl.Get(string(registry.HTMLHubKey))
	if !ok {
		return fmt.Errorf("HTML hub not found in service locator for wargame module")
	}
	htmlHub := htmlHubVal.(*hub.Hub)

	dataHubVal, ok := sl.Get(string(registry.DataHubKey))
	if !ok {
		return fmt.Errorf("data hub not found in service locator for wargame module")
	}
	dataHub := dataHubVal.(*hub.Hub)

	rendererVal, ok := sl.Get(string(registry.TemplateRendererKey))
	if !ok {
		return fmt.Errorf("template renderer not found in service locator for wargame module")
	}
	renderer := rendererVal.(echo.Renderer)

	slog.Info("Initializing wargame engine")
	wargameEngine := NewEngine(htmlHub, dataHub, renderer)
	sl.Set(string(registry.WargameEngineKey), wargameEngine)

	return nil
}

// Boot registers the HTTP routes for the wargame module.
func (m *WargameModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	// This logic is moved from the old init-based route registration.
	wargameEngineVal, ok := sl.Get(string(registry.WargameEngineKey))
	if !ok {
		slog.Warn("Wargame engine not found in service locator, skipping route registration.")
		return nil
	}
	wargameEngine := wargameEngineVal.(*Engine)

	g.GET("/debug/hit", func(c echo.Context) error {
		go wargameEngine.SimulateHit()
		return c.String(http.StatusOK, "Hit event triggered.")
	})
	return nil
}
