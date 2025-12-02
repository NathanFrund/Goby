package app

import (
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/modules/announcer"
	"github.com/nfrund/goby/internal/modules/examples/chat"
	"github.com/nfrund/goby/internal/modules/examples/profile"
	"github.com/nfrund/goby/internal/modules/examples/wargame"
	"github.com/nfrund/goby/internal/presence"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/script"
	"github.com/nfrund/goby/internal/topicmgr"
)

// Dependencies holds the core services that are required by the application's modules.
// This struct is passed from the main application entrypoint to wire up the modules.
type Dependencies struct {
	Publisher        pubsub.Publisher
	Subscriber       pubsub.Subscriber
	Renderer         rendering.Renderer
	TopicMgr         *topicmgr.Manager
	PresenceService  *presence.Service
	ScriptEngine     script.ScriptEngine
	LiveQueryService database.LiveQueryService
	FileRepository   *database.FileStore
}

// chatDeps creates the dependency struct for the chat module.
func chatDeps(deps Dependencies) chat.Dependencies {
	return chat.Dependencies{
		Publisher:       deps.Publisher,
		Subscriber:      deps.Subscriber,
		Renderer:        deps.Renderer,
		TopicMgr:        deps.TopicMgr,
		PresenceService: deps.PresenceService,
	}
}

// wargameDeps creates the dependency struct for the wargame module.
func wargameDeps(deps Dependencies) wargame.Dependencies {
	return wargame.Dependencies{
		Publisher:    deps.Publisher,
		Subscriber:   deps.Subscriber,
		Renderer:     deps.Renderer,
		TopicMgr:     deps.TopicMgr,
		ScriptEngine: deps.ScriptEngine,
	}
}

// announcerDeps creates the dependency struct for the announcer module.
func announcerDeps(deps Dependencies) announcer.Dependencies {
	return announcer.Dependencies{
		LiveQueryService: deps.LiveQueryService,
		Publisher:        deps.Publisher,
	}
}

// profileDeps creates the dependency struct for the profile module.
func profileDeps(deps Dependencies) profile.Dependencies {
	return profile.Dependencies{
		FileRepository: deps.FileRepository,
	}
}
