package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/ican2002/tetris/pkg/protocol"
	"github.com/ican2002/tetris/pkg/tui"
	"github.com/ican2002/tetris/pkg/wsclient"
)

var (
	serverAddr = flag.String("server", "ws://localhost:8080/ws", "WebSocket server address")
)

func main() {
	flag.Parse()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create TUI
	ui, err := tui.New()
	if err != nil {
		log.Fatalf("Failed to create TUI: %v", err)
	}
	defer ui.Close()

	// Check minimum size
	if !ui.CheckMinimumSize() {
		log.Println("Terminal size must be at least 80x24")
		return
	}

	// Show welcome screen
	showWelcome(ui)

	// Create WebSocket client
	client := wsclient.New(*serverAddr)
	client.SetMaxRetries(5)
	client.SetRetryDelay(3 * time.Second)

	// Set up callbacks
	var currentState *protocol.StateMessage
	var message string
	var gameOver bool

	client.SetOnConnected(func() {
		message = "Connected to server"
	})
	client.SetOnDisconnected(func() {
		message = "Disconnected from server"
	})
	client.SetOnError(func(err error) {
		message = fmt.Sprintf("Error: %v", err)
	})
	client.SetOnStateChange(func(data []byte) {
		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}

		switch msg.Type {
		case protocol.MessageTypeState:
			currentState = msg.Data.(*protocol.StateMessage)
		case protocol.MessageTypeError:
			errMsg := msg.Data.(protocol.ErrorMessage)
			message = errMsg.Error
		case protocol.MessageTypeGameOver:
			gameOver = true
			overMsg := msg.Data.(protocol.GameOverMessage)
			message = fmt.Sprintf("Game Over! Score: %d", overMsg.Score)
		}
	})

	// Connect to server
	ui.SetRunning(true)
	message = "Connecting to server..."

	// Start connection in background
	go func() {
		if err := client.Connect(); err != nil {
			message = fmt.Sprintf("Failed to connect: %v", err)
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
			// Draw game
			ui.DrawBoard(2, 1, currentState, style)
			ui.DrawInfoPanel(26, 1, currentState, style)
		}

		// Draw status bar
		ui.DrawStatusBar(0, 23, 80, message, client.IsConnected(), style)

		// Update screen
		ui.Sync()

		// Handle events
		ev := ui.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if !client.IsConnected() {
				// Any key to start connecting
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
			if handleKeyEvent(ev, client) {
				ui.SetRunning(false)
			}

		case *tcell.EventResize:
			ui.UpdateSize()
			if !ui.CheckMinimumSize() {
				message = "Terminal too small (min 80x24)"
			}
		}

		// Check for signals
		select {
		case <-sigChan:
			ui.SetRunning(false)
		default:
		}
	}
}

func showWelcome(ui *tui.TUI) {
	style := tcell.StyleDefault
	ui.DrawWelcomeScreen(style)
	ui.Sync()

	// Wait for any key
	for {
		ev := ui.PollEvent()
		if _, ok := ev.(*tcell.EventKey); ok {
			break
		}
	}
}

func handleKeyEvent(ev *tcell.EventKey, client *wsclient.Client) bool {
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
		return true // Signal to quit
	case tcell.KeyEnter:
		cmdType = protocol.MessageTypeHardDrop
	case tcell.KeyCtrlC:
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
			return false
		}

		if err := client.Send(data); err != nil {
			log.Printf("Failed to send command: %v", err)
		}
	}

	return false
}
