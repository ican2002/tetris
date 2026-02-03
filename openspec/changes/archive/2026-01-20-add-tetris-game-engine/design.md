# Design: Tetris Core Game Engine

## Overview

This document describes the architectural design of the Tetris core game engine, which implements the fundamental game logic for a production-grade Tetris game. The engine is designed to be independent of any frontend, making it suitable for multiple UI implementations (terminal, web, etc.).

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         Game                                │
│  ┌──────────┐  ┌────────────┐  ┌────────────────────────┐  │
│  │  Board   │  │  Piece     │  │     Generator          │  │
│  │          │  │            │  │  (7-bag randomization) │  │
│  │ - cells  │  │ - Type     │  │                        │  │
│  │ - width  │  │ - Color    │  │ - bag: []Type          │  │
│  │ - height │  │ - X, Y     │  │ - rand: *rand.Rand     │  │
│  │          │  │ - Rotation │  │                        │  │
│  └──────────┘  └────────────┘  └────────────────────────┘  │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Game State                         │  │
│  │  - State: Playing/Paused/GameOver                    │  │
│  │  - Score, Level, Lines                               │  │
│  │  - dropInterval: time.Duration                       │  │
│  │  - mu: sync.RWMutex (thread safety)                 │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Package Structure

```
pkg/
├── game/
│   ├── game.go          # Main Game engine
│   └── game_test.go
├── board/
│   ├── board.go         # Board representation and collision detection
│   └── board_test.go
└── piece/
    ├── piece.go         # Piece types and movement
    ├── generator.go     # 7-bag randomizer
    └── piece_test.go
```

## Core Components

### 1. Piece System (`pkg/piece/`)

**Purpose**: Encapsulate piece type, shape, rotation, and movement logic.

**Key Types**:
```go
type Type int          // I, O, T, S, Z, J, L
type Color string      // Hex color codes
type Shape [][]int     // 2D grid representation

type Piece struct {
    Type     Type
    Color    Color
    X, Y     int          // Position on board
    Rotation int          // 0-3 (0°, 90°, 180°, 270°)
}
```

**Design Decisions**:

1. **Shape Representation**: Using 2D arrays (`[][]int`) with 0/1 values allows for:
   - Easy rotation algorithms
   - Clear collision detection
   - Simple visualization

2. **Rotation System**:
   - Pieces rotate clockwise 90° per increment
   - O piece (2x2) is exempt from rotation (symmetry)
   - Wall kicks handle edge cases (rotation near walls/pieces)

3. **Wall Kick Implementation**:
   ```go
   // I-piece: larger kicks (1-2 cells)
   kicks = []wallKick{{-1,0}, {1,0}, {-2,0}, {2,0}, ...}

   // Others: standard kicks (1 cell)
   kicks = []wallKick{{-1,0}, {1,0}, {0,-1}}
   ```

### 2. Board System (`pkg/board/`)

**Purpose**: Manage the game grid, collision detection, and line clearing.

**Key Types**:
```go
const Width, Height = 10, 20

type Cell struct {
    Color Color
    Empty bool
}

type Board struct {
    cells [Height][Width]Cell
}
```

**Design Decisions**:

1. **Fixed Size Array**: Using `[Height][Width]Cell` provides:
   - Compile-time size guarantees
   - Better cache locality
   - Zero allocations during gameplay

2. **Collision Detection Strategy**:
   ```go
   func (b *Board) CheckCollision(x, y int, shape Shape) bool {
       // Check each cell of the piece shape
       // Return true if any cell is:
       //   - Out of bounds
       //   - Overlapping an occupied cell
   }
   ```
   - Collision function is passed to piece operations
   - Allows pieces to test moves before committing

3. **Line Clearing Algorithm**:
   - Scan from bottom to top
   - When a complete line is found:
     - Remove it
     - Shift all rows above down
     - Re-check the current row index (content shifted down)

### 3. Generator (`pkg/piece/generator.go`)

**Purpose**: Ensure fair piece distribution using 7-bag randomization.

**Algorithm**:
```
1. Create a bag with all 7 piece types
2. Shuffle the bag
3. Return pieces in order
4. When empty, refill and reshuffle
```

**Benefits**:
- Eliminates long droughts of specific pieces
- Predictable distribution over 7-piece windows
- Still feels random to players

### 4. Game Engine (`pkg/game/`)

**Purpose**: Orchestrate all game systems and manage state.

**Key Types**:
```go
type State int  // Playing, Paused, GameOver

type Game struct {
    board        *board.Board
    generator    *piece.Generator
    current      *piece.Piece
    next         *piece.Piece
    state        State
    score        int
    level        int
    lines        int
    dropInterval time.Duration
    lastDrop     time.Time
    mu           sync.RWMutex
}
```

**Concurrency Design**:

1. **Thread Safety**: `sync.RWMutex` protects all state mutations
   - Write locks for state changes (move, rotate, score)
   - Read locks for state queries (GetScore, GetBoard, etc.)

2. **Game Loop**:
   ```go
   func (g *Game) Update() bool {
       // Called periodically (e.g., every dropInterval)
       // Attempts to move piece down
       // If blocked: locks piece, clears lines, spawns new piece
   }
   ```

3. **State Snapshot**: For serialization (WebSocket transmission):
   ```go
   func (g *Game) GetStateSnapshot() (board, current, next, ...)
   ```
   - Creates deep copies to avoid shared references
   - Ensures current and next pieces are never the same object

## Scoring System

**Scoring Table**:
| Lines Cleared | Points       |
|---------------|--------------|
| 1             | 100 × level  |
| 2             | 300 × level  |
| 3             | 500 × level  |
| 4 (Tetris)    | 800 × level  |

**Hard Drop Bonus**: `dropDistance × level`

**Level Progression**:
- Level increases every 10 lines
- Speed formula: `max(100ms, 1000ms - (level-1) × 100ms)`
- Level 1: 1000ms per drop
- Level 10: 100ms per drop (maximum speed)

## Game Flow

```
┌─────────────┐
│ Game Start  │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Spawn First Piece   │
│ (check game over)   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐    ┌──────────────┐
│ Player Input Loop   │◄───┤ Auto Drop    │
│ - Move Left/Right   │    │ (periodic)   │
│ - Soft Drop         │    └──────────────┘
│ - Rotate            │
│ - Hard Drop         │
└──────┬──────────────┘
       │ Piece can't move down
       ▼
┌─────────────────────┐
│ Lock Piece          │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Clear Lines         │───┐
│ Update Score        │   │ 0 lines cleared
│ Check Level Up      │   └──────────────────┐
└──────┬──────────────┘                      │
       │                                     │
       │ 1+ lines cleared                    │
       ▼                                     │
┌─────────────────────┐                      │
│ Spawn Next Piece    │                      │
│ (check game over)   │                      │
└──────┬──────────────┘                      │
       │                                     │
       │ Not game over                       │ Game Over
       ▼                                     ▼
┌─────────────────────┐            ┌──────────────────┐
│ Back to Input Loop  │            │ State = GameOver │
└─────────────────────┘            │ Show Final Score │
                                  └──────────────────┘
```

## Testing Strategy

### Unit Tests

1. **Piece Tests** (`piece_test.go`):
   - Shape definitions match Tetris standards
   - Rotation produces correct orientations
   - Wall kicks work as expected
   - Movement respects boundaries

2. **Board Tests** (`board_test.go`):
   - Collision detection accuracy
   - Line clearing with various patterns
   - Boundary checks

3. **Generator Tests** (`generator_test.go`):
   - 7-bag distribution fairness
   - Deterministic with seed (for testing)
   - No duplicate pieces in one bag

4. **Game Tests** (`game_test.go`):
   - Score calculation
   - Level progression
   - State transitions
   - Thread safety (concurrent operations)

### Integration Tests

- Full game scenarios
- Edge cases (game over, maximum score, etc.)

## Performance Considerations

1. **Memory**: Zero allocations during core gameplay loop
   - Fixed-size arrays
   - Pre-allocated shapes
   - Object reuse (current/next piece cycling)

2. **CPU**:
   - Collision detection: O(piece cells) = O(4)
   - Line clearing: O(board height × width) = O(200)
   - Rotation: O(piece cells × rotation count) = O(4)

3. **Concurrency**:
   - Read-write locks minimize contention
   - State snapshot copies are optimized (only copy necessary data)

## Future Enhancements (Out of Scope)

- Hold piece functionality
- Ghost piece (landing preview)
- T-Spin detection and bonuses
- Combo system
- Multiplayer modes
- Replay system
