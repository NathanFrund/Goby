package wargame

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/wargame/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/topicmgr"
)

type WargameModule struct {
	module.BaseModule
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	renderer   rendering.Renderer
	topicMgr   *topicmgr.Manager
	engine     *Engine
}

type Dependencies struct {
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	Renderer   rendering.Renderer
	TopicMgr   *topicmgr.Manager
}

func New(deps Dependencies) *WargameModule {
	return &WargameModule{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		renderer:   deps.Renderer,
		topicMgr:   deps.TopicMgr,
	}
}

func (m *WargameModule) Name() string {
	return "wargame"
}

func (m *WargameModule) Register(reg *registry.Registry) error {
	slog.Info("Initializing wargame engine")

	// Register all wargame topics
	if err := topics.RegisterTopics(); err != nil {
		return fmt.Errorf("failed to register wargame topics: %w", err)
	}

	m.engine = NewEngine(m.publisher, m.topicMgr)
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
