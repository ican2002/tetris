package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/ican2002/tetris/pkg/piece"
)

// TUI is the main UI struct
type TUI struct {
	screen   tcell.Screen
	width    int
	height   int
	eventCh  chan tcell.Event
	quitCh   chan struct{}

	// Layout
	boardX      int
	boardY      int
	boardWidth  int
	boardHeight int
	infoX       int
	infoY       int

	// State
	running bool
}

// Color mapping from hex colors to tcell colors
var colorMap = map[piece.Color]tcell.Color{
	piece.ColorCyan:   tcell.ColorTeal,
	piece.ColorYellow: tcell.ColorYellow,
	piece.ColorPurple: tcell.ColorPurple,
	piece.ColorGreen:  tcell.ColorGreen,
	piece.ColorRed:    tcell.ColorRed,
	piece.ColorBlue:   tcell.ColorBlue,
	piece.ColorOrange: tcell.ColorOrange,
	piece.ColorEmpty:  tcell.ColorDefault,
}

// Color is a type alias for protocol color
type Color = piece.Color

// New creates a new TUI instance
func New() (*TUI, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	t := &TUI{
		screen:  screen,
		width:   80,
		height:  24,
		eventCh: make(chan tcell.Event, 10),
		quitCh:  make(chan struct{}),
	}

	// Set default styles
	screen.SetStyle(tcell.StyleDefault)

	// Clear screen
	screen.Clear()
	screen.Sync()

	// Get terminal size
	t.UpdateSize()

	// Start event pump
	go t.eventPump()

	return t, nil
}

// eventPump continuously polls events and sends them to the channel
func (t *TUI) eventPump() {
	for {
		select {
		case <-t.quitCh:
			return
		default:
			ev := t.screen.PollEvent()
			t.eventCh <- ev
		}
	}
}

// UpdateSize updates the terminal size
func (t *TUI) UpdateSize() {
	w, h := t.screen.Size()
	t.width = w
	t.height = h

	// Calculate layout
	t.boardX = 2
	t.boardY = 3
	t.boardWidth = 22  // 10 cells * 2 width + borders
	t.boardHeight = 22 // 20 cells + borders

	t.infoX = t.boardX + t.boardWidth + 2
	t.infoY = t.boardY
}

// Close closes the TUI and restores terminal state
func (t *TUI) Close() {
	t.running = false
	close(t.quitCh) // Stop event pump
	t.screen.Fini()
}

// Clear clears the screen
func (t *TUI) Clear() {
	t.screen.Clear()
}

// Sync updates the screen
func (t *TUI) Sync() {
	t.screen.Show()
}

// SetRunning sets the running state
func (t *TUI) SetRunning(running bool) {
	t.running = running
}

// IsRunning returns whether the TUI is running
func (t *TUI) IsRunning() bool {
	return t.running
}

// PollEvent waits for and returns the next event
func (t *TUI) PollEvent() tcell.Event {
	return <-t.eventCh
}

// PollEventWithTimeout waits for an event with a timeout
func (t *TUI) PollEventWithTimeout(timeout time.Duration) tcell.Event {
	select {
	case ev := <-t.eventCh:
		return ev
	case <-time.After(timeout):
		return nil
	}
}

// PostEvent posts an event to the event queue
func (t *TUI) PostEvent(ev tcell.Event) {
	t.screen.PostEvent(ev)
}

// GetColor returns the tcell color for a piece color
func GetColor(color piece.Color) tcell.Color {
	if c, ok := colorMap[color]; ok {
		return c
	}
	return tcell.ColorDefault
}

// DrawBox draws a box with borders
func (t *TUI) DrawBox(x, y, width, height int, title string, style tcell.Style) {
	// Draw corners and horizontal lines
	t.screen.SetContent(x, y, '┌', nil, style)
	t.screen.SetContent(x+width-1, y, '┐', nil, style)
	t.screen.SetContent(x, y+height-1, '└', nil, style)
	t.screen.SetContent(x+width-1, y+height-1, '┘', nil, style)

	for i := x + 1; i < x+width-1; i++ {
		t.screen.SetContent(i, y, '─', nil, style)
		t.screen.SetContent(i, y+height-1, '─', nil, style)
	}

	// Draw vertical lines
	for i := y + 1; i < y+height-1; i++ {
		t.screen.SetContent(x, i, '│', nil, style)
		t.screen.SetContent(x+width-1, i, '│', nil, style)
	}

	// Draw title if provided
	if title != "" && width > len(title)+4 {
		titleX := x + (width-len(title))/2
		for i, ch := range title {
			t.screen.SetContent(titleX+i, y, ch, nil, style.Bold(true))
		}
	}
}

// DrawText draws text at the specified position
func (t *TUI) DrawText(x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		t.screen.SetContent(x+i, y, ch, nil, style)
	}
}

// DrawTextAligned draws aligned text
func (t *TUI) DrawTextAligned(x, y, width int, text string, alignment int, style tcell.Style) {
	textLen := len(text)
	if textLen > width {
		text = text[:width]
		textLen = width
	}

	var xPos int
	switch alignment {
	case -1: // Left
		xPos = x
	case 0: // Center
		xPos = x + (width-textLen)/2
	case 1: // Right
		xPos = x + width - textLen
	default:
		xPos = x
	}

	t.DrawText(xPos, y, text, style)
}

// FillRect fills a rectangle with the specified character
func (t *TUI) FillRect(x, y, width, height int, ch rune, style tcell.Style) {
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			t.screen.SetContent(col, row, ch, nil, style)
		}
	}
}

// GetSize returns the terminal size
func (t *TUI) GetSize() (int, int) {
	return t.screen.Size()
}

// CheckMinimumSize checks if terminal meets minimum size requirements
func (t *TUI) CheckMinimumSize() bool {
	w, h := t.screen.Size()
	return w >= 80 && h >= 24
}
