package main

import (
	"fmt"
	"time"

	"github.com/ican2002/tetris/pkg/game"
)

func main() {
	fmt.Println("=== Tetris Game Engine Demo ===")
	fmt.Println()

	// Create a new game
	g := game.New()

	fmt.Printf("Game initialized!\n")
	fmt.Printf("State: %s\n", g.GetState())
	fmt.Printf("Level: %d\n", g.GetLevel())
	fmt.Printf("Score: %d\n", g.GetScore())
	fmt.Printf("Lines: %d\n", g.GetLines())
	fmt.Printf("Drop Interval: %v\n", g.GetDropInterval())
	fmt.Printf("Current Piece: %s\n", g.GetCurrentPiece().Type)
	fmt.Printf("Next Piece: %s\n", g.GetNextPiece().Type)
	fmt.Println()

	// Simulate some moves
	fmt.Println("Simulating moves...")

	// Move right 3 times
	for i := 0; i < 3; i++ {
		g.MoveRight()
	}
	fmt.Printf("Moved piece right 3 times, now at X=%d\n", g.GetCurrentPiece().X)

	// Rotate
	g.Rotate()
	fmt.Printf("Rotated piece, rotation=%d\n", g.GetCurrentPiece().Rotation)

	// Hard drop
	distance := g.HardDrop()
	fmt.Printf("Hard dropped %d cells\n", distance)
	fmt.Printf("Current Piece: %s at Y=%d\n", g.GetCurrentPiece().Type, g.GetCurrentPiece().Y)
	fmt.Printf("Score: %d\n", g.GetScore())
	fmt.Println()

	// Simulate game loop
	fmt.Println("Running game loop for 5 ticks...")

	for i := 0; i < 5; i++ {
		time.Sleep(1100 * time.Millisecond)
		updated := g.Update()
		if updated {
			fmt.Printf("Tick %d: Piece moved down, now at Y=%d\n", i+1, g.GetCurrentPiece().Y)
		}
	}

	fmt.Println()

	// Pause game
	g.Pause()
	fmt.Printf("Game paused. State: %s\n", g.GetState())

	// Resume
	g.Resume()
	fmt.Printf("Game resumed. State: %s\n", g.GetState())

	fmt.Println()

	// Display final stats
	fmt.Println("=== Final Game State ===")
	fmt.Printf("State: %s\n", g.GetState())
	fmt.Printf("Level: %d\n", g.GetLevel())
	fmt.Printf("Score: %d\n", g.GetScore())
	fmt.Printf("Lines: %d\n", g.GetLines())
	fmt.Printf("Current Piece: %s at (%d, %d)\n", g.GetCurrentPiece().Type, g.GetCurrentPiece().X, g.GetCurrentPiece().Y)
	fmt.Printf("Next Piece: %s\n", g.GetNextPiece().Type)

	// Get complete game state
	state := g.GetGameState()
	fmt.Printf("\nComplete State: %+v\n", state)

	fmt.Println()
	fmt.Println("Demo completed!")
}
