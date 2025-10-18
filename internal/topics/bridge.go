package topics

// Bridge topic patterns
const (
	HTMLBroadcastPattern = "bridge.html.broadcast"
	HTMLDirectPattern    = "bridge.html.direct.{recipient}"
	DataBroadcastPattern = "bridge.data.broadcast"
	DataDirectPattern    = "bridge.data.direct.{recipient}"
)

// Bridge topics
var (
	// HTMLBroadcast broadcasts HTML fragments to all connected clients
	HTMLBroadcast = Topic{
		Name:        "html_broadcast",
		Description: "Broadcast HTML fragments to all connected clients",
		Pattern:     HTMLBroadcastPattern,
		Example:     HTMLBroadcastPattern,
	}
	
	// HTMLDirect sends HTML fragments to a specific client
	HTMLDirect = Topic{
		Name:        "html_direct",
		Description: "Send HTML fragments to a specific client",
		Pattern:     HTMLDirectPattern,
		Example:     "bridge.html.direct.user123",
	}
	
	// DataBroadcast broadcasts JSON data to all connected clients
	DataBroadcast = Topic{
		Name:        "data_broadcast",
		Description: "Broadcast JSON data to all connected clients",
		Pattern:     DataBroadcastPattern,
		Example:     DataBroadcastPattern,
	}
	
	// DataDirect sends JSON data to a specific client
	DataDirect = Topic{
		Name:        "data_direct",
		Description: "Send JSON data to a specific client",
		Pattern:     DataDirectPattern,
		Example:     "bridge.data.direct.user123",
	}
)

func init() {
	// Register all bridge topics with validation
	topics := []Topic{
		HTMLBroadcast,
		HTMLDirect,
		DataBroadcast,
		DataDirect,
	}

	for _, topic := range topics {
		if err := ValidateAndRegister(topic); err != nil {
			panic("failed to register bridge topic: " + err.Error())
		}
	}
}
