package pubsub

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates a watermill middleware that adds OpenTelemetry tracing
// to publish and subscribe operations, providing visibility into message flows.
func TracingMiddleware(tracer trace.Tracer) func(message.HandlerFunc) message.HandlerFunc {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			ctx := msg.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// Extract topic and user info from metadata
			topic := msg.Metadata.Get(metaKeyTopic)
			userID := msg.Metadata.Get(metaKeyUserID)

			// Create span for message processing
			spanCtx, span := tracer.Start(ctx, fmt.Sprintf("pubsub.process.%s", topic),
				trace.WithAttributes(
					attribute.String("messaging.system", "watermill"),
					attribute.String("messaging.operation", "process"),
					attribute.String("messaging.destination", topic),
					attribute.String("messaging.message_id", msg.UUID),
					attribute.String("user.id", userID),
					attribute.Int("messaging.message_payload_size_bytes", len(msg.Payload)),
				),
			)
			defer span.End()

			// Update message context with span context
			msg.SetContext(spanCtx)

			// Add payload preview for visibility (first 100 chars)
			payloadPreview := string(msg.Payload)
			if len(payloadPreview) > 100 {
				payloadPreview = payloadPreview[:100] + "..."
			}
			span.SetAttributes(attribute.String("messaging.message_payload_preview", payloadPreview))

			// Execute the handler
			producedMessages, err := h(msg)

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}

			// Record produced messages count
			span.SetAttributes(attribute.Int("messaging.messages_produced", len(producedMessages)))

			return producedMessages, nil
		}
	}
}

// PublisherTracingMiddleware wraps a publisher with tracing capabilities
type PublisherTracingMiddleware struct {
	publisher message.Publisher
	tracer    trace.Tracer
}

// NewPublisherTracingMiddleware creates a new publisher with tracing middleware
func NewPublisherTracingMiddleware(publisher message.Publisher, tracer trace.Tracer) *PublisherTracingMiddleware {
	return &PublisherTracingMiddleware{
		publisher: publisher,
		tracer:    tracer,
	}
}

// Publish wraps the publish operation with tracing
func (p *PublisherTracingMiddleware) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		ctx := msg.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Extract user info from metadata
		userID := msg.Metadata.Get(metaKeyUserID)

		// Create span for publish operation
		spanCtx, span := p.tracer.Start(ctx, fmt.Sprintf("pubsub.publish.%s", topic),
			trace.WithAttributes(
				attribute.String("messaging.system", "watermill"),
				attribute.String("messaging.operation", "publish"),
				attribute.String("messaging.destination", topic),
				attribute.String("messaging.message_id", msg.UUID),
				attribute.String("user.id", userID),
				attribute.Int("messaging.message_payload_size_bytes", len(msg.Payload)),
			),
		)
		defer span.End()

		// Add payload preview for visibility
		payloadPreview := string(msg.Payload)
		if len(payloadPreview) > 100 {
			payloadPreview = payloadPreview[:100] + "..."
		}
		span.SetAttributes(attribute.String("messaging.message_payload_preview", payloadPreview))

		// Update message context with tracing context
		msg.SetContext(spanCtx)
	}

	// Publish the messages
	err := p.publisher.Publish(topic, messages...)
	if err != nil {
		// Record error on all spans
		for _, msg := range messages {
			if span := trace.SpanFromContext(msg.Context()); span != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
		}
	}

	return err
}

// Close closes the underlying publisher
func (p *PublisherTracingMiddleware) Close() error {
	return p.publisher.Close()
}
