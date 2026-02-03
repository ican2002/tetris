# Design: WebSocket Communication Layer

## Overview

This document describes the architectural design of the WebSocket communication layer that enables real-time multiplayer Tetris gameplay. The layer provides bidirectional communication between the game server and clients, supporting concurrent game sessions for multiple players.

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Server                              │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    WebSocket Server                        │ │
│  │  ┌──────────┐  ┌────────────┐  ┌──────────────────────┐   │ │
│  │  │  Hub     │  │   Clients  │  │   Admin Clients      │   │ │
│  │  │          │  │  map[id]   │  │   map[id]            │   │ │
│  │  │ - run()  │  │            │  │                       │   │ │
│  │  │          │  │  Client    │  │   Conn               │   │ │
│  │  └──────────┘  │  - game    │  └──────────────────────┘   │ │
│  │                │  - conn     │                             │ │
│  │                │  - send     │                             │ │
│  │                └────────────┘                             │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
        ▲                                    │
        │ WebSocket                          │ WebSocket
        │ JSON                               │ JSON
        │                                    │
┌───────┴──────────┐               ┌────────┴─────────┐
│  Game Client     │               │   Admin Client   │
│  (Terminal/Web)  │               │   (Dashboard)    │
└──────────────────┘               └──────────────────┘
```

### Package Structure

```
pkg/
├── server/
│   ├── server.go        # WebSocket server and hub
│   └── server_test.go
├── wsclient/
│   ├── client.go        # WebSocket client (for terminal UI)
│   └── errors.go
└── protocol/
    ├── message.go       # Message types and serialization
    └── message_test.go
```

## HTTP Endpoints

The server provides several HTTP endpoints in addition to WebSocket connections:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/ws` | WebSocket | Main game client connection |
| `/ws/admin` | WebSocket | Admin dashboard connection |
| `/` | GET | Web test client (test-client.html) |
| `/admin` | GET | Admin dashboard (admin-client.html) |
| `/health` | GET | Health check endpoint |

**Health Check Response**:
```json
{
  "status": "ok",
  "clients": 5
}
```

## Core Components

### 1. Protocol Layer (`pkg/protocol/`)

**Purpose**: Define message format and types for client-server communication.

**Message Structure**:
```go
type Message struct {
    Type MessageType `json:"type"`
    Data interface{} `json:"data,omitempty"`
}
```

**Message Types**:

| Direction | Type | Purpose |
|-----------|------|---------|
| Client → Server | `move_left`, `move_right`, `move_down` | Movement controls |
| Client → Server | `rotate` | Rotate piece |
| Client → Server | `hard_drop` | Instant drop |
| Client → Server | `pause`, `resume`, `toggle_pause` | Game state controls |
| Client → Server | `restart` | Start new game |
| Server → Client | `state` | Game state update |
| Server → Client | `error` | Error notifications |
| Server → Client | `game_over` | Game end notification |

**State Message Format**:
```go
type StateMessage struct {
    Board        [][]string `json:"board"`           // 10x20 grid
    CurrentPiece PieceData  `json:"current_piece"`
    NextPiece    PieceData  `json:"next_piece"`
    State        string     `json:"state"`           // playing/paused/gameover
    Score        int        `json:"score"`
    Level        int        `json:"level"`
    Lines        int        `json:"lines"`
    DropInterval int        `json:"drop_interval_ms"`
}
```

**Design Decisions**:

1. **JSON Format**:
   - Human-readable for debugging
   - Widely supported across languages
   - Easy to parse and generate

2. **Type-Discriminated Messages**:
   - Single field (`type`) determines message structure
   - Simple dispatch logic on receiving end
   - Easy to extend with new message types

3. **Bidirectional Message Types**:
   - Control messages (C→S): lightweight, just the type
   - State messages (S→C): comprehensive game state

### 2. WebSocket Server (`pkg/server/`)

**Purpose**: Accept client connections, manage game sessions, and handle message routing.

**Key Types**:
```go
type Server struct {
    clients         map[string]*Client
    adminClients    map[string]*websocket.Conn
    register        chan *Client
    unregister      chan *Client
    registerAdmin   chan *websocket.Conn
    unregisterAdmin chan *websocket.Conn
    mu              sync.RWMutex
    adminMu         sync.RWMutex

    // Statistics tracking
    TotalClients int  // Total connections since server start
    PeakClients  int  // Peak concurrent connections

    PingInterval time.Duration
    PongTimeout  time.Duration

    httpServer *http.Server
    addr       string
}

type Client struct {
    id          string
    conn        *websocket.Conn
    send        chan []byte
    server      *Server
    game        *game.Game
    address     string         // Client remote address
    connectTime time.Time      // Connection timestamp
}
```

**Concurrency Model**:

The server uses the **Hub Pattern** with goroutines per connection:

```
                    ┌──────────────┐
                    │    Server    │
                    │      Hub     │
                    └──────┬───────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │  Client 1  │  │  Client 2  │  │  Client N  │
    │            │  │            │  │            │
    │ readPump() │  │ readPump() │  │ readPump() │
    │ writePump()│  │ writePump()│  │ writePump()│
    └────────────┘  └────────────┘  └────────────┘
```

1. **Hub Routine** (`run()`):
   ```go
   for {
       select {
       case client := <-register:
           clients[client.id] = client
       case client := <-unregister:
           delete(clients, client.id)
           close(client.send)
       }
   }
   ```
   - Single goroutine manages all client registration
   - Thread-safe client map updates via channels
   - O(1) add/remove operations

2. **Client Routines** (per connection):

   **readPump**: Receives messages from client
   ```go
   for {
       _, message, err := conn.ReadMessage()
       if err != nil { break }

       // Parse and handle command
       client.handleMessage(message)
   }
   ```

   **writePump**: Sends messages to client
   ```go
   ticker := time.NewTicker(gameUpdateInterval)
   pingTicker := time.NewTicker(PingInterval)

   for {
       select {
       case message := <-send:
           conn.Write(message)
       case <-ticker.C:
           updateGame()
           sendState()
       case <-pingTicker.C:
           conn.WriteMessage(PingMessage, nil)
       }
   }
   ```

**Heartbeat Mechanism**:

- Server sends WebSocket `PingMessage` every 30 seconds
- Client responds with `PongMessage` (handled by WebSocket library)
- Read deadline is reset on each pong
- Connection closes if no pong within 60 seconds

**Benefits**:
- Detects dead connections early
- Prevents resource leaks
- Standard WebSocket protocol feature

### 3. Admin Dashboard

**Purpose**: Provide real-time monitoring and statistics of all connected game clients.

**Architecture**:

```
┌─────────────────────────────────────────────────────────┐
│                   Admin Dashboard                       │
│  ┌───────────────────────────────────────────────────┐  │
│  │         WebSocket Connection (/ws/admin)          │  │
│  └───────────────────┬───────────────────────────────┘  │
│                      │                                  │
│                      ▼                                  │
│  ┌───────────────────────────────────────────────────┐  │
│  │      Server Broadcast (1 second interval)        │  │
│  │  - Current client count                           │  │
│  │  - Total connections (cumulative)                  │  │
│  │  - Peak concurrent connections                     │  │
│  │  - Per-client details                             │  │
│  └───────────────────────────────────────────────────┘  │
│                      │                                  │
│                      ▼                                  │
│  ┌───────────────────────────────────────────────────┐  │
│  │         Real-time Statistics Display              │  │
│  │  - Current connections                            │  │
│  │  - Total connections                              │  │
│  │  - Peak connections                               │  │
│  │  - Playing games count                            │  │
│  │  - Client list with details                       │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

**Admin Broadcast Loop**:

```go
func (s *Server) adminBroadcastLoop() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        <-ticker.C
        s.broadcastClientStatus()
    }
}
```

**Client Information Structure**:
```json
{
  "currentClients": 5,
  "totalClients": 127,
  "peakClients": 15,
  "timestamp": "2026-01-20T12:34:56Z",
  "clients": [
    {
      "id": "client_20260120_123456_123456789_1",
      "address": "192.168.1.100:54321",
      "connectTime": "2026-01-20T12:30:00Z",
      "gameState": "playing",
      "score": 1250,
      "level": 3,
      "lines": 27
    }
  ]
}
```

**Features**:

1. **Real-Time Statistics**:
   - Current connected clients
   - Total connections since server start
   - Peak concurrent connections
   - Number of actively playing games

2. **Per-Client Details**:
   - Unique client ID
   - Remote address (IP:port)
   - Connection time
   - Current game state (playing/paused/gameover)
   - Score, level, lines cleared

3. **Auto-Reconnect**: Dashboard automatically reconnects if connection is lost

4. **Responsive Design**: Works on desktop and mobile devices

**Security Note**:
- Admin dashboard currently has no authentication
- In production, should require authentication
- Should be restricted to admin users only

### 4. WebSocket Client (`pkg/wsclient/`)

**Purpose**: Connect to server, send commands, receive state updates.

**Key Types**:
```go
type Client struct {
    conn       *websocket.Conn
    url        string
    mu         sync.RWMutex
    connected  bool
    reconnect  bool
    maxRetries int
    retryDelay time.Duration
    send       chan []byte
    sendMu     sync.Mutex

    // Callbacks
    onStateChange  func([]byte)
    onConnected    func()
    onDisconnected func()
    onError        func(error)
}
```

**Concurrency Model**:

```
┌─────────────────────────────────────┐
│         Client                      │
├─────────────────────────────────────┤
│                                     │
│  ┌──────────────────────────────┐  │
│  │     readPump (listen)        │  │
│  │  - Receives server messages  │  │
│  │  - Calls onStateChange()     │  │
│  │  - Handles ping/pong         │  │
│  └──────────────────────────────┘  │
│                                     │
│  ┌──────────────────────────────┐  │
│  │     writePump                │  │
│  │  - Sends commands            │  │
│  │  - Thread-safe writes        │  │
│  └──────────────────────────────┘  │
│                                     │
│  ┌──────────────────────────────┐  │
│  │   reconnectLoop             │  │
│  │  - Auto-reconnect on loss   │  │
│  └──────────────────────────────┘  │
└─────────────────────────────────────┘
```

**Auto-Reconnect Logic**:
```go
func (c *Client) reconnectLoop() {
    for i := 0; i < c.maxRetries; i++ {
        time.Sleep(c.retryDelay)
        if err := c.Connect(); err == nil {
            return  // Success
        }
    }
    // Max retries reached, give up
}
```

**Ping/Pong Handling**:
- Server sends `ping` message
- Client automatically responds with `pong`
- Ping messages don't trigger `onStateChange` callback

### 5. Web UI Clients

The project includes two HTML-based clients that can be served directly by the game server.

#### Web Test Client (`test-client.html`)

**Purpose**: Browser-based Tetris game for testing and demonstration.

**Features**:
- Full game board rendering with CSS Grid
- Real-time game state display (score, level, lines, pieces)
- Control buttons and keyboard support
- Message log for debugging
- Auto-reconnect on disconnect

**Controls**:
- Arrow keys: Move/Rotate
- Space: Hard drop
- P: Pause/Resume
- Buttons: Click controls

**Technical Details**:
- Pure JavaScript (no frameworks)
- CSS Grid for board layout
- WebSocket for real-time communication
- Responsive design for mobile/desktop

**Architecture**:
```
test-client.html
       │
       ▼
┌─────────────────────────┐
│   WebSocket (/ws)        │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Message Handler         │
│  - state → updateBoard() │
│  - error → log()         │
│  - ping → send(pong)     │
│  - game_over → show UI   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│   DOM Updates           │
│  - Board cells          │
│  - Info panel           │
│  - Control buttons      │
└─────────────────────────┘
```

#### Admin Dashboard (`admin-client.html`)

**Purpose**: Web-based admin interface for monitoring server statistics.

**Features**:
- Real-time statistics cards
- Live client table with sorting
- Auto-reconnect with status indicator
- Responsive layout
- Color-coded game states

**Architecture**:
```
admin-client.html
       │
       ▼
┌─────────────────────────┐
│  WebSocket (/ws/admin)   │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│  Client Status Update   │
│  (1 second interval)    │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│   Statistics Update     │
│  - Current clients      │
│  - Total clients        │
│  - Peak clients         │
│  - Playing games        │
│  - Client table         │
└─────────────────────────┘
```

### 6. Game Loop Integration

**Server-Side Game Loop**:

Each client has its own game instance, updated periodically:

```go
// In writePump
gameTicker := time.NewTicker(200 * time.Millisecond)

for {
    select {
    case <-gameTicker.C:
        client.updateGame()     // Calls game.Update()
        client.sendState()      // Send state to client
    }
}
```

**Why 200ms?**:
- Balances smoothness (5 FPS) with server load
- Game logic updates are cheap
- Prevents race conditions with user input
- User input immediately triggers state update anyway

**Client Input Flow**:

```
User presses key
       │
       ▼
Client sends command (move_left)
       │
       ▼
Server receives in readPump
       │
       ▼
server.handleMessage()
       │
       ▼
game.MoveLeft()
       │
       ▼
client.sendState()
       │
       ▼
Client receives in listen()
       │
       ▼
onStateChange() callback
       │
       ▼
UI redraws
```

## Message Flow Diagrams

### Connection Establishment

```
Client                    Server
  │                         │
  ├──── WebSocket Open ────►│
  │                         │
  │◄──── Initial State ─────┤
  │    (game started)       │
  │                         │
```

### Gameplay Loop

```
Client                    Server
  │                         │
  ├───── move_left ───────►│
  │                         │
  │◄──── state ────────────┤
  │    (updated board)      │
  │                         │
  │  [auto-drop timer]      │
  │◄──── state ────────────┤
  │    (piece moved down)   │
  │                         │
```

### Heartbeat

```
Client                    Server
  │                         │
  │◄─── PingMessage (WS) ───┤
  │                         │
  ├──── PongMessage (WS) ──►│
  │  (deadline reset)       │
```

### Disconnection

```
Client                    Server
  │                         │
  ├─ CloseMessage ────────►│
  │                         │
  │                         │  (hub unregisters)
  │                         │  (game session ends)
```

## Concurrency Safety

### Server-Side

1. **Client Map**: Protected by `sync.RWMutex`
   - Write lock for register/unregister
   - Read lock for broadcasting

2. **Game Instance**: Each client has its own `*game.Game`
   - No shared game state between clients
   - Game's internal mutex protects its state

3. **Channel Communication**:
   - `register`/`unregister` channels serialize hub operations
   - `send` channel per client serializes writes

### Client-Side

1. **Connection State**: `sync.RWMutex` protects `connected` flag
2. **Send Channel**: `sendMu` prevents duplicate closes
3. **Non-Blocking Send**: `select` with `default` case prevents blocking

## Error Handling

### Server Errors

| Error Type | Handling |
|------------|----------|
| Invalid message format | Send error message, keep connection |
| Unknown command | Send error message, keep connection |
| Game over + input | Send "game over" error |
| Write timeout | Close connection, cleanup |
| Read timeout | Close connection, cleanup |

### Client Errors

| Error Type | Handling |
|------------|----------|
| Connection refused | Retry (up to maxRetries) |
| Connection lost | Auto-reconnect |
| Invalid server response | Call `onError` callback |
| Max retries reached | Give up, notify application |

## Performance Considerations

1. **Per-Client Goroutines**: 2 goroutines per client (read + write)
   - 100 clients = 200 goroutines
   - Go handles this easily (goroutines are lightweight)

2. **Channel Buffers**: `send chan []byte` with size 256
   - Allows bursty writes without blocking
   - Prevents slow clients from blocking server

3. **State Serialization**:
   - Board: 10×20 = 200 cells
   - JSON size: ~2-3 KB per state message
   - At 5 FPS = ~10-15 KB/sec per client

4. **Memory**:
   - Each game instance: ~10 KB
   - Each client struct: ~20 KB
   - 100 concurrent clients: ~3 MB total

## Security Considerations (Future)

### Current (Development)
- `CheckOrigin: func(r *http.Request) bool { return true }`
- Allows all origins for easy testing

### Production Recommendations
1. **Origin Validation**: Whitelist allowed origins
2. **Rate Limiting**: Per-client command rate limits
3. **Authentication**: JWT or session tokens
4. **TLS**: WSS (WebSocket Secure) for encrypted communication

## Testing Strategy

### Unit Tests
- Message serialization/deserialization
- Control message validation
- State message generation

### Integration Tests
- Client connection/disconnection
- Command execution and state updates
- Heartbeat mechanism
- Error handling

### Load Tests
- Concurrent connections (100+ clients)
- Message throughput
- Memory usage over time

## Future Enhancements

1. **Session Persistence**: Allow reconnection to same game
2. **Spectator Mode**: Read-only connections
3. **Replay System**: Record and replay games
4. **Tournaments**: Multiplayer competitive modes
5. **Leaderboards**: Global score tracking
