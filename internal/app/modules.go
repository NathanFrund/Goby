package app

import (
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/chat"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/presence"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/topicmgr"
)

// Dependencies holds the core services that are required by the application's modules.
// This struct is passed from the main application entrypoint to wire up the modules.
type Dependencies struct {
	Publisher       pubsub.Publisher
	Subscriber      pubsub.Subscriber
	Renderer        rendering.Renderer
	TopicMgr        *topicmgr.Manager
	PresenceService *presence.Service
}

// NewModules creates and returns the list of all active modules for the application.
// This is the single source of truth for which features are enabled.
func NewModules(deps Dependencies) []module.Module {

	return []module.Module{
		// Add new application modules here.
		wargame.New(wargame.Dependencies{
			Publisher:  deps.Publisher,
			Subscriber: deps.Subscriber,
			Renderer:   deps.Renderer,
			TopicMgr:   deps.TopicMgr,
		}),
		chat.New(chat.Dependencies{
			Publisher:       deps.Publisher,
			Subscriber:      deps.Subscriber,
			Renderer:        deps.Renderer,
			TopicMgr:        deps.TopicMgr,
			PresenceService: deps.PresenceService,
		}),
	}
}
