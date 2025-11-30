package pubsub

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/nfrund/goby/internal/topicmgr"
)

// Event[T] wraps a topic name and provides type-safe publishing.
// It also implements topicmgr.Topic for registry integration.
type Event[T any] struct {
	topicName string
	config    topicmgr.TopicConfig
}

// NewEvent creates a typed event and auto-registers it with the Default Manager.
// It uses reflection to generate the 'Metadata' fields from the struct tags of T.
func NewEvent[T any](name string, description string) Event[T] {
	// 1. Reflect on T to get field names for documentation
	var zero T
	t := reflect.TypeOf(zero)
	fields := make([]string, 0)

	// Handle both struct and pointer to struct
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			// Extract just the name part of the tag (ignore omitempty, etc.)
			if jsonTag != "" && jsonTag != "-" {
				// Simple parsing to get the name before the first comma
				nameEnd := 0
				for nameEnd < len(jsonTag) && jsonTag[nameEnd] != ',' {
					nameEnd++
				}
				fields = append(fields, jsonTag[:nameEnd])
			}
		}
	}

	// 2. Create Config
	// Extract module name from topic (e.g., "wargame.event.damage" -> "wargame")
	module := ""
	if dotIdx := 0; dotIdx < len(name) {
		for i, ch := range name {
			if ch == '.' {
				module = name[:i]
				break
			}
		}
	}

	config := topicmgr.TopicConfig{
		Name:        name,
		Module:      module,
		Description: description,
		Pattern:     name, // Use exact topic name as pattern
		Metadata: map[string]interface{}{
			"payload_fields": fields,
			"type_name":      t.Name(),
			"is_typed":       true,
		},
	}

	// 3. Register with Topic Manager
	// We use MustRegister because events are usually defined at package level (init time)
	// and a failure here means a configuration error that should stop startup.
	topicmgr.Default().MustRegister(topicmgr.DefineModule(config))

	return Event[T]{
		topicName: name,
		config:    config,
	}
}

// Name returns the topic name.
func (e Event[T]) Name() string {
	return e.topicName
}

// Publish sends a typed event. The compiler ensures 'payload' matches 'T'.
func Publish[T any](ctx context.Context, p Publisher, event Event[T], payload T) error {
	// Marshal payload to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Use underlying Publisher interface
	return p.Publish(ctx, Message{
		Topic:   event.Name(),
		Payload: data,
	})
}
