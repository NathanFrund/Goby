package websocket

import (
	"sync"
)

// ClientManager manages the lifecycle of WebSocket clients.
type ClientManager struct {
	clients map[string]*Client
	users   map[string]map[string]bool // Maps userID to a set of clientIDs
	mu      sync.RWMutex
}

// NewClientManager creates a new ClientManager.
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*Client),
		users:   make(map[string]map[string]bool),
	}
}

// Add registers a new client.
func (m *ClientManager) Add(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client.ID] = client

	if client.UserID != "" {
		if _, ok := m.users[client.UserID]; !ok {
			m.users[client.UserID] = make(map[string]bool)
		}
		m.users[client.UserID][client.ID] = true
	}
}

// Remove unregisters a client and closes its connection.
func (m *ClientManager) Remove(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[clientID]; ok {
		delete(m.clients, clientID)

		if client.UserID != "" && m.users[client.UserID] != nil {
			delete(m.users[client.UserID], clientID)
			if len(m.users[client.UserID]) == 0 {
				delete(m.users, client.UserID)
			}
		}
		// Close the send channel to terminate the writePump
		close(client.Send)
	}
}

// GetByUser returns all clients for a given user ID.
func (m *ClientManager) GetByUser(userID string) []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var userClients []*Client
	if clientIDs, ok := m.users[userID]; ok {
		for id := range clientIDs {
			if client, ok := m.clients[id]; ok {
				userClients = append(userClients, client)
			}
		}
	}
	return userClients
}

// GetAll returns all currently connected clients.
func (m *ClientManager) GetAll() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allClients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		allClients = append(allClients, client)
	}
	return allClients
}
