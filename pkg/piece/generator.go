package piece

import (
	"math/rand"
	"time"
)

// allPieceTypes is a slice of all 7 Tetris piece types
var allPieceTypes = []Type{TypeI, TypeO, TypeT, TypeS, TypeZ, TypeJ, TypeL}

// Generator generates Tetris pieces using the 7-bag randomization algorithm
type Generator struct {
	bag []Type
	rnd *rand.Rand
}

// NewGenerator creates a new piece generator
func NewGenerator() *Generator {
	return &Generator{
		bag: make([]Type, 0, 7),
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewGeneratorWithSeed creates a new piece generator with a specific seed (for testing)
func NewGeneratorWithSeed(seed int64) *Generator {
	return &Generator{
		bag: make([]Type, 0, 7),
		rnd: rand.New(rand.NewSource(seed)),
	}
}

// Next returns the next piece from the bag
// If the bag is empty, it refills with a new shuffled bag of all 7 pieces
func (g *Generator) Next() *Piece {
	if len(g.bag) == 0 {
		g.refillBag()
	}

	// Pop the first piece from the bag
	pieceType := g.bag[0]
	g.bag = g.bag[1:]

	return New(pieceType)
}

// Peek returns the next piece without removing it from the bag
// If the bag is empty, it refills first
func (g *Generator) Peek() *Piece {
	if len(g.bag) == 0 {
		g.refillBag()
	}

	if len(g.bag) > 0 {
		return New(g.bag[0])
	}

	return New(TypeI) // fallback
}

// refillBag creates a new bag with all 7 pieces and shuffles it
func (g *Generator) refillBag() {
	// Create a new bag with all 7 piece types
	g.bag = make([]Type, 7)
	copy(g.bag, allPieceTypes)

	// Shuffle using Fisher-Yates algorithm
	g.shuffle()
}

// shuffle shuffles the bag using Fisher-Yates algorithm
func (g *Generator) shuffle() {
	n := len(g.bag)
	for i := n - 1; i > 0; i-- {
		j := g.rnd.Intn(i + 1)
		g.bag[i], g.bag[j] = g.bag[j], g.bag[i]
	}
}

// BagSize returns the current size of the bag
func (g *Generator) BagSize() int {
	return len(g.bag)
}

// Remaining returns the remaining pieces in the bag
func (g *Generator) Remaining() []Type {
	result := make([]Type, len(g.bag))
	copy(result, g.bag)
	return result
}
