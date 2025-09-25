package registry

// ServiceKey is a type alias for string to provide a bit more type safety for service locator keys.
type ServiceKey string

// Service keys for dependency injection. Using constants prevents typos.
const (
	WargameEngineKey    ServiceKey = "wargame.engine"
	FullChatStoreKey    ServiceKey = "fullchat.store"
	HTMLHubKey          ServiceKey = "hub.html"
	DataHubKey          ServiceKey = "hub.data"
	TemplateRendererKey ServiceKey = "templates.renderer"
	DBConnectionKey     ServiceKey = "database.connection"
	AppConfigKey        ServiceKey = "config.provider"
)
