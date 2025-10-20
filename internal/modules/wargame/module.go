package wargame

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/topics"
)

type WargameModule struct {
	module.BaseModule
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	renderer   rendering.Renderer
	topics     *topics.TopicRegistry
	engine     *Engine
}

type Dependencies struct {
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	Renderer   rendering.Renderer
	Topics     *topics.TopicRegistry
}

func New(deps Dependencies) *WargameModule {
	return &WargameModule{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		renderer:   deps.Renderer,
		topics:     deps.Topics,
	}
}

func (m *WargameModule) Name() string {
	return "wargame"
}

func (m *WargameModule) Register(reg *registry.Registry) error {
	slog.Info("Initializing wargame engine")

	// Register all wargame topics
	if err := RegisterTopics(m.topics); err != nil {
		return fmt.Errorf("failed to register wargame topics: %w", err)
	}

	m.engine = NewEngine(m.publisher, m.topics)
	reg.Set((**Engine)(nil), m.engine)
	return nil
}

func (m *WargameModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	// Create and start the subscriber in a goroutine
	wargameSubscriber := NewSubscriber(m.subscriber, m.publisher, m.renderer)
	go wargameSubscriber.Start(ctx)

	// Register HTTP handlers
	g.GET("/debug/hit", func(c echo.Context) error {
		go m.engine.SimulateHit(c.Request().Context())
		return c.String(http.StatusOK, "Hit event triggered.")
	})

	return nil
}

func (m *WargameModule) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down WargameModule...")
	return nil
}
