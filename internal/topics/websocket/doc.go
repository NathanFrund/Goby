// Package websocket provides WebSocket-specific topic definitions for the Goby application.
//
// This package contains topic definitions for WebSocket communication, organized by message type:
//
// - Broadcast messages: Sent to all connected clients of a specific type (HTML or Data)
// - Direct messages: Sent to specific clients based on recipient ID
//
// Usage example:
//
//	import "github.com/nfrund/goby/internal/topics/websocket"
//
//	// Register all WebSocket topics
//	if err := websocket.RegisterAll(registry); err != nil {
//	    // handle error
//	}
package websocket
