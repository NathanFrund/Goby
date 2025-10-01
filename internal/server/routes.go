package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/middleware" // Your custom middleware
	"github.com/nfrund/goby/internal/registry"
)

// RegisterRoutes sets up all the application routes.
func (s *Server) RegisterRoutes() {
	// Create instances of all application middleware.
	rateLimiter := middleware.RateLimiter()
	// The auth middleware needs the userStore, which is now a dependency of the server.
	authMiddleware := middleware.Auth(s.UserStore)

	// Public routes
	public := s.E.Group("")
	public.GET("/", s.homeHandler.HomeGet)
	public.GET("/about", s.aboutHandler.HandleGet)
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

	auth.GET("/register", s.authHandler.RegisterGetHandler)
	auth.POST("/register", s.authHandler.RegisterPost, rateLimiter)
	auth.GET("/login", s.authHandler.LoginGetHandler)
	auth.POST("/login", s.authHandler.LoginPost, rateLimiter)
	auth.GET("/logout", s.authHandler.Logout)
	auth.GET("/forgot-password", s.authHandler.ForgotPasswordGetHandler)
	auth.POST("/forgot-password", s.authHandler.ForgotPasswordPost, rateLimiter)
	auth.GET("/reset-password", s.authHandler.ResetPasswordGetHandler)
	auth.POST("/reset-password", s.authHandler.ResetPasswordPostHandler)

	// Protected routes (require authentication)
	protected := s.E.Group("/app")
	protected.Use(authMiddleware)

	// --- Module Loading System ---

	// 1. Create a new service locator instance for this request scope.
	sl := registry.NewServiceLocator()

	// 2. Register core framework services that modules might need.
	sl.Set(string(registry.DBConnectionKey), s.DB) // Use direct SurrealDB client
	sl.Set(string(registry.HTMLHubKey), s.htmlHub)
	sl.Set(string(registry.DataHubKey), s.dataHub)
	sl.Set(string(registry.TemplateRendererKey), s.E.Renderer)
	sl.Set(string(registry.AppConfigKey), s.Cfg)
	sl.Set(string(registry.UserStoreKey), s.UserStore)

	// 3. Register services from all active modules.
	// This allows modules to add their services to the container.
	for _, mod := range AppModules {
		if err := mod.Register(sl, s.Cfg); err != nil {
			slog.Error("Failed to register module", "module", mod.Name(), "error", err)
		}
	}

	// 4. Boot all active modules.
	// Now that all services are registered, modules can safely resolve
	// dependencies and set up their routes.
	for _, mod := range AppModules {
		if err := mod.Boot(protected, sl); err != nil {
			slog.Error("Failed to boot module", "module", mod.Name(), "error", err)
		}
	}

	// Standard routes
	protected.GET("/dashboard", s.dashboardHandler.DashboardGet)
	protected.GET("/ws/data", s.dataHandler.ServeWS)
}
