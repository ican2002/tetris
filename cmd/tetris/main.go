package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/ican2002/tetris/pkg/protocol"
	"github.com/ican2002/tetris/pkg/tui"
	"github.com/ican2002/tetris/pkg/wsclient"
)

// LogBuffer manages log messages with thread safety
type LogBuffer struct {
	messages []string
	mu       sync.Mutex
	maxSize  int
}

func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		messages: make([]string, 0, size),
		maxSize:  size,
	}
}

func (lb *LogBuffer) Add(msg string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Add timestamp
	timestamp := time.Now().Format("15:04:05")
	lb.messages = append(lb.messages, fmt.Sprintf("[%s] %s", timestamp, msg))

	// Keep only the last maxSize messages
	if len(lb.messages) > lb.maxSize {
		lb.messages = lb.messages[1:]
	}
}

func (lb *LogBuffer) GetMessages() []string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.messages
}

var (
	serverAddr = flag.String("server", "ws://localhost:8080/ws", "WebSocket server address")
)

func main() {
	flag.Parse()

	// Create log buffer
	logBuffer := NewLogBuffer(100)

	// Create TUI
	ui, err := tui.New()
	if err != nil {
		log.Fatalf("Failed to create TUI: %v", err)
	}
	defer ui.Close()

	// Check minimum size
	if !ui.CheckMinimumSize() {
		log.Println("Terminal size must be at least 80x30")
		return
	}

	logBuffer.Add("TUI initialized")

	// Show welcome screen
	showWelcome(ui, logBuffer)

	// Create WebSocket client
	client := wsclient.New(*serverAddr)
	client.SetMaxRetries(5)
	client.SetRetryDelay(3 * time.Second)

	// Set up callbacks
	var currentState *protocol.StateMessage
	var statusMsg string
	var gameOver bool

	client.SetOnConnected(func() {
		statusMsg = "Connected to server"
		logBuffer.Add("✓ Connected to server")
	})
	client.SetOnDisconnected(func() {
		statusMsg = "Disconnected from server"
		logBuffer.Add("✗ Disconnected from server")
	})
	client.SetOnError(func(err error) {
		statusMsg = fmt.Sprintf("Error: %v", err)
		logBuffer.Add(fmt.Sprintf("✗ Error: %v", err))
	})
	client.SetOnStateChange(func(data []byte) {
		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			logBuffer.Add(fmt.Sprintf("✗ Failed to parse message: %v", err))
			return
		}

		switch msg.Type {
		case protocol.MessageTypeState:
			// Parse StateMessage from map
			state, err := parseStateMessage(msg.Data)
			if err != nil {
				logBuffer.Add(fmt.Sprintf("✗ Failed to parse state: %v", err))
				return
			}
			currentState = state

		case protocol.MessageTypeError:
			errMsg, err := parseErrorMessage(msg.Data)
			if err != nil {
				logBuffer.Add(fmt.Sprintf("✗ Failed to parse error: %v", err))
				return
			}
			statusMsg = errMsg.Error
			logBuffer.Add(fmt.Sprintf("✗ Server error: %s", errMsg.Error))

		case protocol.MessageTypeGameOver:
			gameOver = true
			overMsg, err := parseGameOverMessage(msg.Data)
			if err != nil {
				logBuffer.Add(fmt.Sprintf("✗ Failed to parse game over: %v", err))
				return
			}
			statusMsg = fmt.Sprintf("Game Over! Score: %d", overMsg.Score)
			logBuffer.Add(fmt.Sprintf("† Game Over! Score: %d, Level: %d, Lines: %d",
				overMsg.Score, overMsg.Level, overMsg.Lines))

		case protocol.MessageTypePing:
			// Pings are handled automatically by the client
		}
	})

	// Connect to server
	ui.SetRunning(true)
	statusMsg = "Connecting to server..."
	logBuffer.Add("Connecting to " + *serverAddr)

	// Start connection in background
	go func() {
		if err := client.Connect(); err != nil {
			statusMsg = fmt.Sprintf("Failed to connect: %v", err)
			logBuffer.Add(fmt.Sprintf("✗ Failed to connect: %v", err))
		}
	}()

	// Main loop
	style := tcell.StyleDefault

	for ui.IsRunning() {
		// Handle events first (with short timeout for responsive input)
		ev := ui.PollEventWithTimeout(50 * time.Millisecond)

		if ev != nil {
			switch ev := ev.(type) {
			case *tcell.EventKey:
				// Log the key that was pressed (for debugging)
				keyName := tcell.KeyNames[ev.Key()]
				if keyName == "" {
					keyName = fmt.Sprintf("Rune(%c)", ev.Rune())
				}
				logBuffer.Add(fmt.Sprintf("Key: %s", keyName))

				// Check for quit keys FIRST (before any other logic)
				// This prevents Q key from triggering reconnect when not connected
				if isQuitKey(ev) {
					logBuffer.Add("Quit requested")
					ui.SetRunning(false)
					continue
				}

				if gameOver {
					// Game over state - already handled above
					continue
				}

				if !client.IsConnected() && !gameOver {
					// Any non-quit key to start connecting
					logBuffer.Add("Connecting...")
					go client.Connect()
					continue
				}

				// Handle game control keys
				if handleKeyEvent(ev, client, logBuffer) {
					ui.SetRunning(false)
					continue
				}

			case *tcell.EventResize:
				ui.UpdateSize()
				if !ui.CheckMinimumSize() {
					statusMsg = "Terminal too small (min 80x30)"
				}
			}
		}

		// Then draw current state
		ui.Clear()

		if currentState == nil && !gameOver {
			// Show welcome screen
			ui.DrawWelcomeScreen(style)
		} else if gameOver {
			// Show game over screen
			if currentState != nil {
				ui.DrawGameOverScreen(currentState, style)
			}
		} else if currentState != nil {
			// Draw game (use rows 1-20 for game)
			// Draw a box around the entire game area
			ui.DrawBox(1, 0, 78, 22, "", style)
			ui.DrawBoard(2, 1, currentState, style)
			ui.DrawInfoPanel(26, 1, currentState, style)
		}

		// Draw status bar (row 22)
		ui.DrawStatusBar(0, 22, 80, statusMsg, client.IsConnected(), style)

		// Draw separator line
		ui.DrawText(0, 23, strings.Repeat("─", 80), style.Dim(true))

		// Draw log window (rows 24-29, 6 rows for logs)
		drawLogWindow(ui, 0, 24, 80, 6, logBuffer, style)

		// Update screen
		ui.Sync()
	}
}

// Helper functions to parse messages from map[string]interface{}

func parseStateMessage(data interface{}) (*protocol.StateMessage, error) {
	// Convert to JSON and then to StateMessage
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var state protocol.StateMessage
	if err := json.Unmarshal(jsonBytes, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func parseErrorMessage(data interface{}) (protocol.ErrorMessage, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return protocol.ErrorMessage{}, err
	}

	var errMsg protocol.ErrorMessage
	if err := json.Unmarshal(jsonBytes, &errMsg); err != nil {
		return protocol.ErrorMessage{}, err
	}

	return errMsg, nil
}

func parseGameOverMessage(data interface{}) (protocol.GameOverMessage, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return protocol.GameOverMessage{}, err
	}

	var overMsg protocol.GameOverMessage
	if err := json.Unmarshal(jsonBytes, &overMsg); err != nil {
		return protocol.GameOverMessage{}, err
	}

	return overMsg, nil
}

func showWelcome(ui *tui.TUI, logBuffer *LogBuffer) {
	style := tcell.StyleDefault
	ui.DrawWelcomeScreen(style)
	ui.Sync()

	logBuffer.Add("Welcome! Press any key to start...")

	// Wait for any key
	for {
		ev := ui.PollEvent()
		if _, ok := ev.(*tcell.EventKey); ok {
			logBuffer.Add("Starting game...")
			break
		}
	}
}

func drawLogWindow(ui *tui.TUI, x, y, width, height int, logBuffer *LogBuffer, style tcell.Style) {
	// Draw box border using TUI's DrawBox method
	ui.DrawBox(x, y, width, height, "Messages", style)

	// Get log messages
	messages := logBuffer.GetMessages()

	// Calculate how many messages we can show
	maxLines := height - 2
	startIdx := len(messages) - maxLines
	if startIdx < 0 {
		startIdx = 0
	}

	// Draw messages (bottom-up, showing newest)
	for i := 0; i < maxLines && i < len(messages); i++ {
		msgIdx := startIdx + i
		if msgIdx >= len(messages) {
			break
		}

		lineY := y + height - 2 - i
		msg := messages[msgIdx]

		// Truncate if too long
		maxMsgLen := width - 4
		if len(msg) > maxMsgLen {
			msg = msg[:maxMsgLen]
		}

		// Draw message using TUI's DrawText method
		ui.DrawText(x+2, lineY, msg, style)
	}
}

func handleKeyEvent(ev *tcell.EventKey, client *wsclient.Client, logBuffer *LogBuffer) bool {
	var cmdType protocol.MessageType

	switch ev.Key() {
	case tcell.KeyLeft:
		cmdType = protocol.MessageTypeMoveLeft
	case tcell.KeyRight:
		cmdType = protocol.MessageTypeMoveRight
	case tcell.KeyDown:
		cmdType = protocol.MessageTypeMoveDown
	case tcell.KeyUp:
		cmdType = protocol.MessageTypeRotate
	case tcell.KeyEnter:
		cmdType = protocol.MessageTypeHardDrop
	default:
		switch ev.Rune() {
		case ' ', 'x', 'X':
			cmdType = protocol.MessageTypeHardDrop
		case 'p', 'P':
			cmdType = protocol.MessageTypePause
		case 'r', 'R':
			cmdType = protocol.MessageTypeResume
		default:
			return false
		}
	}

	if cmdType != "" {
		cmd := protocol.ControlMessage{Type: cmdType}
		data, err := json.Marshal(cmd)
		if err != nil {
			log.Printf("Failed to marshal command: %v", err)
			logBuffer.Add(fmt.Sprintf("✗ Failed to marshal command: %v", err))
			return false
		}

		if err := client.Send(data); err != nil {
			log.Printf("Failed to send command: %v", err)
			logBuffer.Add(fmt.Sprintf("✗ Failed to send %s: %v", cmdType, err))
		} else {
			// Log key commands (including rotate for debugging)
			switch cmdType {
			case protocol.MessageTypeRotate:
				logBuffer.Add("→ rotate")
			case protocol.MessageTypeMoveLeft, protocol.MessageTypeMoveRight, protocol.MessageTypeMoveDown:
				logBuffer.Add(fmt.Sprintf("→ %s", cmdType))
			case protocol.MessageTypePause, protocol.MessageTypeResume, protocol.MessageTypeHardDrop:
				logBuffer.Add(fmt.Sprintf("→ %s", cmdType))
			}
		}
	}

	return false
}

// isQuitKey checks if the key event is a quit command
func isQuitKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC, tcell.KeyCtrlD, tcell.KeyCtrlQ, tcell.KeyCtrlX:
		return true
	default:
		switch ev.Rune() {
		case 'q', 'Q':
			return true
		}
	}
	return false
}
