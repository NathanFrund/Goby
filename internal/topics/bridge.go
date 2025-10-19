package topics

import "fmt"

// Bridge topic patterns
const (
	HTMLBroadcastPattern = "bridge.html.broadcast"
	HTMLDirectPattern    = "bridge.html.direct.{recipient}"
	DataBroadcastPattern = "bridge.data.broadcast"
	DataDirectPattern    = "bridge.data.direct.{recipient}"
)

// Bridge topic implementations
type (
	htmlBroadcastTopic struct{ BaseTopic }
	htmlDirectTopic    struct{ BaseTopic }
	dataBroadcastTopic struct{ BaseTopic }
	dataDirectTopic    struct{ BaseTopic }
)

// Ensure all topic types implement the Topic interface
var (
	_ Topic = (*htmlBroadcastTopic)(nil)
	_ Topic = (*htmlDirectTopic)(nil)
	_ Topic = (*dataBroadcastTopic)(nil)
	_ Topic = (*dataDirectTopic)(nil)
)

// Bridge topics
var (
	// HTMLBroadcast broadcasts HTML fragments to all connected clients
	HTMLBroadcast = &htmlBroadcastTopic{
		BaseTopic: NewBaseTopic(
			"html_broadcast",
			"Broadcast HTML fragments to all connected clients",
			HTMLBroadcastPattern,
			HTMLBroadcastPattern,
		),
	}

	// HTMLDirect sends HTML fragments to a specific client
	HTMLDirect = &htmlDirectTopic{
		BaseTopic: NewBaseTopic(
			"html_direct",
			"Send HTML fragments to a specific client",
			HTMLDirectPattern,
			"ws.html.direct.user123",
		),
	}

	// DataBroadcast broadcasts JSON data to all connected clients
	DataBroadcast = &dataBroadcastTopic{
		BaseTopic: NewBaseTopic(
			"data_broadcast",
			"Broadcast JSON data to all connected clients",
			DataBroadcastPattern,
			DataBroadcastPattern,
		),
	}

	// DataDirect sends JSON data to a specific client
	DataDirect = &dataDirectTopic{
		BaseTopic: NewBaseTopic(
			"data_direct",
			"Send JSON data to a specific client",
			DataDirectPattern,
			"bridge.data.direct.user123",
		),
	}
)

// Format implements the Topic interface for htmlDirectTopic
func (t *htmlDirectTopic) Format(vars interface{}) (string, error) {
	defaultVars := map[string]string{
		"recipient": "user123",
	}
	return t.BaseTopic.Format(mergeMaps(vars, defaultVars))
}

// Format implements the Topic interface for dataDirectTopic
func (t *dataDirectTopic) Format(vars interface{}) (string, error) {
	defaultVars := map[string]string{
		"recipient": "user123",
	}
	return t.BaseTopic.Format(mergeMaps(vars, defaultVars))
}

// mergeMaps merges two maps, with values from the second map taking precedence
func mergeMaps(a, b interface{}) map[string]string {
	result := make(map[string]string)
	
	if a != nil {
		if amap, ok := a.(map[string]string); ok {
			for k, v := range amap {
				result[k] = v
			}
		}
	}
	
	if b != nil {
		if bmap, ok := b.(map[string]string); ok {
			for k, v := range bmap {
				result[k] = v
			}
		}
	}
	
	return result
}

func init() {
	// Get the default registry
	registry := Default()

	// Register all bridge topics with validation
	topics := []Topic{
		HTMLBroadcast,
		HTMLDirect,
		DataBroadcast,
		DataDirect,
	}

	for _, topic := range topics {
		if err := ValidateAndRegister(registry, topic); err != nil {
			panic(fmt.Sprintf("failed to register topic %s: %v", topic.Name(), err))
		}
	}
}
