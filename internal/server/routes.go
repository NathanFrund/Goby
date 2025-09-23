package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/full-chat"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
)

// deps is a simple map-based implementation of the registry.ServiceLocator interface.
type deps struct {
	services map[string]any
}

// Get retrieves a dependency by its key.
func (d *deps) Get(k string) any { 
	return d.services[k] 
}

// newDeps creates a new deps instance with the provided services.
func newDeps(services map[string]any) *deps {
	return &deps{
		services: services,
	}
}

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

	// Register modules first to initialize all dependencies
	templatesRenderer, ok := s.E.Renderer.(*templates.Renderer)
	if !ok {
		slog.Error("Failed to get templates renderer")
		return
	}

	// Initialize all modules and get their dependencies
	moduleDependencies := registerModules(
		s.htmlHub,
		s.dataHub,
		templatesRenderer,
		s.DB,
		s.Cfg,
	)

	// Set the wargame engine on the server if it was initialized
	if wargameEngine, ok := moduleDependencies[string(registry.WargameEngineKey)].(*wargame.Engine); ok {
		s.wargameEngine = wargameEngine
	}

	// Apply registered module routes with all dependencies
	deps := newDeps(moduleDependencies)
	registry.Apply(protected, deps)

	// Standard routes
	protected.GET("/dashboard", s.dashboardHandler.DashboardGet)
	protected.GET("/chat", s.chatHandler.ChatGet)
	protected.GET("/ws/html", s.chatHandler.ServeWS)
	protected.GET("/ws/data", s.dataHandler.ServeWS)

	// Register full-chat routes if available
	if svc, ok := moduleDependencies[string(registry.FullChatService)]; ok && svc != nil {
		if fullChatSvc, ok := svc.(fullchat.Service); ok {
			handler := fullchat.NewMessageHandler(fullChatSvc)
			fullChatGroup := protected.Group("/full-chat")
			fullChatGroup.GET("", handler.ChatUI)
			fullChatGroup.GET("/ws", handler.WebSocketHandler)
			
			// API v1 routes
			v1 := fullChatGroup.Group("/api/v1")
			{
				messages := v1.Group("/messages")
				messages.POST("", handler.CreateMessage)
				messages.GET("", handler.ListMessages)
			}
		}
	}
}

