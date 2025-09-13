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
	dashboardHandler := handlers.NewDashboardHandler()

	// Create instances of all application middleware.
	rateLimiter := middleware.RateLimiter()
	authMiddleware := middleware.Auth(userStore)

	// Public routes
	public := s.E.Group("")
	public.GET("/", homeHandler.HomeGet)
	public.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Auth routes
	auth := s.E.Group("/auth")
	auth.GET("/register", authHandler.RegisterGet)
	auth.POST("/register", authHandler.RegisterPost, rateLimiter)
	auth.GET("/login", authHandler.LoginGet)
	auth.POST("/login", authHandler.LoginPost, rateLimiter)
	auth.GET("/logout", authHandler.Logout)
	auth.GET("/forgot-password", authHandler.ForgotPasswordGet)
	auth.POST("/forgot-password", authHandler.ForgotPasswordPost, rateLimiter)
	auth.GET("/reset-password", authHandler.ResetPasswordGet)
	auth.POST("/reset-password", authHandler.ResetPasswordPost)

	// Protected routes (require authentication)
	protected := s.E.Group("/app")
	protected.Use(authMiddleware)
	protected.GET("/dashboard", dashboardHandler.DashboardGet)
}
