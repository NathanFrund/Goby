package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/nfrund/goby/internal/middleware" // Your custom middleware
)

// RegisterRoutes sets up all the application routes.
func (s *Server) RegisterRoutes() {
	// Create instances of all application middleware.
	rateLimiter := middleware.RateLimiter()
	// The auth middleware needs the userStore, which is now a dependency of the server.
	authMiddleware := middleware.Auth(s.UserStore)

	// Instantiate handlers that have dependencies directly within the routing setup.
	// This co-locates handler creation with its routes and keeps the Server struct clean.
	authHandler := handlers.NewAuthHandler(s.UserStore, s.Emailer, s.Cfg.GetAppBaseURL())

	// Public routes
	public := s.E.Group("")
	public.GET("/", handlers.HomeGet)
	public.GET("/about", handlers.AboutGet)
	public.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Auth routes
	auth := s.E.Group("/auth")
	// Redirect both /auth and /auth/ to the login page for convenience.
	redirectLogin := func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/auth/login")
	}
	auth.GET("", redirectLogin)
	auth.GET("/", redirectLogin)

	auth.GET("/register", authHandler.RegisterGetHandler)
	auth.POST("/register", authHandler.RegisterPost, rateLimiter)
	auth.GET("/login", authHandler.LoginGetHandler)
	auth.POST("/login", authHandler.LoginPost, rateLimiter)
	auth.GET("/logout", authHandler.Logout)
	auth.GET("/forgot-password", authHandler.ForgotPasswordGetHandler)
	auth.POST("/forgot-password", authHandler.ForgotPasswordPost, rateLimiter)
	auth.GET("/reset-password", authHandler.ResetPasswordGetHandler)
	auth.POST("/reset-password", authHandler.ResetPasswordPostHandler)

	// Protected routes (require authentication)
	protected := s.E.Group("/app")
	protected.Use(authMiddleware)

	// Standard routes
	protected.GET("/dashboard", s.DashboardHandler.Get)
	protected.GET("/ws/html", s.HTMLBridge.Handler())
	protected.GET("/ws/data", s.DataBridge.Handler())

	// Core File Service Routes are registered under the /app group
	// and are therefore protected by the authentication middleware.
	// The FileHandler is constructed in main.go and passed to the server.
	filesGroup := protected.Group("/files") // e.g., /app/files
	filesGroup.GET("", s.FileHandler.ListFiles)
	filesGroup.POST("/upload", s.FileHandler.UploadFile)
	filesGroup.DELETE("/:id", s.FileHandler.DeleteFile)
	filesGroup.GET("/:id/download", s.FileHandler.DownloadFile)
}
