package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	id          string
	conn        *websocket.Conn
	send        chan []byte
	server      *Server
	game        *game.Game
	address     string
	connectTime time.Time

	// Heartbeat
	lastPong     time.Time
	pingTimer    *time.Timer
	timeoutTimer *time.Timer
}

// Server represents the WebSocket server
type Server struct {
	clients         map[string]*Client
	adminClients    map[string]*websocket.Conn
	register        chan *Client
	unregister      chan *Client
	registerAdmin   chan *websocket.Conn
	unregisterAdmin chan *websocket.Conn
	mu              sync.RWMutex
	adminMu         sync.RWMutex

	// Configuration
	PingInterval time.Duration
	PongTimeout  time.Duration
	TotalClients int
	PeakClients  int

	// HTTP Server
	httpServer *http.Server
	addr       string
}

// New creates a new WebSocket server
func New(addr string) *Server {
	return &Server{
		clients:         make(map[string]*Client),
		adminClients:    make(map[string]*websocket.Conn),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		registerAdmin:   make(chan *websocket.Conn),
		unregisterAdmin: make(chan *websocket.Conn),
		PingInterval:    30 * time.Second,
		PongTimeout:     60 * time.Second,
		TotalClients:    0,
		PeakClients:     0,
		addr:            addr,
	}
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/ws/admin", s.handleAdminWebSocket)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/admin", s.handleAdmin)

	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("WebSocket server starting on %s", s.addr)

	// Start hub routine
	go s.run()
	// Start admin broadcast routine
	go s.adminBroadcastLoop()

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
			s.TotalClients++
			if len(s.clients) > s.PeakClients {
				s.PeakClients = len(s.clients)
			}
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

		case conn := <-s.registerAdmin:
			adminID := generateClientID()
			s.adminMu.Lock()
			s.adminClients[adminID] = conn
			s.adminMu.Unlock()
			log.Printf("Admin client registered: %s (total: %d)", adminID, len(s.adminClients))

		case conn := <-s.unregisterAdmin:
			s.adminMu.Lock()
			for id, c := range s.adminClients {
				if c == conn {
					delete(s.adminClients, id)
					log.Printf("Admin client unregistered: %s (total: %d)", id, len(s.adminClients))
					break
				}
			}
			s.adminMu.Unlock()
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
		id:           generateClientID(),
		conn:         conn,
		send:         make(chan []byte, 256),
		server:       s,
		game:         game.New(),
		address:      r.RemoteAddr,
		connectTime:  time.Now(),
		lastPong:     time.Now(),
		pingTimer:    time.NewTimer(s.PingInterval),
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

// handleAdmin handles admin page requests
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "admin-client.html")
}

// handleAdminWebSocket handles admin WebSocket connections
func (s *Server) handleAdminWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Admin WebSocket upgrade error: %v", err)
		return
	}

	// Register admin client
	s.registerAdmin <- conn

	// Read messages to keep connection alive
	go func() {
		defer func() {
			s.unregisterAdmin <- conn
			conn.Close()
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
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
		// Reset the timeout timer when pong is received
		c.timeoutTimer.Reset(c.server.PongTimeout)
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

	if c.game.IsGameOver() && msgType != protocol.MessageTypePong && msgType != protocol.MessageTypeRestart {
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
	case protocol.MessageTypeRestart:
		log.Printf("[Client %s] Command: restart", c.id)
		// Create a new game instance
		c.game = game.New()
	case protocol.MessageTypePong:
		// Application-layer pong - reset timeout timer
		// This is needed because we use application-layer ping/pong
		// instead of WebSocket protocol ping/pong
		c.timeoutTimer.Reset(c.server.PongTimeout)
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
			// Send proper close frame before closing connection
			c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "timeout"))
			c.conn.Close()
			return
		}
	}
}

// generateClientID generates a unique client ID
var clientIDCounter int64
var clientIDMutex sync.Mutex

func generateClientID() string {
	clientIDMutex.Lock()
	defer clientIDMutex.Unlock()
	clientIDCounter++
	return "client_" + time.Now().Format("20060102_150405_000000000") + "_" + strconv.FormatInt(clientIDCounter, 10)
}

// adminBroadcastLoop broadcasts client status to admin clients every second
func (s *Server) adminBroadcastLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		s.broadcastClientStatus()
	}
}

// broadcastClientStatus broadcasts client status to all admin clients
func (s *Server) broadcastClientStatus() {
	// Collect client information
	clientsInfo := s.getClientsInfo()

	// Serialize to JSON
	data, err := json.Marshal(clientsInfo)
	if err != nil {
		log.Printf("Error marshaling client info: %v", err)
		return
	}

	// Send to all admin clients
	s.adminMu.RLock()
	for id, conn := range s.adminClients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Error sending to admin client %s: %v", id, err)
			// Connection error, close and remove
			conn.Close()
			go func() {
				s.unregisterAdmin <- conn
			}()
		}
	}
	s.adminMu.RUnlock()
}

// getClientsInfo returns information about all connected clients
func (s *Server) getClientsInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Prepare client data
	clients := make([]map[string]interface{}, 0, len(s.clients))
	for _, client := range s.clients {
		gameState := client.game.GetState().String()
		score := client.game.GetScore()
		level := client.game.GetLevel()
		lines := client.game.GetLines()

		clients = append(clients, map[string]interface{}{
			"id":          client.id,
			"address":     client.address,
			"connectTime": client.connectTime,
			"gameState":   gameState,
			"score":       score,
			"level":       level,
			"lines":       lines,
		})
	}

	return map[string]interface{}{
		"currentClients": len(s.clients),
		"totalClients":   s.TotalClients,
		"peakClients":    s.PeakClients,
		"clients":        clients,
		"timestamp":      time.Now(),
	}
}
