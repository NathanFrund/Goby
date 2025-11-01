package server_test

import (
	"context"
	"encoding/gob"
	"log"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	appmiddleware "github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/server"
	"github.com/nfrund/goby/internal/topicmgr"
	wsTopics "github.com/nfrund/goby/internal/websocket"
	"github.com/stretchr/testify/require"
)

// Define registry keys for services needed in integration tests.
var (
	KeyTestConfigProvider = registry.Key[config.Provider]("test.config.Provider")
	KeyTestUserRepository = registry.Key[domain.UserRepository]("test.domain.UserRepository")
)

// TestMain runs once for the entire package before any tests are run.
// It's the perfect place to load test-specific environment variables.
func TestMain(m *testing.M) {
	if err := godotenv.Overload("../../.env.test"); err != nil {
		log.Fatalf("Error loading .env.test file for integration tests: %v", err)
	}
	os.Exit(m.Run())
}

// setupIntegrationTest encapsulates the boilerplate for setting up a full server
// instance for integration testing. It returns the server instance, the test
// server itself, and a cleanup function to be deferred.
func setupIntegrationTest(t *testing.T) (*server.Server, *httptest.Server, func()) {
	t.Helper()

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir("../../")
	require.NoError(t, err)

	// 1. Create config and registry, just like in main.go
	cfg := config.New()
	reg := registry.New(cfg)

	// 2. Initialize and register all core services for the test environment.
	dbConn := database.NewConnection(cfg)
	err = dbConn.Connect(context.Background())
	require.NoError(t, err)
	dbConn.StartMonitoring()

	userDBClient, err := database.NewClient[domain.User](dbConn)
	require.NoError(t, err)

	userStore := database.NewUserStore(userDBClient, dbConn)

	registry.Set(reg, KeyTestConfigProvider, cfg)
	registry.Set(reg, KeyTestUserRepository, userStore)

	emailer, err := email.NewEmailService(cfg)
	require.NoError(t, err)

	ps := pubsub.NewWatermillBridge()

	// Create an isolated topic manager for testing to avoid conflicts between tests
	topicManager := topicmgr.NewManager()

	// Register WebSocket framework topics with the isolated manager
	err = wsTopics.RegisterTopicsWithManager(topicManager)
	require.NoError(t, err, "Failed to register WebSocket topics")

	// Create bridges with dependencies
	htmlBridge := wsTopics.NewBridge("html", wsTopics.BridgeDependencies{ // Use wsTopics alias
		Publisher:    ps,
		Subscriber:   ps,
		TopicManager: topicManager,
		ReadyTopic:   wsTopics.TopicClientReady,
	})

	dataBridge := wsTopics.NewBridge("data", wsTopics.BridgeDependencies{ // Use wsTopics alias
		Publisher:    ps,
		Subscriber:   ps,
		TopicManager: topicManager,
		ReadyTopic:   wsTopics.TopicClientReady,
	})

	renderer := rendering.NewUniversalRenderer()

	e := echo.New()

	// 3. Create the server instance using the populated registry.
	s, err := server.New(server.Dependencies{
		Config:     cfg,
		Emailer:    emailer,
		UserStore:  userStore,
		Renderer:   renderer,
		Publisher:  ps,
		Echo:       e,
		HTMLBridge: htmlBridge,
		DataBridge: dataBridge,
	})
	require.NoError(t, err)

	// 4. Initialize modules and register all routes, just like in main.go
	moduleDeps := app.Dependencies{
		Publisher:  ps,
		Subscriber: ps,
		Renderer:   renderer,
		TopicMgr:   topicManager,
	}
	modules := app.NewModules(moduleDeps)
	s.InitModules(context.Background(), modules, reg)

	// Initialize test emailer
	emailer, err = email.NewEmailService(cfg)
	if err != nil {
		t.Fatalf("Failed to create email service: %v", err)
	}

	// Set required server fields used by RegisterRoutes
	s.Emailer = emailer
	s.UserStore = userStore
	s.Cfg = cfg

	// Register all routes
	s.RegisterRoutes()

	// Register module routes
	for _, m := range modules {
		if routeRegistrar, ok := m.(interface{ RegisterRoutes(e *echo.Echo) }); ok {
			routeRegistrar.RegisterRoutes(s.E)
		}
	}

	// In our tests, the websocket handlers require an authenticated user.
	// Instead of performing a full login flow for every test, we can inject a
	// middleware that simulates an authenticated user when a specific "fake"
	// session cookie is present. This simplifies tests that need an authenticated
	// context but aren't testing the auth flow itself.
	s.E.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("session")
			if err == nil && cookie.Value == "fake-session-for-testing" {
				// Create a dummy user and add it to the context.
				// The user doesn't need to exist in the DB for many tests.
				dummyUser := &domain.User{Email: "test-user@example.com"}
				c.Set(appmiddleware.UserContextKey, dummyUser)
			}
			return next(c)
		}
	})
	gob.Register(&domain.User{})

	// Register core, non-module routes for the test server.
	// This is done here to avoid modifying the main server.go for test purposes.
	// These routes are for core functionalities like WebSockets.
	// We do NOT use the real appmiddleware.Auth here because our test middleware
	// already simulates an authenticated user, which is sufficient for these tests.
	wsGroup := s.E.Group("/ws")
	// wsGroup.Use(appmiddleware.Auth(s.UserStore)) // This is handled by the test-specific middleware above
	wsGroup.GET("/html", s.HTMLBridge.Handler())
	wsGroup.GET("/data", s.DataBridge.Handler())

	// Start the WebSocket bridges so they subscribe to pub/sub topics.
	// This is crucial for the tests to receive messages.
	// If either bridge fails to start, we need to fail the test immediately
	// to avoid mysterious test failures later.
	if err := s.HTMLBridge.Start(context.Background()); err != nil {
		t.Fatalf("failed to start HTML WebSocket bridge: %v", err)
	}

	// Only proceed with starting the data bridge if HTML bridge started successfully
	if err := s.DataBridge.Start(context.Background()); err != nil {
		// Ensure we clean up the HTML bridge before failing
		s.HTMLBridge.Shutdown(context.Background())
		t.Fatalf("failed to start Data WebSocket bridge: %v", err)
	}

	testServer := httptest.NewServer(s.E)

	cleanup := func() {
		// Create a context for graceful shutdown of background services.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 1. Stop the WebSocket bridges first. This stops their subscribers.
		s.HTMLBridge.Shutdown(shutdownCtx)
		s.DataBridge.Shutdown(shutdownCtx)

		// 2. Close the pub/sub system.
		_ = ps.Close()

		// 3. Reset the isolated topic manager to prevent state leakage between tests.
		topicManager.Reset()

		// 4. Close the test server and database connection.
		testServer.Close()
		dbConn.Close(shutdownCtx)
		_ = os.Chdir(originalWD) // Restore original working directory.
	}

	return s, testServer, cleanup
}

// TestSetupIntegrationTest verifies that the entire server setup and teardown
// process can complete without errors.
func TestSetupIntegrationTest(t *testing.T) {
	t.Run("should setup and teardown without errors", func(t *testing.T) {
		_, _, cleanup := setupIntegrationTest(t)
		defer cleanup()
	})
}
