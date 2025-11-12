# OpenTelemetry Integration for Watermill PubSub

This package provides OpenTelemetry tracing integration for watermill-based pub/sub operations, enabling observability into message flows within your application.

## Features

- **Distributed Tracing**: Track message publish and subscribe operations across your system
- **Message Visibility**: See message payloads and metadata in traces (with configurable preview limits)
- **Zipkin Integration**: Export traces to Zipkin for visualization
- **Non-intrusive**: Optional tracing that doesn't affect existing code when disabled

## Quick Start

### 1. Setup OpenTelemetry

```go
import "github.com/nfrund/goby/internal/pubsub"

func main() {
    ctx := context.Background()

    // Configure tracing (can be controlled via environment variables, config files, etc.)
    tracingConfig := pubsub.TracingConfig{
        Enabled:     true, // Set to false to disable tracing completely
        ServiceName: "my-service",
        ZipkinURL:   "http://localhost:9411/api/v2/spans",
    }

    // Setup OTel with Zipkin exporter
    tracer, cleanup, err := pubsub.SetupOTel(ctx, tracingConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer cleanup()

    // Create pubsub bridge with tracing
    bridge := pubsub.NewWatermillBridgeWithTracer(tracer)

    // Use bridge as normal...
}
```

### Configuration Options

#### Programmatic Configuration

```go
// Disable tracing entirely (no performance impact)
config := pubsub.TracingConfig{Enabled: false}

// Enable with custom settings
config := pubsub.TracingConfig{
    Enabled:     true,
    ServiceName: "chat-service",
    ZipkinURL:   "http://zipkin:9411/api/v2/spans",
}

// Use defaults (tracing disabled by default)
config := pubsub.DefaultTracingConfig()
config.Enabled = true // Enable tracing
```

#### Environment Variable Configuration

Load configuration from environment variables (perfect for Docker, Kubernetes, etc.):

```go
// Load from .env file or environment variables
tracingConfig := pubsub.LoadTracingConfigFromEnv()

tracer, cleanup, err := pubsub.SetupOTel(ctx, tracingConfig)
```

Example `.env` file:

```bash
# Enable tracing for debugging
PUBSUB_TRACING_ENABLED=true
PUBSUB_TRACING_SERVICE_NAME=goby-chat-service
PUBSUB_TRACING_ZIPKIN_URL=http://localhost:9411/api/v2/spans
```

### 2. Publishing Messages

When you publish messages, traces will be created showing:

- Message topic and user ID
- Payload size and preview (first 100 characters)
- Publishing operation timing

```go
msg := pubsub.Message{
    Topic:   "chat.messages.new",
    UserID:  "user123",
    Payload: []byte(`{"text": "Hello world!", "timestamp": "2025-01-01T12:00:00Z"}`),
}

err := bridge.Publish(ctx, msg)
```

### 3. Subscribing to Messages

Message processing is automatically traced, showing:

- Processing operation timing
- Message payload preview
- Any errors during processing

```go
handler := func(ctx context.Context, msg pubsub.Message) error {
    // Your message processing logic
    return nil
}

err := bridge.Subscribe(ctx, "chat.messages.new", handler)
```

## Trace Attributes

The following attributes are automatically added to traces:

### Publish Operations

- `messaging.system`: "watermill"
- `messaging.operation`: "publish"
- `messaging.destination`: The topic name
- `messaging.message_id`: Watermill message UUID
- `user.id`: User ID from message metadata
- `messaging.message_payload_size_bytes`: Size of payload
- `messaging.message_payload_preview`: First 100 chars of payload

### Process Operations

- `messaging.system`: "watermill"
- `messaging.operation`: "process"
- `messaging.destination`: The topic name
- `user.id`: User ID from message metadata
- `messaging.message_payload_size_bytes`: Size of payload
- `messaging.message_payload_preview`: First 100 chars of payload

## Configuration

### Zipkin URL

Configure the Zipkin endpoint URL in `SetupOTel()`:

```go
tracer, cleanup, err := pubsub.SetupOTel(ctx, "my-service", "http://zipkin:9411/api/v2/spans")
```

### Service Name

Set your service name for trace identification:

```go
tracer, cleanup, err := pubsub.SetupOTel(ctx, "chat-service", zipkinURL)
```

## Viewing Traces

1. Start Zipkin: `docker run -d -p 9411:9411 openzipkin/zipkin`
2. Access Zipkin UI at `http://localhost:9411`
3. Generate some pub/sub activity in your application
4. Search for traces with service name "my-service" or operation names starting with "pubsub."

## Security Considerations

- Message payloads are truncated to 100 characters in traces
- Consider the sensitivity of data before enabling tracing in production
- Use appropriate sampling rates to control trace volume

## Testing

Run the tests to verify tracing integration:

```bash
go test ./internal/pubsub -v
```

The tests include:

- Basic tracing middleware functionality
- OTel setup with Zipkin exporter
- Message publish/subscribe operations with tracing
