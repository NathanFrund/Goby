package database

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/connection"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// LiveQueryAction represents the type of change in a live query update
type LiveQueryAction string

const (
	ActionCreate LiveQueryAction = "CREATE"
	ActionUpdate LiveQueryAction = "UPDATE"
	ActionDelete LiveQueryAction = "DELETE"
	ActionClose  LiveQueryAction = "CLOSE"
)

// LiveQueryHandler is called when live query data changes
type LiveQueryHandler func(ctx context.Context, action LiveQueryAction, data interface{})

// LiveQueryFilter defines optional filtering for live queries
type LiveQueryFilter struct {
	Where  string                 // SurrealQL WHERE clause
	Params map[string]interface{} // Query parameters
	Fields []string               // Specific fields to watch (optional)
}

// Subscription represents an active live query subscription
type Subscription struct {
	ID     string
	Table  string
	Active bool
}

// LiveQueryService provides real-time data subscriptions via SurrealDB Live Queries
type LiveQueryService interface {
	// Subscribe to a table with optional WHERE clause
	Subscribe(ctx context.Context, table string, filter *LiveQueryFilter, handler LiveQueryHandler) (*Subscription, error)

	// Subscribe with custom SurrealQL query
	SubscribeQuery(ctx context.Context, query string, params map[string]interface{}, handler LiveQueryHandler) (*Subscription, error)

	// Unsubscribe from updates
	Unsubscribe(subID string) error
}

// SurrealLiveQueryService implements LiveQueryService using SurrealDB
type SurrealLiveQueryService struct {
	db DBConnection // Database connection for live queries

	subscriptions sync.Map // map[string]*subscriptionState
}

type subscriptionState struct {
	id          string
	table       string
	handler     LiveQueryHandler
	active      bool
	cancel      context.CancelFunc
	query       string
	params      map[string]interface{}
	liveQueryID string // SurrealDB live query ID
}

// NewSurrealLiveQueryService creates a new live query service
func NewSurrealLiveQueryService(db DBConnection) *SurrealLiveQueryService {
	return &SurrealLiveQueryService{
		db: db,
	}
}

// Subscribe creates a live query subscription for a table
func (s *SurrealLiveQueryService) Subscribe(ctx context.Context, table string, filter *LiveQueryFilter, handler LiveQueryHandler) (*Subscription, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	if s == nil {
		return nil, fmt.Errorf("live query service is nil")
	}

	// Build field list for SELECT clause
	fieldList := "*"
	if filter != nil && len(filter.Fields) > 0 {
		fieldList = s.buildFieldList(filter.Fields)
	}

	// Build SurrealQL query
	query := fmt.Sprintf("LIVE SELECT %s FROM %s", fieldList, table)

	// Add WHERE clause if provided
	if filter != nil && filter.Where != "" {
		query = fmt.Sprintf("%s WHERE %s", query, filter.Where)
	}

	// Get params from filter or create empty map
	var params map[string]interface{}
	if filter != nil && filter.Params != nil {
		params = filter.Params
	} else {
		params = make(map[string]interface{})
	}

	return s.subscribeQuery(ctx, table, query, params, handler)
}

// SubscribeQuery creates a live query subscription with a custom query
func (s *SurrealLiveQueryService) SubscribeQuery(ctx context.Context, query string, params map[string]interface{}, handler LiveQueryHandler) (*Subscription, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	if s == nil {
		return nil, fmt.Errorf("live query service is nil")
	}

	// Validate that the query is a LIVE SELECT query
	trimmedQuery := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(trimmedQuery, "LIVE SELECT") {
		return nil, fmt.Errorf("query must start with 'LIVE SELECT', got: %s", query)
	}

	table := s.extractTableFromQuery(query)

	// Ensure params is not nil
	if params == nil {
		params = make(map[string]interface{})
	}

	return s.subscribeQuery(ctx, table, query, params, handler)
}

func (s *SurrealLiveQueryService) subscribeQuery(ctx context.Context, table, query string, params map[string]interface{}, handler LiveQueryHandler) (*Subscription, error) {
	subID := uuid.New().String()

	// Create subscription state
	subCtx, cancel := context.WithCancel(context.Background())
	state := &subscriptionState{
		id:      subID,
		table:   table,
		handler: handler,
		active:  true,
		cancel:  cancel,
		query:   query,
		params:  params,
	}

	s.subscriptions.Store(subID, state)

	// Start live query using SurrealDB v1.0.0 API
	err := s.db.WithConnection(ctx, func(dbConn *surrealdb.DB) error {
		slog.Info("Creating live query subscription", "subID", subID, "table", table)

		// Execute the LIVE SELECT query to get the live query UUID
		results, err := surrealdb.Query[interface{}](ctx, dbConn, query, params)
		if err != nil {
			return fmt.Errorf("failed to execute live query: %w", err)
		}

		// Extract the live query UUID from the result
		if results == nil || len(*results) == 0 {
			return fmt.Errorf("live query returned no results")
		}

		result := (*results)[0]
		if result.Status != "OK" {
			return fmt.Errorf("live query failed with status: %s", result.Status)
		}

		// The Result field contains the live query UUID
		// It could be a string, models.UUID, or wrapped in a structure
		if result.Result == nil {
			return fmt.Errorf("live query returned nil result")
		}

		// Try to extract the UUID as a string
		switch v := result.Result.(type) {
		case string:
			state.liveQueryID = v
		case models.UUID:
			state.liveQueryID = v.String()
		case map[string]interface{}:
			// Sometimes the UUID might be in a map with an "id" field
			if id, ok := v["id"].(string); ok {
				state.liveQueryID = id
			} else if id, ok := v["id"].(models.UUID); ok {
				state.liveQueryID = id.String()
			} else {
				return fmt.Errorf("live query result map does not contain 'id' field: %+v", v)
			}
		default:
			return fmt.Errorf("unexpected live query result type: %T, value: %+v", result.Result, result.Result)
		}

		if state.liveQueryID == "" {
			return fmt.Errorf("live query returned empty UUID")
		}

		slog.Info("Live query established", "subID", subID, "liveQueryID", state.liveQueryID)

		// Get the notification channel from the SDK
		notificationChan, err := dbConn.LiveNotifications(state.liveQueryID)
		if err != nil {
			return fmt.Errorf("failed to get notification channel: %w", err)
		}

		// Start goroutine to listen for notifications
		go s.listenForNotifications(subCtx, state, notificationChan)

		// Handle cleanup when subscription is cancelled
		go func() {
			<-subCtx.Done()
			if state.liveQueryID != "" {
				// Use a separate context for cleanup to avoid cancellation issues
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cleanupCancel()

				// Close the notification channel first to stop receiving new notifications
				if err := dbConn.CloseLiveNotifications(state.liveQueryID); err != nil {
					slog.Warn("Failed to close live notifications", "error", err, "liveQueryID", state.liveQueryID)
				}

				// Give a moment for the channel to close cleanly
				time.Sleep(100 * time.Millisecond)

				// Kill the live query on the database side using a parameter
				killQuery := "KILL $liveQueryID"
				killParams := map[string]interface{}{
					"liveQueryID": state.liveQueryID,
				}

				_, err := surrealdb.Query[interface{}](cleanupCtx, dbConn, killQuery, killParams)
				if err != nil {
					slog.Warn("Failed to kill live query", "error", err, "liveQueryID", state.liveQueryID)
				} else {
					slog.Debug("Killed live query", "liveQueryID", state.liveQueryID)
				}
			}
		}()

		return nil
	})

	if err != nil {
		cancel()
		s.subscriptions.Delete(subID)
		return nil, fmt.Errorf("failed to start live query: %w", err)
	}

	return &Subscription{
		ID:     subID,
		Table:  table,
		Active: true,
	}, nil
}

// Unsubscribe removes a live query subscription
func (s *SurrealLiveQueryService) Unsubscribe(subID string) error {
	if state, ok := s.subscriptions.Load(subID); ok {
		subState := state.(*subscriptionState)
		subState.cancel()

		s.subscriptions.Delete(subID)
		slog.Info("Live query subscription removed", "subID", subID)
	}
	return nil
}

// listenForNotifications listens for live query notifications from SurrealDB
func (s *SurrealLiveQueryService) listenForNotifications(ctx context.Context, state *subscriptionState, notificationChan <-chan connection.Notification) {
	defer func() {
		state.active = false
		s.subscriptions.Delete(state.id)
	}()

	slog.Info("Live query listener started", "subID", state.id, "liveQueryID", state.liveQueryID)

	// Listen for notifications on the channel
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Live query listener context cancelled", "subID", state.id)
			return

		case notification, ok := <-notificationChan:
			if !ok {
				// Channel closed
				slog.Debug("Live query notification channel closed", "subID", state.id)
				return
			}

			// Map SurrealDB action to our LiveQueryAction
			var action LiveQueryAction
			switch notification.Action {
			case connection.CreateAction:
				action = ActionCreate
			case connection.UpdateAction:
				action = ActionUpdate
			case connection.DeleteAction:
				action = ActionDelete
			default:
				slog.Warn("Unknown notification action", "subID", state.id, "action", notification.Action)
				continue
			}

			slog.Debug("Live query notification received", "subID", state.id, "action", action)

			// Execute handler in a goroutine to avoid blocking the notification listener
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("Panic in live query handler", "subID", state.id, "panic", r)
					}
				}()

				state.handler(ctx, action, notification.Result)
			}()
		}
	}
}

// buildFieldList creates a field list for SELECT queries
func (s *SurrealLiveQueryService) buildFieldList(fields []string) string {
	if len(fields) == 0 {
		return "*"
	}
	return strings.Join(fields, ", ")
}

// extractTableFromQuery attempts to extract table name from a query
func (s *SurrealLiveQueryService) extractTableFromQuery(query string) string {
	// Simple extraction - in practice, this would need more sophisticated parsing
	parts := strings.Fields(query)
	for i, part := range parts {
		if strings.ToUpper(part) == "FROM" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}
