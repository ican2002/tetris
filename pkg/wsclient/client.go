package wsclient

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client
type Client struct {
	conn       *websocket.Conn
	url        string
	mu         sync.RWMutex
	connected  bool
	reconnect  bool
	maxRetries int
	retryDelay time.Duration

	// Callbacks
	onStateChange  func([]byte)
	onConnected    func()
	onDisconnected func()
	onError        func(error)
}

// New creates a new WebSocket client
func New(url string) *Client {
	return &Client{
		url:        url,
		reconnect:  true,
		maxRetries: 5,
		retryDelay: 3 * time.Second,
	}
}

// Connect establishes a WebSocket connection
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.connected = true

	if c.onConnected != nil {
		c.onConnected()
	}

	// Start listening for messages
	go c.listen()

	return nil
}

// listen receives messages from the WebSocket server
func (c *Client) listen() {
	defer c.handleDisconnect()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			break
		}

		if c.onStateChange != nil {
			c.onStateChange(message)
		}
	}
}

// handleDisconnect handles connection disconnection
func (c *Client) handleDisconnect() {
	c.mu.Lock()
	c.connected = false
	c.conn.Close()
	c.mu.Unlock()

	if c.onDisconnected != nil {
		c.onDisconnected()
	}

	// Auto-reconnect if enabled
	if c.reconnect {
		c.reconnectLoop()
	}
}

// reconnectLoop attempts to reconnect to the server
func (c *Client) reconnectLoop() {
	for i := 0; i < c.maxRetries; i++ {
		log.Printf("Attempting to reconnect (%d/%d)...", i+1, c.maxRetries)
		time.Sleep(c.retryDelay)

		if err := c.Connect(); err == nil {
			log.Println("Reconnected successfully")
			return
		}
	}

	log.Println("Max reconnection attempts reached")
}

// Send sends a message to the server
func (c *Client) Send(data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return ErrNotConnected
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reconnect = false // Disable reconnect on manual close
	c.connected = false

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// SetOnStateChange sets the callback for state changes
func (c *Client) SetOnStateChange(fn func([]byte)) {
	c.onStateChange = fn
}

// SetOnConnected sets the callback for connection established
func (c *Client) SetOnConnected(fn func()) {
	c.onConnected = fn
}

// SetOnDisconnected sets the callback for disconnection
func (c *Client) SetOnDisconnected(fn func()) {
	c.onDisconnected = fn
}

// SetOnError sets the callback for errors
func (c *Client) SetOnError(fn func(error)) {
	c.onError = fn
}

// SetMaxRetries sets the maximum number of reconnection attempts
func (c *Client) SetMaxRetries(max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxRetries = max
}

// SetRetryDelay sets the delay between reconnection attempts
func (c *Client) SetRetryDelay(delay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.retryDelay = delay
}
