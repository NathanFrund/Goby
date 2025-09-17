package hub

import "log/slog"

// Subscriber represents a single client that can subscribe to rendered HTML fragments from the Hub.
// It contains the channel through which the Hub sends byte slices to the client.
type Subscriber struct {
	// Send is a buffered channel of outbound messages. The Hub sends messages
	// to this channel, and the client is responsible for reading from it.
	Send chan []byte
}

// Hub is a generic, concurrent event bus. It maintains the set of active
// subscribers and broadcasts messages to them.
type Hub struct {
	// Registered subscribers.
	subscribers map[*Subscriber]bool

	// Broadcast is the channel for inbound messages from any client.
	// Any component can send a message to this channel to have it broadcast
	// to all subscribers.
	Broadcast chan []byte

	// Register is a channel for new subscribers to register with the hub.
	Register chan *Subscriber

	// Unregister is a channel for subscribers to unregister from the hub.
	Unregister chan *Subscriber
}

// NewHub creates and returns a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		Broadcast:   make(chan []byte),
		Register:    make(chan *Subscriber),
		Unregister:  make(chan *Subscriber),
		subscribers: make(map[*Subscriber]bool),
	}
}

// Run starts the Hub's message processing loop. It must be run in a separate
// goroutine. It listens on its channels and orchestrates all communication.
func (h *Hub) Run() {
	for {
		select {
		case subscriber := <-h.Register:
			h.subscribers[subscriber] = true
			slog.Info("New subscriber registered", "total_subscribers", len(h.subscribers))

		case subscriber := <-h.Unregister:
			if _, ok := h.subscribers[subscriber]; ok {
				delete(h.subscribers, subscriber)
				close(subscriber.Send)
				slog.Info("Subscriber unregistered", "total_subscribers", len(h.subscribers))
			}

		case message := <-h.Broadcast:
			slog.Debug("Broadcasting message", "recipient_count", len(h.subscribers))
			for subscriber := range h.subscribers {
				// Use a non-blocking send. If the subscriber's buffer is full,
				// it suggests the client is lagging or disconnected.
				select {
				case subscriber.Send <- message:
				default:
					// The client's send buffer is full. We assume it's dead or stuck,
					// so we unregister it and close its channel.
					close(subscriber.Send)
					delete(h.subscribers, subscriber)
					slog.Warn("Unregistering slow subscriber", "total_subscribers", len(h.subscribers))
				}
			}
		}
	}
}
