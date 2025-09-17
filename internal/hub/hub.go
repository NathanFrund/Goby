package hub

import "log/slog"

// Subscriber represents a single client connection.
type Subscriber struct {
	// UserID identifies the user associated with this connection.
	UserID string
	// Send is a buffered channel for outbound messages for this specific client.
	Send chan []byte
}

// DirectMessage is a struct for sending a message to a specific user.
type DirectMessage struct {
	UserID  string
	Payload []byte
}

// Hub is a generic, concurrent event bus. It maintains the set of active
// subscribers and broadcasts messages to them.
type Hub struct {
	// A map of all active subscribers, keyed by the subscriber instance.
	subscribers map[*Subscriber]bool

	// A map to look up all subscribers belonging to a specific UserID.
	// This allows for efficient direct messaging.
	userSubscribers map[string]map[*Subscriber]bool

	// Direct is the channel for sending messages to a specific user.
	Direct chan *DirectMessage

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
		Broadcast:       make(chan []byte),
		Direct:          make(chan *DirectMessage),
		Register:        make(chan *Subscriber),
		Unregister:      make(chan *Subscriber),
		subscribers:     make(map[*Subscriber]bool),
		userSubscribers: make(map[string]map[*Subscriber]bool),
	}
}

// Run starts the Hub's message processing loop. It must be run in a separate
// goroutine. It listens on its channels and orchestrates all communication.
func (h *Hub) Run() {
	for {
		select {
		case subscriber := <-h.Register:
			h.subscribers[subscriber] = true
			// Also add the subscriber to the user-specific map.
			if h.userSubscribers[subscriber.UserID] == nil {
				h.userSubscribers[subscriber.UserID] = make(map[*Subscriber]bool)
			}
			h.userSubscribers[subscriber.UserID][subscriber] = true
			slog.Info("New subscriber registered", "userID", subscriber.UserID, "total_subscribers", len(h.subscribers))

		case subscriber := <-h.Unregister:
			if _, ok := h.subscribers[subscriber]; ok {
				delete(h.subscribers, subscriber)
				// Also remove the subscriber from the user-specific map.
				if userSubs, ok := h.userSubscribers[subscriber.UserID]; ok {
					delete(userSubs, subscriber)
					// If this was the user's last connection, remove their entry from the map.
					if len(userSubs) == 0 {
						delete(h.userSubscribers, subscriber.UserID)
					}
				}
				close(subscriber.Send)
				slog.Info("Subscriber unregistered", "userID", subscriber.UserID, "total_subscribers", len(h.subscribers))
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

		case message := <-h.Direct:
			slog.Debug("Sending direct message", "userID", message.UserID)
			// Check if the user has any active subscribers.
			if userSubs, ok := h.userSubscribers[message.UserID]; ok {
				// Send the message to all of their connections.
				for subscriber := range userSubs {
					select {
					case subscriber.Send <- message.Payload:
					default:
						close(subscriber.Send)
						delete(h.subscribers, subscriber)
					}
				}
			}
		}
	}
}
