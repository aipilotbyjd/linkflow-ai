package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate origin against allowed list
		return true
	},
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan *WebSocketMessage
	mu         sync.RWMutex
}

// WebSocketClient represents a single WebSocket connection
type WebSocketClient struct {
	hub      *WebSocketHub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	channels map[string]bool
}

// WebSocketMessage represents a message to be sent via WebSocket
type WebSocketMessage struct {
	Type     string                 `json:"type"`
	Channel  string                 `json:"channel,omitempty"`
	UserID   string                 `json:"userId,omitempty"`
	Data     map[string]interface{} `json:"data"`
	Time     time.Time              `json:"timestamp"`
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan *WebSocketMessage),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *WebSocketHub) broadcastMessage(msg *WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Check if message should be sent to this client
		if msg.UserID != "" && client.userID != msg.UserID {
			continue
		}
		if msg.Channel != "" && !client.channels[msg.Channel] {
			continue
		}

		select {
		case client.send <- data:
		default:
			// Client's send buffer is full, close connection
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(msg *WebSocketMessage) {
	msg.Time = time.Now()
	h.broadcast <- msg
}

// SendToUser sends a message to a specific user
func (h *WebSocketHub) SendToUser(userID string, msg *WebSocketMessage) {
	msg.UserID = userID
	msg.Time = time.Now()
	h.broadcast <- msg
}

// SendToChannel sends a message to all clients subscribed to a channel
func (h *WebSocketHub) SendToChannel(channel string, msg *WebSocketMessage) {
	msg.Channel = channel
	msg.Time = time.Now()
	h.broadcast <- msg
}

// WebSocketHandler handles WebSocket connection upgrades
type WebSocketHandler struct {
	hub *WebSocketHub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *WebSocketHub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// ServeHTTP handles the WebSocket upgrade request
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &WebSocketClient{
		hub:      h.hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		channels: make(map[string]bool),
	}

	h.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WebSocketClient) handleMessage(message []byte) {
	var msg struct {
		Type    string `json:"type"`
		Channel string `json:"channel,omitempty"`
		Token   string `json:"token,omitempty"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "auth":
		// Authenticate the connection
		c.authenticate(msg.Token)
	case "subscribe":
		// Subscribe to a channel
		c.channels[msg.Channel] = true
		c.sendAck("subscribed", msg.Channel)
	case "unsubscribe":
		// Unsubscribe from a channel
		delete(c.channels, msg.Channel)
		c.sendAck("unsubscribed", msg.Channel)
	case "ping":
		c.sendAck("pong", "")
	}
}

func (c *WebSocketClient) authenticate(token string) {
	// Validate JWT token and extract user ID
	// This is a placeholder - implement actual JWT validation
	if token != "" {
		c.userID = "authenticated-user" // Would be extracted from token
		c.sendAck("authenticated", "")
	} else {
		c.sendError("authentication_failed", "Invalid token")
	}
}

func (c *WebSocketClient) sendAck(msgType, channel string) {
	msg := map[string]interface{}{
		"type":      msgType,
		"timestamp": time.Now(),
	}
	if channel != "" {
		msg["channel"] = channel
	}

	data, _ := json.Marshal(msg)
	c.send <- data
}

func (c *WebSocketClient) sendError(code, message string) {
	msg := map[string]interface{}{
		"type":      "error",
		"code":      code,
		"message":   message,
		"timestamp": time.Now(),
	}

	data, _ := json.Marshal(msg)
	c.send <- data
}

// Event types for real-time notifications
const (
	EventWorkflowCreated     = "workflow.created"
	EventWorkflowUpdated     = "workflow.updated"
	EventWorkflowDeleted     = "workflow.deleted"
	EventWorkflowActivated   = "workflow.activated"
	EventWorkflowDeactivated = "workflow.deactivated"
	
	EventExecutionStarted   = "execution.started"
	EventExecutionCompleted = "execution.completed"
	EventExecutionFailed    = "execution.failed"
	EventExecutionCancelled = "execution.cancelled"
	EventNodeStarted        = "execution.node.started"
	EventNodeCompleted      = "execution.node.completed"
	EventNodeFailed         = "execution.node.failed"
	
	EventNotificationNew = "notification.new"
	
	EventSystemHealth = "system.health"
	EventSystemAlert  = "system.alert"
)
