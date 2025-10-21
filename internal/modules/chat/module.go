package chat

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/chat/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/topicmgr"
)

// Topic references for backward compatibility during migration
var (
	// ClientMessageNew is the topic for a client sending a new chat message.
	ClientMessageNew = topics.TopicNewMessage

	// Messages is the topic for broadcasting rendered chat messages to all clients.
	Messages = topics.TopicMessages

	// Direct is the topic for sending a rendered direct message to a specific client.
	Direct = topics.TopicDirectMessage
)

// ChatModule implements the module.Module interface for the chat feature.
type ChatModule struct {
	module.BaseModule
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	renderer   rendering.Renderer
	topicMgr   *topicmgr.Manager
}

// Dependencies holds all the services that the ChatModule requires to operate.
// This struct is used for constructor injection to make dependencies explicit.
type Dependencies struct {
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	Renderer   rendering.Renderer
	TopicMgr   *topicmgr.Manager
}

// New creates a new instance of the ChatModule, injecting its dependencies.
func New(deps Dependencies) *ChatModule {
	return &ChatModule{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		renderer:   deps.Renderer,
		topicMgr:   deps.TopicMgr,
	}
}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// Shutdown is called on application termination.
func (m *ChatModule) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down ChatModule...")
	// In a real module, you might wait for background workers to finish here.
	return nil
}

// Boot sets up the routes and starts background services for the chat module.
func (m *ChatModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	// Register chat module topics
	if err := topics.RegisterTopics(); err != nil {
		return err
	}

	// --- Start Background Services ---

	// Create and start the chat subscriber in a goroutine.
	chatSubscriber := NewChatSubscriber(m.subscriber, m.publisher, m.renderer)
	go chatSubscriber.Start(ctx)

	// Create and start the presence subscriber for real-time presence updates
	presenceSubscriber := NewPresenceSubscriber(m.subscriber, m.publisher, m.renderer)
	go presenceSubscriber.Start(ctx)

	// --- Register HTTP Handlers ---
	slog.Info("Booting ChatModule: Setting up routes...")
	handler := NewHandler(m.publisher)

	// Set up routes - the server mounts us under /app/chat, so we use root paths here
	g.GET("", handler.ChatGet)
	g.POST("/message", handler.MessagePost)
	g.GET("/presence", handler.PresenceGet) // HTML endpoint for presence component

	return nil
}
