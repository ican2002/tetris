# Design: Terminal Frontend

## Overview

This document describes the architectural design of the terminal-based user interface for Tetris. The terminal frontend provides a complete gameplay experience using text-based rendering, keyboard input, and real-time WebSocket communication with the game server.

**Note**: The project also includes a Web-based UI client (`test-client.html`) that provides similar functionality through a browser interface. See the WebSocket Communication Layer design for details on the Web UI client.

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Application                        │
│  ┌──────────────┐  ┌───────────────┐  ┌──────────────────────┐ │
│  │    TUI       │  │  WSClient     │  │    App Logic         │ │
│  │              │  │               │  │                      │ │
│  │ - Screen     │  │ - conn        │  │ - Main loop          │ │
│  │ - Renderer   │  │ - send/recv   │  │ - State management   │ │
│  │ - Input      │  │ - callbacks   │  │ - Error handling     │ │
│  └──────────────┘  └───────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │                    │                       │
         ▼                    ▼                       ▼
┌─────────────┐      ┌──────────────┐      ┌──────────────────┐
│   tcell     │      │   WebSocket  │      │   Game States    │
│  Terminal   │      │   Server     │      │  (Welcome/Play/   │
│   I/O       │      │   (Network)  │      │   Pause/GameOver) │
└─────────────┘      └──────────────┘      └──────────────────┘
```

### Package Structure

```
pkg/
└── tui/
    ├── tui.go         # TUI initialization and core utilities
    ├── draw.go        # Rendering functions (board, info, screens)
    └── draw_test.go

cmd/
└── tetris-client/
    └── main.go        # Application entry point and main loop
```

## Core Components

### 1. TUI Framework (`pkg/tui/tui.go`)

**Purpose**: Provide terminal rendering capabilities and input handling.

**Library**: `github.com/gdamore/tcell/v2`

**Key Features**:
- Cross-platform terminal support (Linux, macOS, Windows)
- Unicode support (for box-drawing characters)
- Color support (256 colors + TrueColor)
- Event-driven input model

**TUI Structure**:
```go
type TUI struct {
    screen   tcell.Screen
    width    int
    height   int
    eventCh  chan tcell.Event
    quitCh   chan struct{}

    // Layout configuration
    boardX, boardY      int
    boardWidth, boardHeight int
    infoX, infoY         int

    running bool
}
```

**Color Mapping**:
```go
var colorMap = map[piece.Color]tcell.Color{
    piece.ColorCyan:   tcell.ColorTeal,
    piece.ColorYellow: tcell.ColorYellow,
    piece.ColorPurple: tcell.ColorPurple,
    piece.ColorGreen:  tcell.ColorGreen,
    piece.ColorRed:    tcell.ColorRed,
    piece.ColorBlue:   tcell.ColorBlue,
    piece.ColorOrange: tcell.ColorOrange,
}
```

**Event Loop**:
```
┌─────────────────────────────────┐
│         Main Loop               │
├─────────────────────────────────┤
│                                 │
│  ┌───────────────────────────┐  │
│  │    Event Pump (goroutine) │  │
│  │  - Polls screen events   │  │
│  │  - Sends to eventCh      │  │
│  └───────────────────────────┘  │
│             │                    │
│             ▼                    │
│  ┌───────────────────────────┐  │
│  │   Event Dispatcher        │  │
│  │  - Key events → commands  │  │
│  │  - Resize → re-layout     │  │
│  └───────────────────────────┘  │
│             │                    │
│             ▼                    │
│  ┌───────────────────────────┐  │
│  │   Renderer                │  │
│  │  - Draw board             │  │
│  │  - Draw info panel        │  │
│  │  - Draw status bar        │  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

### 2. Rendering System (`pkg/tui/draw.go`)

**Purpose**: Draw all visual elements to the terminal.

**Layout**:
```
┌──────────────────────────────────────────────────────────────┐
│  TETRIS - Terminal Edition                                   │
├────────────────────────┬─────────────────────────────────────┤
│                        │                                     │
│   ┌────────────────┐   │  Score:                             │
│   │                │   │  1250                               │
│   │   Game Board   │   │                                     │
│   │    10×20       │   │  Level:                             │
│   │                │   │  3                                  │
│   │                │   │                                     │
│   │                │   │  Lines:                             │
│   │                │   │  27                                 │
│   │                │   │                                     │
│   │                │   │  State:                             │
│   │                │   │  Playing                            │
│   │                │   │                                     │
│   └────────────────┘   │  Next:                              │
│                        │  ▓▓                                 │
│                        │  ▓                                  │
├────────────────────────┴─────────────────────────────────────┤
│ ● Connected                    ESC: Quit | P: Pause | ...    │
└──────────────────────────────────────────────────────────────┘
```

**Rendering Pipeline**:

```
StateMessage received
       │
       ▼
┌────────────────────────┐
│  Clear Screen          │
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  Draw Board            │  1. Create display board
│                        │  2. Copy locked cells
│                        │  3. Overlay current piece
│                        │  4. Render cells:
│                        │     - Filled: 2 spaces with BG color
│                        │     - Empty: 2 dots (··)
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  Draw Info Panel       │  - Score, Level, Lines
│                        │  - Game State (colored)
│                        │  - Next Piece Preview
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  Draw Status Bar       │  - Connection status
│                        │  - Error messages
│                        │  - Control hints
└──────────┬─────────────┘
           │
           ▼
     screen.Show()
```

**Cell Rendering**:
- Each board cell = 2 terminal characters
  - Filled: `  ` (two spaces with background color)
  - Empty: `··` (two middle dots)
- Width: 10 cells × 2 = 20 chars + borders = ~22 chars
- Height: 20 cells + borders = ~22 chars

**Special Screens**:

1. **Welcome Screen**:
   ```
   ┌─────────────────────────────────────┐
   │                                     │
   │        🎮 TETRIS 🎮                 │
   │      Terminal Edition               │
   │                                     │
   │        Controls:                    │
   │          ⬆️  Arrow Up - Rotate       │
   │          ⬇️  Arrow Down - Soft Drop  │
   │          ...                        │
   │                                     │
   │     Press any key to connect...     │
   │                                     │
   └─────────────────────────────────────┘
   ```

2. **Game Over Screen**:
   ```
   ┌─────────────────────────────────────┐
   │                                     │
   │          GAME OVER                  │
   │      Final Score: 1250              │
   │                                     │
   │          Level: 3                   │
   │          Lines: 27                  │
   │                                     │
   │      Press R to restart             │
   │      Press Q to quit...             │
   │                                     │
   └─────────────────────────────────────┘
   ```

### 3. WebSocket Client Integration (`pkg/wsclient/`)

**Purpose**: Communicate with the game server.

**Connection Flow**:
```
Application Start
       │
       ▼
┌────────────────────────┐
│  Create WSClient       │
│  - Set callbacks       │
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  client.Connect()      │
│  - WebSocket handshake │
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  onConnected callback  │
│  - Transition to       │
│    Play state          │
└────────────────────────┘
```

**Message Handling**:

**Send (Input → Server)**:
```go
switch key {
case KeyLeft:
    msg := ControlMessage{Type: "move_left"}
    client.Send(json.Marshal(msg))
case KeyUp:
    msg := ControlMessage{Type: "rotate"}
    client.Send(json.Marshal(msg))
// ... etc
}
```

**Receive (Server → Display)**:
```go
client.SetOnStateChange(func(data []byte) {
    var state StateMessage
    json.Unmarshal(data, &state)
    // Redraw screen with new state
})
```

**Auto-Reconnect**:
- Automatic reconnection on connection loss
- Up to 5 retry attempts with 3-second delay
- Shows "Reconnecting..." message to user

### 4. Input Handling

**Key Bindings**:

| Key | Action | Command |
|-----|--------|---------|
| ← | Move Left | `move_left` |
| → | Move Right | `move_right` |
| ↓ | Soft Drop | `move_down` |
| ↑ | Rotate | `rotate` |
| Space | Hard Drop | `hard_drop` |
| P | Pause/Resume | `toggle_pause` |
| Q / ESC | Quit | (local) |
| R (Game Over) | Restart | `restart` |

**Input Processing**:
```
User presses key
       │
       ▼
┌────────────────────────┐
│  PollEvent()           │  (or PollEventWithTimeout)
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  Key type switch       │
└──────────┬─────────────┘
           │
           ├─→ Game control key
           │       │
           │       ▼
           │  ┌──────────────────┐
           │  │  Create command  │
           │  │  Send to server  │
           │  └──────────────────┘
           │
           ├─→ Quit key
           │       │
           │       ▼
           │  ┌──────────────────┐
           │  │  Close client    │
           │  │  Exit program    │
           │  └──────────────────┘
           │
           └─→ Resize event
                   │
                   ▼
              ┌──────────────────┐
              │  Recalculate     │
              │  layout          │
              │  Redraw screen   │
              └──────────────────┘
```

### 5. Application States

**State Machine**:
```
┌──────────┐
│ Welcome  │  Show welcome screen, wait for connection
└────┬─────┘
     │ User presses any key
     ▼
┌──────────┐
│ Connect  │  Connecting to server...
└────┬─────┘
     │ Connection established
     ▼
┌──────────┐
│  Play    │  Main gameplay loop
└────┬─────┘
     │
     ├─→ P pressed ──┐
     │                ▼
     │            ┌────────┐
     │            │ Pause  │  Game paused
     │            └───┬────┘
     │                │ P pressed again
     │                └──────────────────┘
     │
     ├─→ Game over received
     │       │
     │       ▼
     │   ┌──────────┐
     │   │GameOver  │  Show final score
     │   └────┬─────┘
     │        │
     │        ├─→ R pressed → Restart (new game)
     │        └─→ Q pressed → Quit
     │
     └─→ Q/ESC pressed
             │
             ▼
         ┌────────┐
         │  Quit  │  Clean up and exit
         └────────┘
```

## Performance Optimization

### 1. Partial Redraw (Current Implementation)

**Current**: Full screen redraw on each state update
- Simpler implementation
- Negligible performance impact for this use case
- Terminal text rendering is fast enough

**Potential Optimization** (if needed):
```go
// Track dirty regions
dirtyRegions := []Rect{}
// Only redraw changed cells
for _, region := range dirtyRegions {
    redrawRegion(region)
}
```

### 2. Frame Rate Limiting

- Terminal: ~60 FPS max (but realistically 30-60 FPS)
- WebSocket updates: ~5 FPS (200ms interval on server)
- Input: Event-driven (immediate)

### 3. Double Buffering

tcell automatically handles double buffering:
- `screen.Show()` presents the complete frame
- Prevents flickering
- No tearing artifacts

## Responsive Layout

**Minimum Size**:
- Width: 80 chars
- Height: 24 lines

**Size Checking**:
```go
func (t *TUI) CheckMinimumSize() bool {
    w, h := t.screen.Size()
    return w >= 80 && h >= 24
}
```

**Resize Handling**:
```go
switch ev := t.PollEvent().(type) {
case *tcell.EventResize:
    t.UpdateSize()
    t.Redraw()
}
```

## Error Handling

### Connection Errors

```
Connection Failed
       │
       ▼
┌────────────────────────┐
│  Show error message    │
│  in status bar         │
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│  Attempt reconnect     │
│  (up to 5 times)       │
└──────────┬─────────────┘
           │
           ├─→ Success → Continue game
           │
           └─→ Max retries → Show "Disconnected"
                             → Exit on key press
```

### Protocol Errors

```
Invalid message received
       │
       ▼
┌────────────────────────┐
│  Log error             │
│  Continue running      │
└────────────────────────┘
```

## User Experience Features

### 1. Visual Feedback

- **Connection Status**: Green "● Connected" / Red "● Disconnected"
- **Game State Color Coding**:
  - Playing: Green text
  - Paused: Yellow text
  - Game Over: Red text
- **Piece Colors**: Mapped to closest terminal colors

### 2. Keyboard Shortcuts

All primary actions accessible without modifiers:
- Arrow keys for movement
- Space for hard drop
- Single-letter keys for system functions (P, Q, R)

### 3. Status Bar Information

- Connection status
- Current error/warning messages
- Control hints (always visible)

## Testing Strategy

### Unit Tests
- Shape rotation logic
- Color mapping
- Piece preview rendering

### Integration Tests
- Full rendering pipeline
- WebSocket communication
- State transitions

### Manual Testing
- Different terminal sizes
- Different terminal emulators
- Different color schemes
- Keyboard input on different platforms

## Future Enhancements

1. **Visual Enhancements**:
   - Ghost piece (landing preview)
   - Smooth animations (if terminal supports it)
   - Particle effects for line clears

2. **Gameplay Features**:
   - Hold piece display
   - Score multipliers indicator
   - Combo counter

3. **Accessibility**:
   - High contrast mode
   - Larger text mode
   - Sound effects (optional)

4. **Customization**:
   - Custom key bindings
   - Color schemes
   - Layout options

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gdamore/tcell/v2` | ^2.7.0 | Terminal UI |
| `github.com/gorilla/websocket` | ^1.5.0 | WebSocket client |
| `github.com/ican2002/tetris/pkg/...` | local | Game protocol & types |
