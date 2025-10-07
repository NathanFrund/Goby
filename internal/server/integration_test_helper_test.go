package server_test

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/pubsub"
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

	cfg := config.New()
	db, err := database.NewDB(context.Background(), cfg)
	require.NoError(t, err)

	emailer, err := email.NewEmailService(cfg)
	require.NoError(t, err)

	// Correctly initialize and inject all required services, exactly as in main.go.
	ps := pubsub.NewWatermillBridge()
	renderer := rendering.NewUniversalRenderer()
	bridge := websocket.NewBridge(ps)

	s, err := server.New(
		server.WithConfig(cfg),
		server.WithDB(db, cfg.GetDBNs(), cfg.GetDBDb()),
		server.WithEmailer(emailer),
		server.WithRenderer(renderer),
		server.WithPubSub(ps),
		server.WithWebsocketBridge(bridge),
	)
	require.NoError(t, err)
	testServer := httptest.NewServer(s.E)

	cleanup := func() {
		_ = ps.Close() // Cleanly shut down the pub/sub system.
		testServer.Close()
		db.Close(context.Background())
		_ = os.Chdir(originalWD)
	}

	return s, testServer, cleanup
}
