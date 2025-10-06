package registry

// ServiceKey is a type alias for string to provide a bit more type safety for service locator keys.
type ServiceKey string

// Service keys for dependency injection. Using constants prevents typos.
const (
	WargameEngineKey    ServiceKey = "wargame.engine"
	ChatHandlerKey      ServiceKey = "chat.handler"
	FullChatStoreKey    ServiceKey = "fullchat.store"
	HTMLHubKey          ServiceKey = "hub.html"
	DataHubKey          ServiceKey = "hub.data"
	TemplateRendererKey ServiceKey = "templates.renderer"
	DBConnectionKey     ServiceKey = "database.connection"
	AppConfigKey        ServiceKey = "config.provider"
	UserStoreKey        ServiceKey = "user.store"
	MessengerHandlerKey ServiceKey = "messenger.handler"
	PubSubKey           ServiceKey = "pubsub"
	WebsocketBridgeKey  ServiceKey = "websocket.bridge"
	// NewWebsocketBridgeKey is the key for the new (V2) WebSocket bridge used in the strangler fig migration.
	NewWebsocketBridgeKey ServiceKey = "websocket.bridge.new"
)
