package pubsub

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// WatermillBridge implements the Publisher and Subscriber interfaces using watermill's GoChannel.
type WatermillBridge struct {
	pub message.Publisher
	sub message.Subscriber
	// Logger for watermill to use
	logger watermill.LoggerAdapter
}

const (
	// Metadata keys used to transfer our Message structure fields through watermill's message.
	metaKeyUserID = "user_id"
	metaKeyTopic  = "topic"
)

// NewWatermillBridge initializes an in-memory Pub/Sub system for testing.
func NewWatermillBridge() *WatermillBridge {
	logger := watermill.NewStdLogger(false, false)
	// GoChannel is a simple in-memory pub/sub implementation.
	goChannel := gochannel.NewGoChannel(
		gochannel.Config{},
		logger,
	)

	return &WatermillBridge{
		pub:    goChannel,
		sub:    goChannel,
		logger: logger,
	}
}

// mapToWatermillMessage converts our pubsub.Message to a watermill message.
func mapToWatermillMessage(msg Message) *message.Message {
	wmMsg := message.NewMessage(watermill.NewUUID(), msg.Payload)

	// Transfer our custom fields to watermill's metadata
	wmMsg.Metadata.Set(metaKeyUserID, msg.UserID)
	wmMsg.Metadata.Set(metaKeyTopic, msg.Topic)

	// Merge any additional metadata
	for k, v := range msg.Metadata {
		wmMsg.Metadata.Set(k, v)
	}

	return wmMsg
}

// mapToPubSubMessage converts a watermill message back to our internal pubsub.Message.
func mapToPubSubMessage(wmMsg *message.Message) Message {
	// Extract our custom fields from watermill's metadata
	userID := wmMsg.Metadata.Get(metaKeyUserID)
	topic := wmMsg.Metadata.Get(metaKeyTopic)

	// Create a new map for additional metadata, excluding our reserved keys
	// but ensuring user_id is present if it exists.
	metadata := make(map[string]string)
	for k, v := range wmMsg.Metadata {
		if k != metaKeyUserID && k != metaKeyTopic {
			metadata[k] = v
		}
	}
	if userID != "" {
		metadata[metaKeyUserID] = userID
	}

	return Message{
		Topic:    topic,
		UserID:   userID,
		Payload:  wmMsg.Payload,
		Metadata: metadata,
	}
}

// Publish implements the Publisher interface.
func (wb *WatermillBridge) Publish(ctx context.Context, msg Message) error {
	wmMsg := mapToWatermillMessage(msg)
	// We use the message's internal topic (msg.Topic) as the watermill topic.
	return wb.pub.Publish(msg.Topic, wmMsg)
}

// Subscribe implements the Subscriber interface.
func (wb *WatermillBridge) Subscribe(ctx context.Context, topic string, handler Handler) error {
	// The Subscribe method returns a channel of messages.
	messages, err := wb.sub.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	// Run the message processing in a separate goroutine so that Subscribe is non-blocking.
	go func() {
		for wmMsg := range messages {
			// Convert the watermill message to our internal structure
			msg := mapToPubSubMessage(wmMsg)

			// Process the message using the provided handler
			if err := handler(ctx, msg); err != nil {
				slog.Error("Failed to handle message", "topic", topic, "msg_id", wmMsg.UUID, "error", err)
				// A non-nil return from the handler means we assume the message was NOT processed successfully.
				// Watermill can be configured to retry, but for the in-memory pub/sub, we acknowledge and log the error.
				wmMsg.Nack()
			} else {
				// Acknowledge the message to signal successful processing.
				wmMsg.Ack()
			}
		}
		slog.Debug("Subscription message loop ended", "topic", topic)
	}()

	// Return immediately, as the subscription is now active and running in the background.
	return nil
}

// Close implements the Publisher and Subscriber interface to shut down the bridge.
func (wb *WatermillBridge) Close() error {
	// Closing the subscriber will close the gochannel and stop message consumption.
	return wb.sub.Close()
}
