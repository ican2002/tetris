package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ican2002/tetris/pkg/game"
	"github.com/ican2002/tetris/pkg/protocol"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Client represents a WebSocket client connection
type Client struct {
	id     string
	conn   *websocket.Conn
	send   chan []byte
	server *Server
	game   *game.Game

	// Heartbeat
	lastPong     time.Time
	pingTimer    *time.Timer
	timeoutTimer *time.Timer
}

// Server represents the WebSocket server
type Server struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	// Configuration
	PingInterval time.Duration
	PongTimeout  time.Duration

	// HTTP Server
	httpServer *http.Server
	addr       string
}

// New creates a new WebSocket server
func New(addr string) *Server {
	return &Server{
		clients:      make(map[string]*Client),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		PingInterval: 30 * time.Second,
		PongTimeout:  60 * time.Second,
		addr:         addr,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("WebSocket server starting on %s", s.addr)

	// Start hub routine
	go s.run()

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("WebSocket server shutting down...")

	// Close all client connections
	s.mu.Lock()
	for _, client := range s.clients {
		client.conn.Close()
		close(client.send)
	}
	s.clients = make(map[string]*Client)
	s.mu.Unlock()

	// Shutdown HTTP server
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// run handles client registration and unregistration
func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client.id] = client
			s.mu.Unlock()
			log.Printf("Client registered: %s (total: %d)", client.id, len(s.clients))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client.id]; ok {
				delete(s.clients, client.id)
				close(client.send)
				log.Printf("Client unregistered: %s (total: %d)", client.id, len(s.clients))
			}
			s.mu.Unlock()
		}
	}
}

// handleWebSocket handles WebSocket connection upgrades
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create new client
	client := &Client{
		id:       generateClientID(),
		conn:     conn,
		send:     make(chan []byte, 256),
		server:   s,
		game:     game.New(),
		lastPong: time.Now(),
		pingTimer: time.NewTimer(s.PingInterval),
		timeoutTimer: time.NewTimer(s.PongTimeout),
	}

	// Register client
	s.register <- client

	// Start client routines
	go client.writePump()
	go client.readPump()
	go client.heartbeat()

	// Send initial game state
	client.sendState()
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	clientCount := len(s.clients)
	s.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"clients": clientCount,
	})
}

// handleRoot handles root path requests
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "test-client.html")
}

// readPump handles messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(c.server.PongTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.server.PongTimeout))
		c.lastPong = time.Now()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.handleMessage(message)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump() {
	// Update game state periodically for smooth gameplay
	// Use a longer interval to avoid race conditions with user input
	ticker := time.NewTicker(200 * time.Millisecond)
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

			// Add queued messages to the current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Update game and send state
			c.updateGame()

		case <-c.pingTimer.C:
			// Send ping
			c.sendPing()
		}
	}
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(data []byte) {
	msgType, err := protocol.ParseControlMessage(data)
	if err != nil {
		c.sendError("Invalid message format")
		return
	}

	if !protocol.IsValidControlType(msgType) {
		c.sendError("Unknown message type: " + string(msgType))
		return
	}

	if c.game.IsGameOver() && msgType != protocol.MessageTypePong {
		c.sendError("Game is over")
		return
	}

	switch msgType {
	case protocol.MessageTypeMoveLeft:
		log.Printf("[Client %s] Command: move_left", c.id)
		c.game.MoveLeft()
	case protocol.MessageTypeMoveRight:
		log.Printf("[Client %s] Command: move_right", c.id)
		c.game.MoveRight()
	case protocol.MessageTypeMoveDown:
		log.Printf("[Client %s] Command: move_down", c.id)
		c.game.MoveDown()
	case protocol.MessageTypeRotate:
		log.Printf("[Client %s] Command: rotate", c.id)
		c.game.Rotate()
	case protocol.MessageTypeHardDrop:
		log.Printf("[Client %s] Command: hard_drop", c.id)
		c.game.HardDrop()
	case protocol.MessageTypePause:
		log.Printf("[Client %s] Command: pause", c.id)
		c.game.Pause()
	case protocol.MessageTypeResume:
		log.Printf("[Client %s] Command: resume", c.id)
		c.game.Resume()
	case protocol.MessageTypePong:
		// Pong is handled by SetPongHandler
		return
	}

	c.sendState()

	// Check for game over
	if c.game.IsGameOver() {
		c.sendGameOver()
	}
}

// updateGame updates the game state
func (c *Client) updateGame() {
	if c.game.IsPlaying() {
		c.game.Update()
		c.sendState()

		if c.game.IsGameOver() {
			c.sendGameOver()
		}
	}
}

// sendState sends the current game state to the client
func (c *Client) sendState() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in sendState: %v", r)
		}
	}()

	msg := protocol.NewStateMessage(c.game)
	data, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing state: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		// Channel full or closed, skip this message
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(errMsg string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in sendError: %v", r)
		}
	}()

	msg := protocol.NewErrorMessage(errMsg, 400)
	data, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing error: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		// Channel full or closed, skip this message
	}
}

// sendPing sends a ping message to the client
func (c *Client) sendPing() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in sendPing: %v", r)
		}
	}()

	msg := protocol.NewPingMessage(time.Now().Unix())
	data, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing ping: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		// Channel full or closed, skip this message
	}
}

// sendGameOver sends a game over message to the client
func (c *Client) sendGameOver() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered in sendGameOver: %v", r)
		}
	}()

	msg := protocol.NewGameOverMessage(c.game)
	data, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing game over: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		// Channel full or closed, skip this message
	}
}

// heartbeat manages ping/pong heartbeat
func (c *Client) heartbeat() {
	for {
		select {
		case <-c.pingTimer.C:
			c.sendPing()
			c.pingTimer.Reset(c.server.PingInterval)

		case <-c.timeoutTimer.C:
			log.Printf("Client %s timeout, disconnecting", c.id)
			c.conn.Close()
			return
		}
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return "client_" + time.Now().Format("20060102_150405_000000000")
}
