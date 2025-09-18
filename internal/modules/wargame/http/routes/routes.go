package routes

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/registry"
)

func init() {
	registry.Register(func(g *echo.Group, sl registry.ServiceLocator) {
		// Retrieve the wargame engine from the service locator.
		engine, ok := sl.Get(string(registry.WargameEngineKey)).(*wargame.Engine)
		if !ok || engine == nil {
			slog.Warn("wargame.engine not found in service locator, skipping route registration")
			return
		}

		// Call the original route registration function with the retrieved dependency.
		RegisterRoutes(g, engine)
	})
}

// RegisterRoutes wires the wargame HTTP endpoints under the provided group.
// Example mount: protected group -> /app
func RegisterRoutes(g *echo.Group, engine *wargame.Engine) {
	// Debug route to trigger a wargame event
	g.GET("/debug/hit", func(c echo.Context) error {
		go engine.SimulateHit()
		return c.String(http.StatusOK, "Wargame hit event triggered.")
	})
}
