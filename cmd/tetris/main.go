package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for ui.IsRunning() {
		// Draw current state
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
			ui.DrawBoard(2, 1, currentState, style)
			ui.DrawInfoPanel(26, 1, currentState, style)
		}

		// Draw status bar (row 21)
		ui.DrawStatusBar(0, 21, 80, statusMsg, client.IsConnected(), style)

		// Draw log window (rows 22-29, 8 rows for logs)
		drawLogWindow(ui, 0, 22, 80, 8, logBuffer, style)

		// Update screen
		ui.Sync()

		// Handle events
		ev := ui.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if !client.IsConnected() && !gameOver {
				// Any key to start connecting
				logBuffer.Add("Reconnecting...")
				go client.Connect()
				continue
			}

			if gameOver {
				// Game over - Q or ESC to quit
				if ev.Key() == tcell.KeyEsc || ev.Rune() == 'q' || ev.Rune() == 'Q' {
					ui.SetRunning(false)
				}
				continue
			}

			// Handle keyboard input
			if handleKeyEvent(ev, client, logBuffer) {
				ui.SetRunning(false)
			}

		case *tcell.EventResize:
			ui.UpdateSize()
			if !ui.CheckMinimumSize() {
				statusMsg = "Terminal too small (min 80x30)"
			}
		}

		// Check for signals
		select {
		case <-sigChan:
			logBuffer.Add("Received interrupt signal, shutting down...")
			ui.SetRunning(false)
		default:
		}
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
	case tcell.KeyEscape:
		logBuffer.Add("Quit requested (ESC)")
		return true // Signal to quit
	case tcell.KeyEnter:
		cmdType = protocol.MessageTypeHardDrop
	case tcell.KeyCtrlC:
		logBuffer.Add("Quit requested (Ctrl+C)")
		return true // Signal to quit
	default:
		switch ev.Rune() {
		case ' ', 'x', 'X':
			cmdType = protocol.MessageTypeHardDrop
		case 'p', 'P':
			cmdType = protocol.MessageTypePause
		case 'r', 'R':
			cmdType = protocol.MessageTypeResume
		case 'q', 'Q':
			logBuffer.Add("Quit requested (Q)")
			return true // Signal to quit
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
			// Log key commands (but not all, to avoid spam)
			switch cmdType {
			case protocol.MessageTypePause, protocol.MessageTypeResume, protocol.MessageTypeHardDrop:
				logBuffer.Add(fmt.Sprintf("→ %s", cmdType))
			}
		}
	}

	return false
}
