package app

import (
	"github.com/nfrund/goby/internal/database"
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
