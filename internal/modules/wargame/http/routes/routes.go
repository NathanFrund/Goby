package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/modules/wargame"
)

// RegisterRoutes wires the wargame HTTP endpoints under the provided group.
// Example mount: protected group -> /app
func RegisterRoutes(g *echo.Group, engine *wargame.Engine) {
	// Debug route to trigger a wargame event
	g.GET("/debug/hit", func(c echo.Context) error {
		go engine.SimulateHit()
		return c.String(http.StatusOK, "Wargame hit event triggered.")
	})
}
