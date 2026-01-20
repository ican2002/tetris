package board

import (
	"github.com/ican2002/tetris/pkg/piece"
)

const (
	Width  = 10
	Height = 20
)

// Cell represents a single cell on the board
type Cell struct {
	Color piece.Color `json:"color,omitempty"`
	Empty bool        `json:"empty"`
}

// Board represents the Tetris game board
type Board struct {
	cells [Height][Width]Cell
}

// New creates a new empty board
func New() *Board {
	b := &Board{}
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			b.cells[y][x] = Cell{Empty: true}
		}
	}
	return b
}

// GetCell returns the cell at the given position
// Returns error if position is out of bounds
func (b *Board) GetCell(x, y int) (Cell, error) {
	if !b.isValidPosition(x, y) {
		return Cell{}, &OutOfBoundsError{X: x, Y: y}
	}
	return b.cells[y][x], nil
}

// SetCell sets the cell at the given position
func (b *Board) SetCell(x, y int, color piece.Color) error {
	if !b.isValidPosition(x, y) {
		return &OutOfBoundsError{X: x, Y: y}
	}
	b.cells[y][x] = Cell{Color: color, Empty: false}
	return nil
}

// IsEmpty returns true if the cell at (x, y) is empty
func (b *Board) IsEmpty(x, y int) bool {
	if !b.isValidPosition(x, y) {
		return false
	}
	return b.cells[y][x].Empty
}

// IsOccupied returns true if the cell at (x, y) is occupied
func (b *Board) IsOccupied(x, y int) bool {
	return !b.IsEmpty(x, y)
}

// IsValidPosition checks if a position is within the board boundaries
func (b *Board) isValidPosition(x, y int) bool {
	return x >= 0 && x < Width && y >= 0 && y < Height
}

// CheckCollision checks if placing a piece at (x, y) would cause a collision
// Returns true if there is a collision (invalid or occupied)
func (b *Board) CheckCollision(x int, y int, shape piece.Shape) bool {
	for r := 0; r < shape.Height(); r++ {
		for c := 0; c < shape.Width(); c++ {
			if shape[r][c] == 1 {
				boardX := x + c
				boardY := y + r

				// Check boundaries
				if !b.isValidPosition(boardX, boardY) {
					return true
				}

				// Check if occupied
				if b.IsOccupied(boardX, boardY) {
					return true
				}
			}
		}
	}
	return false
}

// LockPiece locks a piece onto the board
func (b *Board) LockPiece(p *piece.Piece) error {
	shape := p.GetShape()
	for r := 0; r < shape.Height(); r++ {
		for c := 0; c < shape.Width(); c++ {
			if shape[r][c] == 1 {
				boardX := p.X + c
				boardY := p.Y + r
				if err := b.SetCell(boardX, boardY, p.Color); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ClearLines clears complete rows and returns the number of lines cleared
func (b *Board) ClearLines() int {
	linesCleared := 0

	// Find complete lines from bottom to top
	for y := Height - 1; y >= 0; y-- {
		if b.isLineComplete(y) {
			b.removeLine(y)
			linesCleared++
			y++ // Recheck this row as everything shifted down
		}
	}

	return linesCleared
}

// isLineComplete checks if a row is completely filled
func (b *Board) isLineComplete(y int) bool {
	for x := 0; x < Width; x++ {
		if b.IsEmpty(x, y) {
			return false
		}
	}
	return true
}

// removeLine removes a row and shifts all rows above down
func (b *Board) removeLine(y int) {
	// Shift all rows above down
	for row := y; row > 0; row-- {
		for x := 0; x < Width; x++ {
			b.cells[row][x] = b.cells[row-1][x]
		}
	}

	// Clear the top row
	for x := 0; x < Width; x++ {
		b.cells[0][x] = Cell{Empty: true}
	}
}

// GetCells returns a 2D array of all cells
func (b *Board) GetCells() [Height][Width]Cell {
	return b.cells
}

// OutOfBoundsError represents an error for out of bounds access
type OutOfBoundsError struct {
	X int
	Y int
}

func (e *OutOfBoundsError) Error() string {
	return "position out of bounds"
}

// Clone creates a deep copy of the board
func (b *Board) Clone() *Board {
	newBoard := New()
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			newBoard.cells[y][x] = b.cells[y][x]
		}
	}
	return newBoard
}
