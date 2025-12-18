// Package handlers provides HTTP and WebSocket handlers for the gateway
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// MessageType defines WebSocket message types
type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
	MessageTypeEvent       MessageType = "event"
	MessageTypeError       MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Event     string          `json:"event,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	UserID   string
	TenantID string
	Conn     *websocket.Conn
	Channels map[string]bool
	Send     chan []byte
	hub      *Hub
	mu       sync.RWMutex
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	channels   map[string]map[*Client]bool
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	Channel string
	Event   string
	Data    interface{}
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		channels:   make(map[string]map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)

				// Remove from all channels
				for channel := range client.Channels {
					if clients, ok := h.channels[channel]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.channels, channel)
						}
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.broadcastToChannel(message)
		}
	}
}

func (h *Hub) broadcastToChannel(message *BroadcastMessage) {
	h.mu.RLock()
	clients, ok := h.channels[message.Channel]
	h.mu.RUnlock()

	if !ok {
		return
	}

	data, err := json.Marshal(message.Data)
	if err != nil {
		return
	}

	msg := Message{
		Type:      MessageTypeEvent,
		Channel:   message.Channel,
		Event:     message.Event,
		Data:      data,
		Timestamp: time.Now(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	for client := range clients {
		select {
		case client.Send <- msgBytes:
		default:
			h.mu.RUnlock()
			h.unregister <- client
			h.mu.RLock()
		}
	}
	h.mu.RUnlock()
}

// Subscribe adds a client to a channel
func (h *Hub) Subscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true

	client.mu.Lock()
	client.Channels[channel] = true
	client.mu.Unlock()
}

// Unsubscribe removes a client from a channel
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.channels[channel]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.channels, channel)
		}
	}

	client.mu.Lock()
	delete(client.Channels, channel)
	client.mu.Unlock()
}

// Broadcast sends a message to all clients in a channel
func (h *Hub) Broadcast(channel, event string, data interface{}) {
	h.broadcast <- &BroadcastMessage{
		Channel: channel,
		Event:   event,
		Data:    data,
	}
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// ServeHTTP handles WebSocket upgrade requests
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Get user info from context (set by auth middleware)
	userID := r.URL.Query().Get("userId")
	tenantID := r.URL.Query().Get("tenantId")

	client := &Client{
		ID:       generateClientID(),
		UserID:   userID,
		TenantID: tenantID,
		Conn:     conn,
		Channels: make(map[string]bool),
		Send:     make(chan []byte, 256),
		hub:      h.hub,
	}

	h.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	// Send welcome message
	welcome := Message{
		Type:      MessageTypeEvent,
		Event:     "connected",
		Data:      json.RawMessage(`{"message":"Connected to LinkFlow WebSocket"}`),
		Timestamp: time.Now(),
	}
	if data, err := json.Marshal(welcome); err == nil {
		client.Send <- data
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(&msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypeSubscribe:
		if msg.Channel != "" {
			c.hub.Subscribe(c, msg.Channel)
			response := Message{
				Type:      MessageTypeEvent,
				Event:     "subscribed",
				Channel:   msg.Channel,
				Timestamp: time.Now(),
			}
			if data, err := json.Marshal(response); err == nil {
				c.Send <- data
			}
		}

	case MessageTypeUnsubscribe:
		if msg.Channel != "" {
			c.hub.Unsubscribe(c, msg.Channel)
			response := Message{
				Type:      MessageTypeEvent,
				Event:     "unsubscribed",
				Channel:   msg.Channel,
				Timestamp: time.Now(),
			}
			if data, err := json.Marshal(response); err == nil {
				c.Send <- data
			}
		}

	case MessageTypePing:
		response := Message{
			Type:      MessageTypePong,
			Timestamp: time.Now(),
		}
		if data, err := json.Marshal(response); err == nil {
			c.Send <- data
		}
	}
}

func generateClientID() string {
	return "client-" + time.Now().Format("20060102150405.000")
}

// Common channel names
const (
	ChannelExecutions    = "executions"
	ChannelWorkflows     = "workflows"
	ChannelNotifications = "notifications"
	ChannelSystem        = "system"
)

// Helper functions for broadcasting events

// BroadcastExecutionUpdate broadcasts an execution update
func (h *Hub) BroadcastExecutionUpdate(executionID string, status string, data interface{}) {
	h.Broadcast(ChannelExecutions, "execution.updated", map[string]interface{}{
		"executionId": executionID,
		"status":      status,
		"data":        data,
	})
}

// BroadcastWorkflowUpdate broadcasts a workflow update
func (h *Hub) BroadcastWorkflowUpdate(workflowID string, action string, data interface{}) {
	h.Broadcast(ChannelWorkflows, "workflow."+action, map[string]interface{}{
		"workflowId": workflowID,
		"data":       data,
	})
}

// BroadcastNotification broadcasts a notification
func (h *Hub) BroadcastNotification(userID string, notification interface{}) {
	h.Broadcast(ChannelNotifications+"."+userID, "notification", notification)
}

// BroadcastSystemEvent broadcasts a system event
func (h *Hub) BroadcastSystemEvent(event string, data interface{}) {
	h.Broadcast(ChannelSystem, event, data)
}
