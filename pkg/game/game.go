package game

import (
	"sync"
	"time"

	"github.com/ican2002/tetris/pkg/board"
	"github.com/ican2002/tetris/pkg/piece"
)

// State represents the current game state
type State int

const (
	StatePlaying State = iota
	StatePaused
	StateGameOver
)

// String returns the string representation of the game state
func (s State) String() string {
	names := map[State]string{
		StatePlaying:  "playing",
		StatePaused:   "paused",
		StateGameOver: "gameover",
	}
	return names[s]
}

// Game represents the Tetris game engine
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
	mu           sync.RWMutex // Protects game state during concurrent access
}

// New creates a new game
func New() *Game {
	g := &Game{
		board:        board.New(),
		generator:    piece.NewGenerator(),
		state:        StatePlaying,
		score:        0,
		level:        1,
		lines:        0,
		dropInterval: calculateDropInterval(1),
		lastDrop:     time.Now(),
	}

	g.spawnPiece()
	g.prepareNext()

	return g
}

// NewWithSeed creates a new game with a specific seed (for testing)
func NewWithSeed(seed int64) *Game {
	g := &Game{
		board:        board.New(),
		generator:    piece.NewGeneratorWithSeed(seed),
		state:        StatePlaying,
		score:        0,
		level:        1,
		lines:        0,
		dropInterval: calculateDropInterval(1),
		lastDrop:     time.Now(),
	}

	g.spawnPiece()
	g.prepareNext()

	return g
}

// spawnPiece creates a new current piece
func (g *Game) spawnPiece() {
	// Get the next piece
	if g.next != nil {
		g.current = g.next
	} else {
		g.current = g.generator.Next()
	}

	// Check for game over
	if g.board.CheckCollision(g.current.X, g.current.Y, g.current.GetShape()) {
		g.state = StateGameOver
	}
}

// prepareNext prepares the next piece
func (g *Game) prepareNext() {
	g.next = g.generator.Next()
}

// MoveLeft attempts to move the current piece left
func (g *Game) MoveLeft() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StatePlaying {
		return false
	}

	collision := func(x, y int, shape piece.Shape) bool {
		return g.board.CheckCollision(x, y, shape)
	}

	return g.current.MoveLeft(collision)
}

// MoveRight attempts to move the current piece right
func (g *Game) MoveRight() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StatePlaying {
		return false
	}

	collision := func(x, y int, shape piece.Shape) bool {
		return g.board.CheckCollision(x, y, shape)
	}

	return g.current.MoveRight(collision)
}

// MoveDown attempts to move the current piece down (soft drop)
func (g *Game) MoveDown() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StatePlaying {
		return false
	}

	collision := func(x, y int, shape piece.Shape) bool {
		return g.board.CheckCollision(x, y, shape)
	}

	success := g.current.MoveDown(collision)
	if !success {
		// Piece locked, spawn new piece
		g.lockAndSpawnLocked()
	}

	return success
}

// HardDrop drops the piece to the lowest position
func (g *Game) HardDrop() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StatePlaying {
		return 0
	}

	collision := func(x, y int, shape piece.Shape) bool {
		return g.board.CheckCollision(x, y, shape)
	}

	dropDistance := g.current.HardDrop(collision)

	// Award hard drop bonus points
	g.score += dropDistance * g.level

	// Lock and spawn new piece
	g.lockAndSpawnLocked()

	return dropDistance
}

// Rotate attempts to rotate the current piece
func (g *Game) Rotate() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StatePlaying {
		return false
	}

	collision := func(x, y int, shape piece.Shape) bool {
		return g.board.CheckCollision(x, y, shape)
	}

	return g.current.Rotate(collision)
}

// lockAndSpawn locks the current piece and spawns a new one
// Note: This method assumes mu is NOT held and will lock it itself
func (g *Game) lockAndSpawn() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lockAndSpawnLocked()
}

// lockAndSpawnLocked is the internal implementation that assumes mu is already held
func (g *Game) lockAndSpawnLocked() {
	// Lock the piece
	g.board.LockPiece(g.current)

	// Clear lines and update score
	linesCleared := g.board.ClearLines()
	g.updateScore(linesCleared)

	// Spawn new piece
	g.spawnPiece()
	g.prepareNext()
}

// updateScore updates the score based on lines cleared
func (g *Game) updateScore(linesCleared int) {
	if linesCleared == 0 {
		return
	}

	// Update score based on lines cleared
	scoreMultiplier := map[int]int{
		1: 100,
		2: 300,
		3: 500,
		4: 800,
	}

	g.score += scoreMultiplier[linesCleared] * g.level

	// Update lines
	g.lines += linesCleared

	// Update level every 10 lines
	newLevel := (g.lines / 10) + 1
	if newLevel > g.level {
		g.level = newLevel
		g.dropInterval = calculateDropInterval(g.level)
	}
}

// calculateDropInterval calculates the drop interval for a given level
func calculateDropInterval(level int) time.Duration {
	// Formula: max(100ms, 1000ms - (level-1) * 100ms)
	// Level 1: 1000ms, Level 10: 100ms
	ms := 1000 - (level-1)*100
	if ms < 100 {
		ms = 100
	}
	return time.Duration(ms) * time.Millisecond
}

// Pause pauses the game
func (g *Game) Pause() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == StatePlaying {
		g.state = StatePaused
	}
}

// Resume resumes the game
func (g *Game) Resume() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == StatePaused {
		g.state = StatePlaying
		g.lastDrop = time.Now()
	}
}

// TogglePause toggles the pause state
func (g *Game) TogglePause() {
	if g.state == StatePlaying {
		g.Pause()
	} else if g.state == StatePaused {
		g.Resume()
	}
}

// Update updates the game state (should be called in a loop)
func (g *Game) Update() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StatePlaying {
		return false
	}

	now := time.Now()
	if now.Sub(g.lastDrop) >= g.dropInterval {
		g.lastDrop = now

		collision := func(x, y int, shape piece.Shape) bool {
			return g.board.CheckCollision(x, y, shape)
		}

		// Try to move down
		if !g.current.MoveDown(collision) {
			// Piece locked, spawn new piece
			g.lockAndSpawnLocked()
		}

		return true
	}

	return false
}

// GetState returns the current game state
func (g *Game) GetState() State {
	return g.state
}

// GetBoard returns the game board
func (g *Game) GetBoard() *board.Board {
	return g.board
}

// GetCurrentPiece returns the current piece
func (g *Game) GetCurrentPiece() *piece.Piece {
	return g.current
}

// GetNextPiece returns the next piece
func (g *Game) GetNextPiece() *piece.Piece {
	return g.next
}

// GetScore returns the current score
func (g *Game) GetScore() int {
	return g.score
}

// GetLevel returns the current level
func (g *Game) GetLevel() int {
	return g.level
}

// GetLines returns the number of lines cleared
func (g *Game) GetLines() int {
	return g.lines
}

// GetDropInterval returns the current drop interval
func (g *Game) GetDropInterval() time.Duration {
	return g.dropInterval
}

// IsGameOver returns true if the game is over
func (g *Game) IsGameOver() bool {
	return g.state == StateGameOver
}

// IsPaused returns true if the game is paused
func (g *Game) IsPaused() bool {
	return g.state == StatePaused
}

// IsPlaying returns true if the game is playing
func (g *Game) IsPlaying() bool {
	return g.state == StatePlaying
}

// GameState represents a snapshot of the game state (for serialization)
type GameState struct {
	Board        *board.Board  `json:"board"`
	CurrentPiece *piece.Piece  `json:"current_piece"`
	NextPiece    *piece.Piece  `json:"next_piece"`
	State        State         `json:"state"`
	Score        int           `json:"score"`
	Level        int           `json:"level"`
	Lines        int           `json:"lines"`
	DropInterval time.Duration `json:"drop_interval"`
}

// GetStateSnapshot returns a consistent snapshot of the game state for serialization
// This ensures that current and next pieces are never the same object
func (g *Game) GetStateSnapshot() (boardCopy [][]string, current *piece.Piece, next *piece.Piece, stateStr string, score, level, lines int, dropInterval time.Duration) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Clone board
	boardCopy = make([][]string, board.Height)
	b := g.board
	for y := 0; y < board.Height; y++ {
		boardCopy[y] = make([]string, board.Width)
		for x := 0; x < board.Width; x++ {
			cell, _ := b.GetCell(x, y)
			if cell.Empty {
				boardCopy[y][x] = ""
			} else {
				boardCopy[y][x] = string(cell.Color)
			}
		}
	}

	// Clone pieces to avoid shared references
	if g.current != nil {
		current = &piece.Piece{
			Type:     g.current.Type,
			Color:    g.current.Color,
			X:        g.current.X,
			Y:        g.current.Y,
			Rotation: g.current.Rotation,
		}
	}

	if g.next != nil {
		next = &piece.Piece{
			Type:     g.next.Type,
			Color:    g.next.Color,
			X:        g.next.X,
			Y:        g.next.Y,
			Rotation: g.next.Rotation,
		}
	}

	stateStr = g.state.String()
	score = g.score
	level = g.level
	lines = g.lines
	dropInterval = g.dropInterval

	return
}

// GetGameState returns a complete snapshot of the game state
func (g *Game) GetGameState() GameState {
	return GameState{
		Board:        g.board.Clone(),
		CurrentPiece: g.current,
		NextPiece:    g.next,
		State:        g.state,
		Score:        g.score,
		Level:        g.level,
		Lines:        g.lines,
		DropInterval: g.dropInterval,
	}
}
