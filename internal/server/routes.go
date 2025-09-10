package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/nfrund/goby/internal/middleware"
)

// RegisterRoutes sets up all the application routes.
func (s *Server) RegisterRoutes() {
	// Create instances of all application handlers.
	homeHandler := handlers.NewHomeHandler()
	userStore := database.NewUserStore(s.DB, s.Cfg.DBNs, s.Cfg.DBDb)
	authHandler := handlers.NewAuthHandler(userStore, s.Emailer, s.Cfg.AppBaseURL)
	rateLimiter := middleware.RateLimiter()

	// Register routes.
	s.E.GET("/", homeHandler.HomeGet)

	s.E.GET("/register", authHandler.RegisterGet)
	s.E.POST("/register", authHandler.RegisterPost, rateLimiter)

	s.E.GET("/login", authHandler.LoginGet)
	s.E.POST("/login", authHandler.LoginPost, rateLimiter)
	s.E.GET("/logout", authHandler.Logout)

	s.E.GET("/forgot-password", authHandler.ForgotPasswordGet)
	s.E.POST("/forgot-password", authHandler.ForgotPasswordPost, rateLimiter)

	s.E.GET("/reset-password", authHandler.ResetPasswordGet)
	s.E.POST("/reset-password", authHandler.ResetPasswordPost)

	s.E.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
}
