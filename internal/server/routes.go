package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/middleware" // Your custom middleware
	"github.com/nfrund/goby/internal/websocket"
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
	// This logic has been moved to server.InitModules to be more explicit.
	// reg := registry.New(s.Cfg)

	// // 1. Register core framework services that modules might need.
	// reg.Set((*domain.UserRepository)(nil), s.UserStore)
	// reg.Set((*domain.EmailSender)(nil), s.Emailer)
	// reg.Set((*domain.Renderer)(nil), s.Renderer)
	// reg.Set((*pubsub.Publisher)(nil), s.PubSub)
	// reg.Set((*pubsub.Subscriber)(nil), s.PubSub)
	// reg.Set((*websocket.Bridge)(nil), s.bridge)

	// // 2. Register services from all active modules.
	// for _, mod := range s.modules {
	// 	if err := mod.Register(reg); err != nil {
	// 		slog.Error("Failed to register module", "module", mod.Name(), "error", err)
	// 	}
	// }

	// // 3. Boot all active modules, setting up their routes.
	// for _, mod := range s.modules {
	// 	if err := mod.Boot(protected, reg); err != nil {
	// 		slog.Error("Failed to boot module", "module", mod.Name(), "error", err)
	// 	}
	// }

	// Standard routes
	protected.GET("/dashboard", s.dashboardHandler.DashboardGet)

	// Register WebSocket endpoints.
	protected.GET("/ws/html", s.bridge.Handler(websocket.ConnectionTypeHTML))
	protected.GET("/ws/data", s.bridge.Handler(websocket.ConnectionTypeData))
}
