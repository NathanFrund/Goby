package fullchat

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
)

// messageHandler implements the MessageHandler interface
type messageHandler struct {
	service   Service
	clients   map[*websocket.Conn]struct{}
	broadcast chan *Message
	mutex     sync.Mutex
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(service Service) MessageHandler {
	h := &messageHandler{
		service:   service,
		clients:   make(map[*websocket.Conn]struct{}),
		broadcast: make(chan *Message, 100),
	}
	go h.broadcastMessages()
	return h
}

// ChatUI renders the chat interface
func (h *messageHandler) ChatUI(c echo.Context) error {
	// Get recent messages
	messages, err := h.service.GetMessages(c.Request().Context(), 50) // Get last 50 messages
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load messages").SetInternal(err)
	}

	// Reverse the messages to show newest last
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return c.Render(http.StatusOK, "chat.html", map[string]interface{}{
		"Messages": messages,
	})
}

// WebSocketHandler handles WebSocket connections for the chat
func (h *messageHandler) WebSocketHandler(c echo.Context) error {
	ws, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Allow all origins for development
	})
	if err != nil {
		return err
	}
	defer ws.Close(websocket.StatusInternalError, "Internal server error")

	// Register client
	h.mutex.Lock()
	h.clients[ws] = struct{}{}
	h.mutex.Unlock()

	// Send existing messages
	messages, err := h.service.GetMessages(c.Request().Context(), 50)
	if err == nil {
		for _, msg := range messages {
			if err := ws.Write(c.Request().Context(), websocket.MessageText, []byte(msg.Text)); err != nil {
				return err
			}
		}
	}

	// Handle incoming messages
	for {
		_, message, err := ws.Read(c.Request().Context())
		if err != nil {
			break
		}

		// Save the message
		msg, err := h.service.SendMessage(c.Request().Context(), string(message))
		if err != nil {
			continue
		}

		// Broadcast to all clients
		h.broadcast <- msg
	}

	// Unregister client
	h.mutex.Lock()
	delete(h.clients, ws)
	h.mutex.Unlock()

	return ws.Close(websocket.StatusNormalClosure, "")
}

// broadcastMessages sends messages to all connected clients
func (h *messageHandler) broadcastMessages() {
	for msg := range h.broadcast {
		h.mutex.Lock()
		for client := range h.clients {
			go func(ws *websocket.Conn) {
				// Convert message to JSON
				msgJSON, err := json.Marshal(msg)
				if err != nil {
					return
				}
				ws.Write(context.Background(), websocket.MessageText, msgJSON)
			}(client)
		}
		h.mutex.Unlock()
	}
}

// CreateMessage handles the creation of a new message
func (h *messageHandler) CreateMessage(c echo.Context) error {
	var req struct {
		Text string `json:"text"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body").SetInternal(err)
	}

	msg, err := h.service.SendMessage(c.Request().Context(), req.Text)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to send message").SetInternal(err)
	}

	// Broadcast to all clients
	h.broadcast <- msg

	return c.JSON(http.StatusCreated, msg)
}

// ListMessages retrieves a list of messages
func (h *messageHandler) ListMessages(c echo.Context) error {
	limitStr := c.QueryParam("limit")
	limit := 50
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid limit parameter").SetInternal(err)
		}
	}

	messages, err := h.service.GetMessages(c.Request().Context(), limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve messages").SetInternal(err)
	}

	return c.JSON(http.StatusOK, messages)
}
