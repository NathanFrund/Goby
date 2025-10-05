package pubsub

import (
	"context"
)

// Message is the structure passed between components on the bus.
// It is intentionally simple to act as a wrapper for raw data.
type Message struct {
	// Topic identifies the channel the message belongs to (e.g., "chat.messages.new").
	Topic string
	// UserID identifies the user who initiated the message.
	UserID string
	// Payload contains the raw message data (e.g., chat text, JSON).
	Payload []byte
	// Metadata can contain arbitrary key-value pairs for context (e.g., timestamps).
	Metadata map[string]string
}

// Handler defines the function signature for processing a received message.
type Handler func(ctx context.Context, msg Message) error

// Publisher defines the contract for sending messages to the Pub/Sub system.
type Publisher interface {
	Publish(ctx context.Context, msg Message) error
	Close() error
}

// Subscriber defines the contract for receiving messages from the Pub/Sub system.
type Subscriber interface {
	// Subscribe starts listening to the given topic, processing messages with the handler.
	// It blocks until the context is canceled or an irrecoverable error occurs.
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Close() error
}
