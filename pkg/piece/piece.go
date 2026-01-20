package piece

import (
	"fmt"
)

// Type represents a Tetris piece type
type Type int

const (
	TypeI Type = iota
	TypeO
	TypeT
	TypeS
	TypeZ
	TypeJ
	TypeL
)

// Color represents the color of a piece
type Color string

const (
	ColorCyan   Color = "#00FFFF" // I
	ColorYellow Color = "#FFFF00" // O
	ColorPurple Color = "#800080" // T
	ColorGreen  Color = "#00FF00" // S
	ColorRed    Color = "#FF0000" // Z
	ColorBlue   Color = "#0000FF" // J
	ColorOrange Color = "#FFA500" // L
	ColorEmpty  Color = ""
)

// Piece represents a Tetris piece with its type, position, and rotation
type Piece struct {
	Type     Type
	Color    Color
	X        int
	Y        int
	Rotation int // 0-3, representing 0°, 90°, 180°, 270° clockwise
}

// Shape defines the 2D grid of a piece
// Each cell is 0 (empty) or 1 (filled)
type Shape [][]int

// shapes defines all 7 Tetris pieces in their base rotation (0°)
var shapes = map[Type]Shape{
	TypeI: {{1, 1, 1, 1}},
	TypeO: {{1, 1}, {1, 1}},
	TypeT: {{0, 1, 0}, {1, 1, 1}},
	TypeS: {{0, 1, 1}, {1, 1, 0}},
	TypeZ: {{1, 1, 0}, {0, 1, 1}},
	TypeJ: {{1, 0, 0}, {1, 1, 1}},
	TypeL: {{0, 0, 1}, {1, 1, 1}},
}

// colors maps piece types to their colors
var colors = map[Type]Color{
	TypeI: ColorCyan,
	TypeO: ColorYellow,
	TypeT: ColorPurple,
	TypeS: ColorGreen,
	TypeZ: ColorRed,
	TypeJ: ColorBlue,
	TypeL: ColorOrange,
}

// New creates a new piece of the given type
func New(t Type) *Piece {
	return &Piece{
		Type:     t,
		Color:    colors[t],
		X:        3, // Start in the middle of a 10-wide board
		Y:        0,
		Rotation: 0,
	}
}

// GetShape returns the shape of the piece in its current rotation
func (p *Piece) GetShape() Shape {
	baseShape := shapes[p.Type]
	return rotate(baseShape, p.Rotation)
}

// rotate rotates a shape by the given number of 90° clockwise rotations
func rotate(shape Shape, times int) Shape {
	result := shape
	for i := 0; i < times; i++ {
		result = rotate90(result)
	}
	return result
}

// rotate90 rotates a shape 90° clockwise
func rotate90(shape Shape) Shape {
	rows := len(shape)
	cols := len(shape[0])
	rotated := make(Shape, cols)

	for i := range rotated {
		rotated[i] = make([]int, rows)
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			rotated[c][rows-1-r] = shape[r][c]
		}
	}

	return rotated
}

// Rotate rotates the piece 90° clockwise
// Returns true if successful, false if blocked
func (p *Piece) Rotate(checkCollision func(x, y int, shape Shape) bool) bool {
	if p.Type == TypeO {
		// O piece doesn't change shape when rotated
		return true
	}

	newRotation := (p.Rotation + 1) % 4
	newShape := rotate(shapes[p.Type], newRotation)

	// Try basic rotation
	if !checkCollision(p.X, p.Y, newShape) {
		p.Rotation = newRotation
		return true
	}

	// Try wall kicks
	kicks := getWallKicks(p.Type, newRotation)
	for _, kick := range kicks {
		newX := p.X + kick.dx
		newY := p.Y + kick.dy
		if !checkCollision(newX, newY, newShape) {
			p.X = newX
			p.Y = newY
			p.Rotation = newRotation
			return true
		}
	}

	return false
}

// wallKick represents a position adjustment for rotation
type wallKick struct {
	dx, dy int
}

// getWallKicks returns wall kick offsets for a piece type and rotation
func getWallKicks(t Type, rotation int) []wallKick {
	if t == TypeI {
		// I piece gets larger kicks (1-2 cells)
		return []wallKick{
			{-1, 0}, {1, 0}, {-2, 0}, {2, 0},
			{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
		}
	}

	// Other pieces get smaller kicks (1 cell)
	return []wallKick{
		{-1, 0}, {1, 0}, {0, -1},
	}
}

// MoveLeft attempts to move the piece left by one cell
// Returns true if successful
func (p *Piece) MoveLeft(checkCollision func(x, y int, shape Shape) bool) bool {
	shape := p.GetShape()
	if !checkCollision(p.X-1, p.Y, shape) {
		p.X--
		return true
	}
	return false
}

// MoveRight attempts to move the piece right by one cell
// Returns true if successful
func (p *Piece) MoveRight(checkCollision func(x, y int, shape Shape) bool) bool {
	shape := p.GetShape()
	if !checkCollision(p.X+1, p.Y, shape) {
		p.X++
		return true
	}
	return false
}

// MoveDown attempts to move the piece down by one cell
// Returns true if successful
func (p *Piece) MoveDown(checkCollision func(x, y int, shape Shape) bool) bool {
	shape := p.GetShape()
	if !checkCollision(p.X, p.Y+1, shape) {
		p.Y++
		return true
	}
	return false
}

// HardDrop drops the piece to the lowest possible position
// Returns the number of cells dropped
func (p *Piece) HardDrop(checkCollision func(x, y int, shape Shape) bool) int {
	dropDistance := 0
	shape := p.GetShape()

	for !checkCollision(p.X, p.Y+1, shape) {
		p.Y++
		dropDistance++
	}

	return dropDistance
}

// String returns a string representation of the piece
func (p *Piece) String() string {
	return fmt.Sprintf("Piece{Type: %v, Pos: (%d, %d), Rot: %d}", p.Type, p.X, p.Y, p.Rotation)
}

// String returns the string representation of a piece type
func (t Type) String() string {
	names := map[Type]string{
		TypeI: "I",
		TypeO: "O",
		TypeT: "T",
		TypeS: "S",
		TypeZ: "Z",
		TypeJ: "J",
		TypeL: "L",
	}
	return names[t]
}

// Width returns the width of a shape
func (s Shape) Width() int {
	if len(s) == 0 {
		return 0
	}
	return len(s[0])
}

// Height returns the height of a shape
func (s Shape) Height() int {
	return len(s)
}
