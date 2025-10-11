package server_test

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/server"
	"github.com/nfrund/goby/internal/websocket"
	"github.com/stretchr/testify/require"
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

	userDBClient, err := database.NewClient[domain.User](dbConn, cfg)
	require.NoError(t, err)

	userStore := database.NewUserStore(userDBClient, cfg)

	reg.Set((*config.Provider)(nil), cfg)
	reg.Set((*domain.UserRepository)(nil), userStore)

	emailer, err := email.NewEmailService(cfg)
	require.NoError(t, err)

	ps := pubsub.NewWatermillBridge()

	wsBridge := websocket.NewBridge(ps)

	renderer := rendering.NewUniversalRenderer()

	e := echo.New()

	// 3. Create the server instance using the populated registry.
	s, err := server.New(server.Dependencies{
		Config:    cfg,
		Emailer:   emailer,
		UserStore: userStore,
		Renderer:  renderer,
		Publisher: ps,
		Echo:      e,
		Bridge:    wsBridge,
	})
	require.NoError(t, err)

	// 4. Initialize modules and register all routes, just like in main.go
	moduleDeps := app.Dependencies{Publisher: ps, Subscriber: ps, Bridge: wsBridge, Renderer: renderer}
	modules := app.NewModules(moduleDeps)
	s.InitModules(context.Background(), modules, reg)

	s.RegisterRoutes()

	testServer := httptest.NewServer(s.E)

	cleanup := func() {
		_ = ps.Close() // Cleanly shut down the pub/sub system.
		testServer.Close()
		dbConn.Close(context.Background()) // Close the database connection.
		_ = os.Chdir(originalWD)
	}

	return s, testServer, cleanup
}
