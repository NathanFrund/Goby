package announcer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/announcer/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
)

// AnnouncerModule demonstrates Live Query integration by publishing user account events
type AnnouncerModule struct {
	module.BaseModule
	liveQueries database.LiveQueryService
	publisher   pubsub.Publisher
	subID       string // Live query subscription ID
}

// Dependencies holds the services required by the AnnouncerModule
type Dependencies struct {
	LiveQueryService database.LiveQueryService
	Publisher        pubsub.Publisher
}

// New creates a new AnnouncerModule instance
func New(deps Dependencies) *AnnouncerModule {
	return &AnnouncerModule{
		liveQueries: deps.LiveQueryService,
		publisher:   deps.Publisher,
	}
}

// Name returns the module name
func (m *AnnouncerModule) Name() string {
	return "announcer"
}

// Register registers the module's services with the registry
func (m *AnnouncerModule) Register(reg *registry.Registry) error {
	// Register our topics first
	if err := m.registerTopics(); err != nil {
		return err
	}

	slog.Info("AnnouncerModule registered")
	return nil
}

// registerTopics registers the announcer module's topics
func (m *AnnouncerModule) registerTopics() error {
	return topics.RegisterTopics()
}

// Boot starts the module and sets up live query subscriptions
func (m *AnnouncerModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	slog.Info("Booting AnnouncerModule...")

	// Subscribe to user table changes with custom query excluding password field
	query := "LIVE SELECT id, email, name FROM user"
	sub, err := m.liveQueries.SubscribeQuery(ctx, query, nil, func(ctx context.Context, action database.LiveQueryAction, data interface{}) {
		m.handleUserChange(ctx, action, data)
	})
	if err != nil {
		slog.Error("Failed to subscribe to user live query", "error", err)
		return err
	}

	m.subID = sub.ID
	slog.Info("AnnouncerModule subscribed to user live query", "subscriptionID", m.subID)

	return nil
}

func (m *AnnouncerModule) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down AnnouncerModule...")

	if m.subID != "" {
		if err := m.liveQueries.Unsubscribe(m.subID); err != nil {
			slog.Error("Failed to unsubscribe from live query", "error", err, "subscriptionID", m.subID)
		}
	}

	return nil
}

// handleUserChange processes user table changes and publishes events
func (m *AnnouncerModule) handleUserChange(ctx context.Context, action database.LiveQueryAction, data interface{}) {
	// Log the user change with human-readable JSON data
	if dataMap, ok := data.(map[string]interface{}); ok {
		if jsonData, err := json.MarshalIndent(dataMap, "", "  "); err == nil {
			slog.Info("AnnouncerModule received user change", "action", action, "data", string(jsonData))
		} else {
			slog.Info("AnnouncerModule received user change", "action", action, "data", data)
		}
	} else {
		slog.Info("AnnouncerModule received user change", "action", action, "data", data)
	}

	// Publish events based on the action
	switch action {
	case database.ActionCreate:
		m.publishUserCreated(ctx, data)
	case database.ActionDelete:
		m.publishUserDeleted(ctx, data)
	default:
		// Log other actions but don't publish events for them
		slog.Debug("AnnouncerModule ignoring action", "action", action)
	}
}

// publishUserCreated publishes a user creation event
func (m *AnnouncerModule) publishUserCreated(ctx context.Context, data interface{}) {
	userData, ok := data.(map[string]interface{})
	if !ok {
		slog.Warn("Unexpected user data format for creation", "data", data)
		return
	}

	// Extract user information
	userID := extractUserID(userData)
	email := extractStringField(userData, "email")
	name := extractStringField(userData, "name")

	eventData := map[string]interface{}{
		"userID":    userID,
		"email":     email,
		"name":      name,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		slog.Error("Failed to marshal user created event", "error", err)
		return
	}

	msg := pubsub.Message{
		Topic:   topics.TopicUserCreated.Name(),
		Payload: payload,
		UserID:  "system",
	}

	if err := m.publisher.Publish(ctx, msg); err != nil {
		slog.Error("Failed to publish user created event", "error", err, "userID", userID)
	} else {
		slog.Info("Published user created event", "userID", userID, "email", email)
	}
}

// publishUserDeleted publishes a user deletion event
func (m *AnnouncerModule) publishUserDeleted(ctx context.Context, data interface{}) {
	userData, ok := data.(map[string]interface{})
	if !ok {
		slog.Warn("Unexpected user data format for deletion", "data", data)
		return
	}

	// Extract user information
	userID := extractUserID(userData)
	email := extractStringField(userData, "email")
	name := extractStringField(userData, "name")

	eventData := map[string]interface{}{
		"userID":    userID,
		"email":     email,
		"name":      name,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		slog.Error("Failed to marshal user deleted event", "error", err)
		return
	}

	msg := pubsub.Message{
		Topic:   topics.TopicUserDeleted.Name(),
		Payload: payload,
		UserID:  "system",
	}

	if err := m.publisher.Publish(ctx, msg); err != nil {
		slog.Error("Failed to publish user deleted event", "error", err, "userID", userID)
	} else {
		slog.Info("Published user deleted event", "userID", userID, "email", email)
	}
}

// extractUserID extracts the user ID from SurrealDB data format
func extractUserID(data map[string]interface{}) string {
	if id, ok := data["id"].(map[string]interface{}); ok {
		if table, tableOk := id["Table"].(string); tableOk {
			if idVal, idOk := id["ID"].(string); idOk {
				return table + ":" + idVal
			}
		}
	}
	return "unknown"
}

// extractStringField safely extracts a string field from data
func extractStringField(data map[string]interface{}, field string) string {
	if value, ok := data[field].(string); ok {
		return value
	}
	return ""
}
