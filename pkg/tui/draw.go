package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ican2002/tetris/pkg/piece"
	"github.com/ican2002/tetris/pkg/protocol"
)

// isValidPieceType checks if a piece type is valid (one of the 7 Tetris pieces)
func isValidPieceType(t piece.Type) bool {
	switch t {
	case piece.TypeI, piece.TypeO, piece.TypeT,
		piece.TypeS, piece.TypeZ, piece.TypeJ, piece.TypeL:
		return true
	default:
		return false
	}
}

// DrawBoard draws the Tetris board
func (t *TUI) DrawBoard(x, y int, state *protocol.StateMessage, style tcell.Style) {
	// Create a display board that includes locked pieces and current piece
	displayBoard := make([][]string, 20)
	for row := 0; row < 20; row++ {
		displayBoard[row] = make([]string, 10)
		if row < len(state.Board) {
			for col := 0; col < 10; col++ {
				if col < len(state.Board[row]) {
					displayBoard[row][col] = state.Board[row][col]
				}
			}
		}
	}

	// Overlay the current piece on the display board
	currentPiece := state.CurrentPiece
	// Check if the piece type is valid (TypeI = 0, so we need to check against valid types)
	if isValidPieceType(currentPiece.Type) && currentPiece.Color != "" {
		shape := getPieceShape(currentPiece)
		if shape != nil {
			for row := 0; row < len(shape); row++ {
				for col := 0; col < len(shape[row]); col++ {
					if shape[row][col] == 1 {
						boardY := currentPiece.Y + row
						boardX := currentPiece.X + col
						if boardY >= 0 && boardY < 20 && boardX >= 0 && boardX < 10 {
							displayBoard[boardY][boardX] = string(currentPiece.Color)
						}
					}
				}
			}
		}
	}

	// Draw cells
	for row := 0; row < 20; row++ {
		for col := 0; col < 10; col++ {
			cellX := x + col*2
			cellY := y + row

			colorStr := displayBoard[row][col]
			if colorStr != "" {
				// Filled cell
				cellStyle := style.Background(GetColor(piece.Color(colorStr)))
				t.screen.SetContent(cellX, cellY, ' ', nil, cellStyle)
				t.screen.SetContent(cellX+1, cellY, ' ', nil, cellStyle)
			} else {
				// Empty cell
				dimStyle := style.Dim(true)
				t.screen.SetContent(cellX, cellY, '·', nil, dimStyle)
				t.screen.SetContent(cellX+1, cellY, '·', nil, dimStyle)
			}
		}
	}
}

// DrawInfoPanel draws the information panel
func (t *TUI) DrawInfoPanel(x, y int, state *protocol.StateMessage, style tcell.Style) {
	// Draw information
	line := y + 1
	t.DrawText(x, line, "Score:", style.Bold(true))
	t.DrawText(x, line+1, fmt.Sprintf("%d", state.Score), style)

	line += 3
	t.DrawText(x, line, "Level:", style.Bold(true))
	t.DrawText(x, line+1, fmt.Sprintf("%d", state.Level), style)

	line += 3
	t.DrawText(x, line, "Lines:", style.Bold(true))
	t.DrawText(x, line+1, fmt.Sprintf("%d", state.Lines), style)

	line += 3
	t.DrawText(x, line, "State:", style.Bold(true))
	stateStyle := style
	switch state.State {
	case "playing":
		stateStyle = stateStyle.Foreground(tcell.ColorGreen.TrueColor())
	case "paused":
		stateStyle = stateStyle.Foreground(tcell.ColorYellow.TrueColor())
	case "gameover":
		stateStyle = stateStyle.Foreground(tcell.ColorRed.TrueColor())
	}
	t.DrawText(x, line+1, capitalize(state.State), stateStyle)

	// Draw next piece preview
	line += 3
	t.DrawText(x, line, "Next:", style.Bold(true))
	t.DrawPiecePreview(x, line+1, state.NextPiece, style)
}

// DrawPiecePreview draws a piece preview (4x4 grid)
func (t *TUI) DrawPiecePreview(x, y int, pieceData protocol.PieceData, style tcell.Style) {
	// Clear the preview area
	t.FillRect(x, y, 8, 4, ' ', style)

	// Validate piece data (TypeI = 0, so we can't use "!= 0" check)
	if !isValidPieceType(pieceData.Type) || pieceData.Color == "" {
		// Empty/invalid piece data, show placeholder
		t.DrawText(x+2, y+1, "No piece", style.Dim(true))
		return
	}

	// Get piece shape
	shape := getPieceShape(pieceData)
	if shape == nil {
		// Shape not found, show error
		t.DrawText(x+1, y+1, "Error", style.Dim(true).Foreground(tcell.ColorRed))
		return
	}

	// Calculate offset to center the piece
	offsetX := (4 - len(shape[0])) / 2
	offsetY := (4 - len(shape)) / 2

	// Draw the piece
	for row := 0; row < len(shape); row++ {
		for col := 0; col < len(shape[row]); col++ {
			if shape[row][col] == 1 {
				cellX := x + (col+offsetX)*2
				cellY := y + row + offsetY

				cellStyle := style.Background(GetColor(pieceData.Color))
				t.screen.SetContent(cellX, cellY, ' ', nil, cellStyle)
				t.screen.SetContent(cellX+1, cellY, ' ', nil, cellStyle)
			}
		}
	}
}

// DrawStatusBar draws the status bar at the bottom
func (t *TUI) DrawStatusBar(x, y, width int, message string, connected bool, style tcell.Style) {
	// Draw status bar background
	t.FillRect(x, y, width, 1, ' ', style.Reverse(true))

	// Draw connection status
	statusText := "● Connected"
	statusStyle := style.Foreground(tcell.ColorGreen.TrueColor())
	if !connected {
		statusText = "● Disconnected"
		statusStyle = style.Foreground(tcell.ColorRed.TrueColor())
	}
	t.DrawText(x+2, y, statusText, statusStyle.Reverse(true))

	// Draw message
	if message != "" {
		msgX := x + len(statusText) + 4
		if msgX+len(message) < x+width-2 {
			t.DrawText(msgX, y, message, style.Reverse(true))
		}
	}

	// Draw quit hint
	hintText := "ESC/Ctrl+C/D/Q: Quit | P: Pause | Space: Drop | Arrows: Move"
	hintX := x + width - len(hintText) - 2
	if hintX > x+len(statusText)+4 {
		t.DrawText(hintX, y, hintText, style.Reverse(true).Dim(true))
	}
}

// DrawWelcomeScreen draws the welcome/startup screen
func (t *TUI) DrawWelcomeScreen(style tcell.Style) {
	w, h := t.screen.Size()

	title := "🎮 TETRIS 🎮"
	subtitle := "Terminal Edition"

	// Center the title
	titleX := (w - len(title)) / 2
	titleY := h / 3
	t.DrawText(titleX, titleY, title, style.Bold(true).Foreground(tcell.ColorTeal.TrueColor()))

	subX := (w - len(subtitle)) / 2
	t.DrawText(subX, titleY+2, subtitle, style.Foreground(tcell.ColorYellow.TrueColor()))

	// Draw instructions
	instructions := []string{
		"Controls:",
		"  ⬆️  Arrow Up    - Rotate",
		"  ⬇️  Arrow Down  - Soft Drop",
		"  ⬅️  Arrow Left  - Move Left",
		"  ➡️  Arrow Right - Move Right",
		"  ␣ Space        - Hard Drop",
		"  P              - Pause/Resume",
		"  Q / ESC        - Quit game",
		"  Ctrl+C/D/Q/X   - Exit",
		"",
		"Press any key to connect...",
	}

	instY := titleY + 6
	for _, inst := range instructions {
		instX := (w - len(inst)) / 2
		t.DrawText(instX, instY, inst, style)
		instY++
	}

	// Draw version info
	version := "Version 1.0.0"
	versionX := (w - len(version)) / 2
	t.DrawText(versionX, h-3, version, style.Dim(true))
}

// DrawGameOverScreen draws the game over screen
func (t *TUI) DrawGameOverScreen(state *protocol.StateMessage, style tcell.Style) {
	w, h := t.screen.Size()

	title := "GAME OVER"
	subtitle := fmt.Sprintf("Final Score: %d", state.Score)

	// Center the title
	titleX := (w - len(title)) / 2
	titleY := h / 3
	t.DrawText(titleX, titleY, title, style.Bold(true).Foreground(tcell.ColorRed.TrueColor()))

	subX := (w - len(subtitle)) / 2
	t.DrawText(subX, titleY+2, subtitle, style.Bold(true).Foreground(tcell.ColorYellow.TrueColor()))

	// Draw stats
	stats := []string{
		fmt.Sprintf("Level: %d", state.Level),
		fmt.Sprintf("Lines: %d", state.Lines),
		"",
		"Press Q or ESC to quit...",
	}

	statsY := titleY + 6
	for _, stat := range stats {
		statX := (w - len(stat)) / 2
		t.DrawText(statX, statsY, stat, style)
		statsY++
	}
}

// getPieceShape returns the rotated shape for a piece
func getPieceShape(pieceData protocol.PieceData) [][]int {
	// Get base shape
	shapes := map[piece.Type][][]int{
		piece.TypeI: {{1, 1, 1, 1}},
		piece.TypeO: {{1, 1}, {1, 1}},
		piece.TypeT: {{0, 1, 0}, {1, 1, 1}},
		piece.TypeS: {{0, 1, 1}, {1, 1, 0}},
		piece.TypeZ: {{1, 1, 0}, {0, 1, 1}},
		piece.TypeJ: {{1, 0, 0}, {1, 1, 1}},
		piece.TypeL: {{0, 0, 1}, {1, 1, 1}},
	}

	baseShape := shapes[pieceData.Type]
	if baseShape == nil {
		return nil
	}

	// Apply rotation
	return rotateShape(baseShape, pieceData.Rotation)
}

// rotateShape rotates a shape by the given number of 90° clockwise rotations
func rotateShape(shape [][]int, rotation int) [][]int {
	result := shape
	for i := 0; i < rotation; i++ {
		result = rotate90(result)
	}
	return result
}

// rotate90 rotates a shape 90° clockwise
func rotate90(shape [][]int) [][]int {
	rows := len(shape)
	cols := len(shape[0])
	rotated := make([][]int, cols)

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

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
