package server_test

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/chat"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/server"
	"github.com/nfrund/goby/internal/websocket"
	"github.com/stretchr/testify/require"
)

// testModules defines the list of modules to be loaded for integration tests.
var testModules = []module.Module{
	wargame.New(),
	chat.New(),
}

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
	surrealDB, err := database.NewDB(context.Background(), cfg)
	require.NoError(t, err)

	dbClient := database.NewClient(surrealDB)
	reg.Set((*database.Client)(nil), dbClient)
	reg.Set((*config.Provider)(nil), cfg)

	userStore := database.NewSurrealUserStore(surrealDB, cfg.GetDBNs(), cfg.GetDBDb())
	reg.Set((*domain.UserRepository)(nil), userStore)

	emailer, err := email.NewEmailService(cfg)
	require.NoError(t, err)
	reg.Set((*domain.EmailSender)(nil), emailer)

	ps := pubsub.NewWatermillBridge()
	reg.Set((*pubsub.Publisher)(nil), ps)
	reg.Set((*pubsub.Subscriber)(nil), ps)

	wsBridge := websocket.NewBridge(ps)
	reg.Set((*websocket.Bridge)(nil), wsBridge)

	renderer := rendering.NewUniversalRenderer()
	reg.Set((*rendering.Renderer)(nil), renderer)
	reg.Set((*echo.Renderer)(nil), renderer)

	// 3. Create the server instance using the populated registry.
	s, err := server.New(reg)
	require.NoError(t, err)

	// 4. Initialize modules and register all routes, just like in main.go
	s.InitModules(testModules, reg)
	s.RegisterRoutes()

	testServer := httptest.NewServer(s.E)

	cleanup := func() {
		_ = ps.Close() // Cleanly shut down the pub/sub system.
		testServer.Close()
		surrealDB.Close(context.Background()) // Close the database connection.
		_ = os.Chdir(originalWD)
	}

	return s, testServer, cleanup
}
