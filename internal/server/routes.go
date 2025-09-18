package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/registry"

	// Load generated module route imports (zz_routes_imports.go)
	_ "github.com/nfrund/goby/internal/modules"
)

// deps is a simple map-based implementation of the registry.ServiceLocator interface.
type deps map[string]any

// Get retrieves a dependency by its key.
func (d deps) Get(k string) any { return d[k] }

// RegisterRoutes sets up all the application routes.
func (s *Server) RegisterRoutes() {
	// Create instances of all application middleware.
	rateLimiter := middleware.RateLimiter()
	// The auth middleware needs the userStore, which is now a dependency of the server.
	authMiddleware := middleware.Auth(s.UserStore)

	// Public routes
	public := s.E.Group("")
	public.GET("/", s.homeHandler.HomeGet)
	public.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Auth routes
	auth := s.E.Group("/auth")
	auth.GET("/register", s.authHandler.RegisterGet)
	auth.POST("/register", s.authHandler.RegisterPost, rateLimiter)
	auth.GET("/login", s.authHandler.LoginGet)
	auth.POST("/login", s.authHandler.LoginPost, rateLimiter)
	auth.GET("/logout", s.authHandler.Logout)
	auth.GET("/forgot-password", s.authHandler.ForgotPasswordGet)
	auth.POST("/forgot-password", s.authHandler.ForgotPasswordPost, rateLimiter)
	auth.GET("/reset-password", s.authHandler.ResetPasswordGet)
	auth.POST("/reset-password", s.authHandler.ResetPasswordPost)

	// Protected routes (require authentication)
	protected := s.E.Group("/app")
	protected.Use(authMiddleware)
	protected.GET("/dashboard", s.dashboardHandler.DashboardGet)
	protected.GET("/chat", s.chatHandler.ChatGet)

	// WebSocket endpoint for broadcasting HTML fragments to htmx clients.
	protected.GET("/ws/html", s.chatHandler.ServeWS)

	// WebSocket endpoint for broadcasting raw data (JSON) to other clients.
	protected.GET("/ws/data", s.dataHandler.ServeWS)

	// --- Auto-apply all registered module routes ---
	registry.Apply(protected, deps{
		"wargame.engine": s.WargameEngine,
	})
}
