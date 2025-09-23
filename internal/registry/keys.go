package registry

// ServiceKey is a type alias for string to provide a bit more type safety for service locator keys.
type ServiceKey string

// Service keys for dependency injection. Using constants prevents typos.
const (
	WargameEngineKey  ServiceKey = "wargame.engine"
	FullChatService   ServiceKey = "fullchat.service"
	// To add a new service key:
	// GreeterServiceKey ServiceKey = "greeter.service"
)
