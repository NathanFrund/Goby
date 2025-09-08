package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/handlers"
)

// RegisterRoutes sets up all the application routes.
func (s *Server) RegisterRoutes() {
	// Create instances of all application handlers.
	homeHandler := handlers.NewHomeHandler()
	authHandler := handlers.NewAuthHandler(s.DB, s.Cfg.DBNs, s.Cfg.DBDb)

	// Register routes.
	s.E.GET("/", homeHandler.HomeGet)

	s.E.GET("/register", authHandler.RegisterGet)
	s.E.POST("/register", authHandler.RegisterPost)

	s.E.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
}
