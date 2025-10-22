# Goby Performance Optimization Guide

## Overview

This document outlines performance optimizations for the Goby WebSocket + Watermill architecture. The focus is on maximizing throughput and capacity while maintaining the single binary deployment model.

## Current Architecture

```
Client → WebSocket Bridge → Watermill → Module Subscribers → Watermill → WebSocket Bridge → Client
```

**Key Components:**

- Dual WebSocket endpoints (HTML + Data)
- Topic-based pub/sub routing via Watermill
- Module-based subscribers
- Single binary deployment

## Performance Bottlenecks & Solutions

### 1. Message Serialization Overhead

**Problem:** Multiple JSON serialization steps in the message pipeline.

**Current Flow:**

```
WebSocket → JSON → Watermill → JSON → Module → JSON → Watermill → JSON → WebSocket
```

**Solution: Message Pooling**

```go
var messagePool = sync.Pool{
    New: func() interface{} { return &pubsub.Message{} },
}

func (b *Bridge) publishMessage(topic string, payload []byte) {
    msg := messagePool.Get().(*pubsub.Message)
    defer messagePool.Put(msg)

    msg.Topic = topic
    msg.Payload = payload
    b.publisher.Publish(ctx, *msg)
}
```

**Expected Gain:** 15-25% reduction in CPU usage

### 2. Topic Subscription Efficiency

**Problem:** Each client creates multiple Watermill subscriptions.

**Current:**

```go
client.Subscribe("chat.messages")
client.Subscribe("presence.updates")
client.Subscribe("notifications")
// = 3 separate subscriptions per client
```

**Solution: Topic Multiplexing**

```go
type TopicFilter struct {
    clientID string
    topics   map[string]bool
}

func (b *Bridge) handleMultiplexedMessage(msg pubsub.Message) {
    for clientID, client := range b.clients {
        if client.isSubscribedTo(msg.Topic) {
            client.send(msg)
        }
    }
}
```

**Expected Gain:** 20-30% improvement in message routing efficiency

### 3. WebSocket Connection Management

**Problem:** Each WebSocket connection uses separate goroutine + buffers.

**Current:**

```
1000 clients = 1000 goroutines + 1000 read/write buffers
```

**Solution: Connection Pooling**

```go
type ConnectionPool struct {
    workers    int           // Match CPU cores
    clients    chan *Client  // Queue of clients to process
    bufferPool sync.Pool     // Reuse buffers
}

func (p *ConnectionPool) Start() {
    for i := 0; i < p.workers; i++ {
        go p.worker()
    }
}

func (p *ConnectionPool) worker() {
    buffer := p.bufferPool.Get().([]byte)
    defer p.bufferPool.Put(buffer)

    for client := range p.clients {
        p.processClient(client, buffer)
    }
}
```

**Expected Gain:** 30-50% improvement in memory usage and connection handling

### 4. HTML Fragment Caching

**Problem:** Re-rendering identical HTML fragments for multiple clients.

**Solution: Smart Caching**

```go
type HTMLCache struct {
    cache map[string]CachedHTML
    mutex sync.RWMutex
}

type CachedHTML struct {
    content   []byte
    timestamp time.Time
    ttl       time.Duration
}

func (c *HTMLCache) GetOrRender(key string, renderFunc func() []byte) []byte {
    c.mutex.RLock()
    if cached, exists := c.cache[key]; exists {
        if time.Since(cached.timestamp) < cached.ttl {
            c.mutex.RUnlock()
            return cached.content
        }
    }
    c.mutex.RUnlock()

    // Render and cache
    content := renderFunc()
    c.mutex.Lock()
    c.cache[key] = CachedHTML{
        content:   content,
        timestamp: time.Now(),
        ttl:       30 * time.Second,
    }
    c.mutex.Unlock()
    return content
}
```

**Cache Key Strategies:**

- Presence: `presence:count:{userCount}`
- Chat messages: `chat:message:{messageID}`
- User lists: `users:online:{hash}`

**Expected Gain:** 40-60% reduction in template rendering CPU usage

### 5. Message Batching

**Problem:** Individual message processing creates overhead.

**Solution: Batch Processing**

```go
type MessageBatcher struct {
    updates chan Message
    batch   []Message
    timer   *time.Timer
    maxSize int
    maxWait time.Duration
}

func (b *MessageBatcher) Start() {
    b.timer = time.NewTimer(b.maxWait)
    for {
        select {
        case msg := <-b.updates:
            b.batch = append(b.batch, msg)
            if len(b.batch) >= b.maxSize {
                b.flush()
            }
        case <-b.timer.C:
            if len(b.batch) > 0 {
                b.flush()
            }
        }
    }
}

func (b *MessageBatcher) flush() {
    // Process batch of messages
    b.processBatch(b.batch)
    b.batch = b.batch[:0] // Reset slice
    b.timer.Reset(b.maxWait)
}
```

**Recommended Settings:**

- Batch size: 10-50 messages
- Max wait: 5-10ms
- Use for: presence updates, chat messages, notifications

**Expected Gain:** 20-30% improvement in message throughput

### 6. Smart Topic Routing

**Problem:** Broadcasting messages to all subscribers regardless of interest.

**Solution: Targeted Routing**

```go
type TopicRouter struct {
    routes map[string][]string // topic -> client IDs
    mutex  sync.RWMutex
}

func (r *TopicRouter) Subscribe(clientID, topic string) {
    r.mutex.Lock()
    defer r.mutex.Unlock()

    if clients, exists := r.routes[topic]; exists {
        r.routes[topic] = append(clients, clientID)
    } else {
        r.routes[topic] = []string{clientID}
    }
}

func (r *TopicRouter) RouteMessage(topic string, msg []byte) {
    r.mutex.RLock()
    clients := r.routes[topic]
    r.mutex.RUnlock()

    for _, clientID := range clients {
        r.sendToClient(clientID, msg)
    }
}
```

**Expected Gain:** 20-30% reduction in unnecessary message processing

### 7. Presence-Specific Optimizations

**Problem:** Frequent individual presence updates create noise.

**Solution: Presence Batching**

```go
type PresenceBatcher struct {
    updates chan PresenceUpdate
    batch   map[string]PresenceUpdate // userID -> latest update
    timer   *time.Timer
}

func (b *PresenceBatcher) AddUpdate(update PresenceUpdate) {
    select {
    case b.updates <- update:
    default:
        // Drop update if channel is full (backpressure)
    }
}

func (b *PresenceBatcher) processBatch() {
    if len(b.batch) == 0 {
        return
    }

    // Convert to slice and publish single update
    users := make([]string, 0, len(b.batch))
    for userID, update := range b.batch {
        if update.Status == StatusOnline {
            users = append(users, userID)
        }
    }

    b.publishPresenceUpdate(users)

    // Clear batch
    for k := range b.batch {
        delete(b.batch, k)
    }
}
```

**Expected Gain:** 50-70% reduction in presence-related messages

### 8. Message Deduplication

**Problem:** Duplicate messages across HTML/Data bridges.

**Solution: Deduplication Layer**

```go
type MessageDeduplicator struct {
    seen map[string]time.Time
    ttl  time.Duration
    mutex sync.RWMutex
}

func (d *MessageDeduplicator) ShouldProcess(msgID string) bool {
    d.mutex.RLock()
    if lastSeen, exists := d.seen[msgID]; exists {
        shouldProcess := time.Since(lastSeen) > d.ttl
        d.mutex.RUnlock()
        return shouldProcess
    }
    d.mutex.RUnlock()

    d.mutex.Lock()
    d.seen[msgID] = time.Now()
    d.mutex.Unlock()
    return true
}

// Cleanup old entries periodically
func (d *MessageDeduplicator) cleanup() {
    d.mutex.Lock()
    defer d.mutex.Unlock()

    cutoff := time.Now().Add(-d.ttl)
    for msgID, timestamp := range d.seen {
        if timestamp.Before(cutoff) {
            delete(d.seen, msgID)
        }
    }
}
```

**Expected Gain:** 10-20% reduction in duplicate processing

## Watermill Configuration Optimizations

### Memory-Optimized Config

```go
config := gochannel.Config{
    OutputChannelBuffer:            100,    // Reduce from default
    Persistent:                     false,  // In-memory for speed
    BlockPublishUntilSubscriberAck: false,  // Don't wait for acks
}
```

### Batching Config

```go
type WatermillBatchConfig struct {
    MaxBatchSize: 10,
    MaxLatency:   5 * time.Millisecond,
    BufferSize:   1000,
}
```

## Monitoring & Metrics

### Key Performance Indicators

```go
type PerformanceMetrics struct {
    // Throughput
    MessagesPerSecond    int64
    BytesPerSecond      int64

    // Connections
    ActiveConnections    int64
    ConnectionsPerSecond int64

    // Latency
    MessageLatency       time.Duration
    RenderLatency       time.Duration

    // Resources
    MemoryUsage         int64
    CPUUsage           float64
    GoroutineCount     int64

    // Watermill
    QueueDepth         int64
    TopicSubscriptions int64
}
```

### Performance Thresholds

```go
const (
    // Alerts
    MaxMessagesPerSecond = 1000
    MaxConnections      = 1000
    MaxLatency          = 100 * time.Millisecond
    MaxMemoryMB         = 512
    MaxCPUPercent       = 80

    // Scaling triggers
    ScaleUpConnections  = 800
    ScaleUpLatency      = 50 * time.Millisecond
)
```

### Monitoring Implementation

```go
type Monitor struct {
    metrics *PerformanceMetrics
    alerts  chan Alert
}

func (m *Monitor) CheckThresholds() {
    if m.metrics.MessagesPerSecond > MaxMessagesPerSecond {
        m.alerts <- Alert{
            Type:    "throughput",
            Message: "Message rate exceeded threshold",
            Value:   m.metrics.MessagesPerSecond,
        }
    }
}
```

## Implementation Roadmap

### Phase 1: Quick Wins (1-2 weeks)

1. **HTML Fragment Caching** - Biggest impact, easiest to implement
2. **Message Batching** - Good ROI, moderate complexity
3. **Basic Monitoring** - Essential for measuring improvements

**Expected Gain:** 50-80% capacity improvement

### Phase 2: Architecture Improvements (2-4 weeks)

1. **Connection Pooling** - Significant memory savings
2. **Topic Routing Optimization** - Better message targeting
3. **Presence Batching** - Reduce presence noise

**Expected Gain:** Additional 30-50% capacity improvement

### Phase 3: Advanced Optimizations (4-6 weeks)

1. **Message Deduplication** - Fine-tuning
2. **Advanced Caching Strategies** - Context-aware caching
3. **Load Balancing Preparation** - Multi-instance support

**Expected Gain:** Additional 20-30% capacity improvement

## Configuration Templates

### Development Config

```go
type DevConfig struct {
    // Aggressive caching for fast iteration
    HTMLCacheTTL:     5 * time.Second,
    BatchSize:       5,
    BatchTimeout:    10 * time.Millisecond,

    // Detailed logging
    LogLevel:        "debug",
    MetricsEnabled: true,
}
```

### Production Config

```go
type ProdConfig struct {
    // Balanced performance/memory
    HTMLCacheTTL:     30 * time.Second,
    BatchSize:       20,
    BatchTimeout:    5 * time.Millisecond,

    // Optimized logging
    LogLevel:        "info",
    MetricsEnabled: true,

    // Resource limits
    MaxConnections: 1000,
    MaxMemoryMB:   512,
}
```

### High-Scale Config

```go
type HighScaleConfig struct {
    // Aggressive optimization
    HTMLCacheTTL:     60 * time.Second,
    BatchSize:       50,
    BatchTimeout:    2 * time.Millisecond,

    // Minimal logging
    LogLevel:        "warn",
    MetricsEnabled: false,

    // High limits
    MaxConnections: 5000,
    MaxMemoryMB:   2048,
}
```

## Testing & Validation

### Load Testing Script

```bash
#!/bin/bash
# test_load.sh - WebSocket load testing

CONCURRENT_USERS=${1:-100}
DURATION=${2:-60}
SERVER_URL=${3:-"ws://localhost:8080"}

echo "Testing $CONCURRENT_USERS concurrent users for ${DURATION}s"

for i in $(seq 1 $CONCURRENT_USERS); do
    wscat -c "$SERVER_URL/app/ws/html" &
    PIDS+=($!)
done

sleep $DURATION

# Cleanup
for pid in "${PIDS[@]}"; do
    kill $pid 2>/dev/null
done

echo "Load test complete"
```

### Performance Benchmarks

```go
func BenchmarkMessageRouting(b *testing.B) {
    router := NewTopicRouter()

    // Setup 1000 clients with various subscriptions
    for i := 0; i < 1000; i++ {
        clientID := fmt.Sprintf("client_%d", i)
        router.Subscribe(clientID, "chat.messages")
        if i%10 == 0 {
            router.Subscribe(clientID, "presence.updates")
        }
    }

    message := []byte("test message")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        router.RouteMessage("chat.messages", message)
    }
}
```

## Expected Overall Performance Gains

### Baseline (Current)

- **Concurrent Users:** 400-600
- **Messages/sec:** 500-1000
- **Memory Usage:** 200-400MB
- **CPU Usage:** 40-60%

### After All Optimizations

- **Concurrent Users:** 1200-2000 (2-3x improvement)
- **Messages/sec:** 2000-5000 (3-5x improvement)
- **Memory Usage:** 150-300MB (25% reduction)
- **CPU Usage:** 30-45% (25% reduction)

## Notes

- All optimizations maintain single binary deployment
- Backward compatibility preserved
- Incremental implementation possible
- Monitoring essential for validation
- Focus on real bottlenecks, not theoretical optimizations

## Future Considerations

- **Multi-instance deployment** with shared state
- **Database connection pooling** optimizations
- **CDN integration** for static assets
- **Compression** for WebSocket messages
- **Protocol upgrades** (WebSocket → WebTransport)

---

_This guide should be revisited quarterly and updated based on real-world performance data and changing requirements._
