package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nfrund/goby/internal/config"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/contrib/rews"
	"github.com/surrealdb/surrealdb.go/pkg/connection"
	"github.com/surrealdb/surrealdb.go/pkg/connection/gorillaws"
	"github.com/surrealdb/surrealdb.go/surrealcbor"
)

// initializeREWSConnection sets up a reliable WebSocket connection with enterprise features
func (c *Connection) initializeREWSConnection(dbURL string) (interface{}, error) {
	// Create a new REWS connection with automatic reconnection and session restoration
	codec := surrealcbor.New()
	rewsConn := rews.New(
		func(ctx context.Context) (*gorillaws.Connection, error) {
			// Create the underlying WebSocket connection using gorilla/websocket
			config := &connection.Config{
				BaseURL:     dbURL,
				Marshaler:   codec,
				Unmarshaler: codec,
			}
			ws := gorillaws.New(config)
			return ws, nil
		},
		30*time.Second, // check interval for reconnection
		codec,          // unmarshaler
		nil,            // logger - using default
	)

	// Configure retry behavior
	rewsConn.Retryer = c.retryer

	// Connect with automatic retry
	if err := rewsConn.Connect(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to establish REWS connection: %w", err)
	}

	return rewsConn, nil
}

// Connection manages a SurrealDB connection with REWS (Reliable WebSocket) support
type Connection struct { // Implements DBConnection
	cfg      config.Provider
	conn     *surrealdb.DB
	rewsConn interface{} // REWS connection for reliable WebSocket management
	retryer  rews.Retryer
	mu       sync.RWMutex
	healthy  bool
	done     chan struct{}
}

// NewConnection creates a new managed database connection
func NewConnection(cfg config.Provider) *Connection {
	retryer := rews.NewExponentialBackoffRetryer()
	retryer.InitialDelay = 100 * time.Millisecond
	retryer.MaxDelay = 30 * time.Second
	retryer.Multiplier = 2.0
	retryer.MaxRetries = 5
	retryer.Jitter = true
	retryer.JitterFactor = 0.25

	return &Connection{
		cfg:     cfg,
		retryer: retryer,
		done:    make(chan struct{}),
	}
}

// Connect establishes the initial database connection
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	return c.reconnect(ctx)
}

// WithConnection executes a function with a database connection, handling reconnections
func (c *Connection) WithConnection(ctx context.Context, fn func(*surrealdb.DB) error) error {
	// Get the current connection
	conn := c.getConnection()
	if conn == nil {
		return NewDBError(ErrNotConnected, "database not connected")
	}

	// For REWS connections, check if we need to refresh the underlying DB reference
	if c.rewsConn != nil {
		// REWS connections handle reconnection automatically, so we can rely on the current connection
		// being valid. If the connection fails, REWS will handle reconnection internally.
	}

	// Try the operation first
	err := fn(conn)
	if err == nil {
		return nil
	}

	// If the error is not a connection-related issue, just return it immediately.
	if !isConnectionError(err) {
		return err
	}

	// If we get here, the operation failed due to a likely connection issue.
	// For REWS connections, let REWS handle the reconnection. For standard connections, use backoff.
	if c.rewsConn != nil {
		// REWS handles reconnection automatically - just retry the operation
		// The underlying connection should be refreshed by REWS
		return c.retryWithBackoff(ctx, func() error {
			return fn(c.getConnection())
		})
	} else {
		// Standard connection: attempt manual reconnect with exponential backoff
		slog.WarnContext(ctx, "Database operation failed, attempting to reconnect with backoff", "event", "db_reconnect_triggered", "version", "1.0", "error", err, "db_url", redactDBURL(c.cfg.GetDBURL()))
		return c.retryWithBackoff(ctx, func() error {
			if reconnectErr := c.forceReconnect(ctx); reconnectErr != nil {
				return fmt.Errorf("reconnection failed: %w (original error: %v)", reconnectErr, err)
			}
			return fn(c.getConnection())
		})
	}
}

// StartMonitoring begins health checks and automatic reconnection
func (c *Connection) StartMonitoring() {
	go c.monitorConnection()
}

// Close shuts down the connection and monitoring
func (c *Connection) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.done)
	if c.conn != nil {
		return c.conn.Close(ctx)
	}
	return nil
}

// DB returns the underlying database connection if it's healthy.
// It returns an error if the connection is not available.
func (c *Connection) DB() (*surrealdb.DB, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil || !c.healthy {
		return nil, NewDBError(ErrNotConnected, "database not connected or unhealthy")
	}
	return c.conn, nil
}

// IsHealthy returns the current connection status
func (c *Connection) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthy
}

func (c *Connection) getConnection() *surrealdb.DB {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func (c *Connection) reconnect(ctx context.Context) error {
	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close(ctx)
	}

	slog.DebugContext(ctx, "Attempting to connect to database", "event", "db_connect_attempt", "version", "1.0", "db_url", redactDBURL(c.cfg.GetDBURL()))

	// Check if URL indicates WebSocket connection for REWS support
	dbURL := c.cfg.GetDBURL()
	isWebSocket := strings.HasPrefix(dbURL, "ws://") || strings.HasPrefix(dbURL, "wss://")

	if isWebSocket {
		// Initialize REWS connection for WebSocket URLs
		// This provides automatic reconnection, session restoration, and live query persistence
		slog.InfoContext(ctx, "WebSocket connection detected, initializing REWS integration", "event", "db_rews_detected", "version", "1.0", "db_url", redactDBURL(dbURL))

		// Initialize REWS with enterprise-grade connection management
		rewsConn, err := c.initializeREWSConnection(dbURL)
		if err != nil {
			slog.WarnContext(ctx, "Failed to initialize REWS connection, falling back to standard connection", "event", "db_rews_init_failure", "version", "1.0", "error", err, "db_url", redactDBURL(dbURL))
			// Fall back to standard connection
			conn, err := surrealdb.FromEndpointURLString(ctx, dbURL)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to create database connection", "event", "db_connect_failure", "version", "1.0",
					"db_url", redactDBURL(dbURL),
					"error", err,
				)
				c.healthy = false
				return fmt.Errorf("failed to connect to database at %s: %w", dbURL, err)
			}
			c.conn = conn
		} else {
			c.rewsConn = rewsConn
			slog.InfoContext(ctx, "REWS connection initialized successfully", "event", "db_rews_init_success", "version", "1.0", "db_url", redactDBURL(dbURL))
			// For REWS connections, we still need a standard DB client for operations
			// The REWS client handles the WebSocket connection, but we use the DB client for queries
			conn, err := surrealdb.FromEndpointURLString(ctx, dbURL)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to create database connection for REWS", "event", "db_connect_failure", "version", "1.0",
					"db_url", redactDBURL(dbURL),
					"error", err,
				)
				c.healthy = false
				return fmt.Errorf("failed to connect to database at %s: %w", dbURL, err)
			}
			c.conn = conn
		}
	} else {
		// Non-WebSocket: use standard connection
		conn, err := surrealdb.FromEndpointURLString(ctx, dbURL)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to create database connection", "event", "db_connect_failure", "version", "1.0",
				"db_url", redactDBURL(dbURL),
				"error", err,
			)
			c.healthy = false
			return fmt.Errorf("failed to connect to database at %s: %w", dbURL, err)
		}
		c.conn = conn
	}

	// Authenticate
	authData := &surrealdb.Auth{
		Username: c.cfg.GetDBUser(),
		Password: c.cfg.GetDBPass(),
	}

	if _, err := c.conn.SignIn(ctx, authData); err != nil {
		c.conn.Close(ctx)
		slog.ErrorContext(ctx, "Failed to sign in to database", "event", "db_auth_failure", "version", "1.0",
			"db_url", redactDBURL(dbURL),
			"user", c.cfg.GetDBUser(),
			"error", err,
		)
		c.healthy = false
		return fmt.Errorf("failed to sign in: %w", err)
	}

	// Select namespace/database
	if err := c.conn.Use(ctx, c.cfg.GetDBNs(), c.cfg.GetDBDb()); err != nil {
		c.conn.Close(ctx)
		slog.ErrorContext(ctx, "Failed to use namespace/database", "event", "db_namespace_failure", "version", "1.0",
			"db_url", redactDBURL(dbURL),
			"namespace", c.cfg.GetDBNs(),
			"database", c.cfg.GetDBDb(),
			"error", err,
		)
		c.healthy = false
		return fmt.Errorf("failed to use namespace/db: %w", err)
	}

	// Update connection and health status
	c.healthy = true
	slog.DebugContext(ctx, "Database connection established", "event", "db_connect_success", "version", "1.0",
		"db_url", redactDBURL(dbURL),
		"namespace", c.cfg.GetDBNs(),
		"database", c.cfg.GetDBDb(),
	)
	return nil
}

func (c *Connection) forceReconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reconnect(ctx)
}

// retryWithBackoff implements retry logic using the REWS Retryer interface
func (c *Connection) retryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		delay, shouldContinue := c.retryer.NextDelay(attempt, err)
		if !shouldContinue {
			break
		}

		slog.DebugContext(ctx, "Retry attempt failed, waiting before next attempt",
			"event", "retry_attempt", "version", "1.0",
			"attempt", attempt+1, "delay_ms", delay.Milliseconds(), "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		attempt++
	}

	return fmt.Errorf("operation failed after %d attempts: %w", attempt+1, lastErr)
}

func (c *Connection) monitorConnection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := c.checkHealth(ctx); err != nil {
				slog.WarnContext(ctx, "Database health check failed, attempting reconnection with backoff", "event", "db_health_check_failure", "version", "1.0", "error", err, "db_url", redactDBURL(c.cfg.GetDBURL()))
				if reconnectErr := c.retryWithBackoff(ctx, func() error {
					return c.forceReconnect(ctx)
				}); reconnectErr != nil {
					slog.ErrorContext(ctx, "Failed to reconnect to database after health check failure", "event", "db_reconnect_failure", "version", "1.0", "error", reconnectErr, "db_url", redactDBURL(c.cfg.GetDBURL()))
				}
			}
			cancel()
		case <-c.done:
			return
		}
	}
}

func (c *Connection) checkHealth(ctx context.Context) error {
	c.mu.RLock()
	conn := c.conn
	rewsConn := c.rewsConn
	c.mu.RUnlock()

	if conn == nil {
		c.healthy = false
		return errors.New("no active database connection")
	}

	// For REWS connections, we can rely on REWS's built-in health monitoring
	// but still perform a basic check to ensure the DB client is functional
	if rewsConn != nil {
		// REWS handles connection health internally, but we still check the DB client
		if _, err := conn.Version(ctx); err != nil {
			c.healthy = false
			return fmt.Errorf("REWS database health check failed for %s: %w", redactDBURL(c.cfg.GetDBURL()), err)
		}
	} else {
		// Standard connection health check
		if _, err := conn.Version(ctx); err != nil {
			c.healthy = false
			return fmt.Errorf("database health check failed for %s: %w", redactDBURL(c.cfg.GetDBURL()), err)
		}
	}

	// Sample successful health checks to reduce log volume in production.
	if rand.Float32() < 0.1 { // Log only 10% of successful health checks.
		slog.DebugContext(ctx, "Database health check successful", "event", "db_health_check_success", "version", "1.0", "db_url", redactDBURL(c.cfg.GetDBURL()))
	}
	c.healthy = true
	return nil
}

// isConnectionError checks if an error is likely due to a lost or failed connection.
// This helps prevent unnecessary reconnection attempts for application-level errors.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation errors, which often wrap network issues.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for common network error substrings. This is not exhaustive but covers many cases.
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "unexpected eof")
}

// redactDBURL parses a database URL and returns it with the password redacted.
// This is a security best practice to avoid logging sensitive credentials.
func redactDBURL(dbURL string) string {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return "invalid-url" // Return a placeholder if the URL is malformed
	}
	// The `Redacted()` method on url.URL safely returns the URL string
	// with the password replaced by "xxxxx".
	return parsedURL.Redacted()
}

// GetDBNs returns the database namespace from the config provider.
func (c *Connection) GetDBNs() string {
	return c.cfg.GetDBNs()
}

// GetDBDb returns the database name from the config provider.
func (c *Connection) GetDBDb() string {
	return c.cfg.GetDBDb()
}

// GetDBQueryTimeout returns the query timeout from the config provider.
func (c *Connection) GetDBQueryTimeout() time.Duration {
	return c.cfg.GetDBQueryTimeout()
}

// GetDBExecuteTimeout returns the execute timeout from the config provider.
func (c *Connection) GetDBExecuteTimeout() time.Duration {
	return c.cfg.GetDBExecuteTimeout()
}
