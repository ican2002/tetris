package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// Connect to the server
	u := url.URL{Scheme: "ws", Host: "localhost:9292", Path: "/ws"}
	fmt.Printf("Connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Read messages from server
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			// Process message
			processMessage(message)
		}
	}()

	// Send some test commands
	testCommands := []string{
		"move_left",
		"move_right",
		"rotate",
		"move_down",
		"hard_drop",
	}

	for i, cmd := range testCommands {
		time.Sleep(time.Duration(i+1) * time.Second)
		fmt.Printf("Sending %s command\n", cmd)
		if err := sendCommand(c, cmd); err != nil {
			log.Println("send:", err)
			return
		}
	}

	// Keep connection open for a while
	time.Sleep(5 * time.Second)

	// Send pause command
	fmt.Println("Sending pause command")
	if err := sendCommand(c, "pause"); err != nil {
		log.Println("send:", err)
		return
	}

	// Keep connection open for a while more
	time.Sleep(5 * time.Second)

	// Send resume command
	fmt.Println("Sending resume command")
	if err := sendCommand(c, "resume"); err != nil {
		log.Println("send:", err)
		return
	}

	// Keep connection open for final check
	time.Sleep(5 * time.Second)

	fmt.Println("Closing connection")
	if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		log.Println("write close:", err)
		return
	}

	select {
	case <-done:
	case <-time.After(time.Second):
	}

	fmt.Println("Test completed")
}

// sendCommand sends a command to the server
func sendCommand(c *websocket.Conn, cmdType string) error {
	cmd := map[string]string{"type": cmdType}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}

// processMessage processes a message from the server
func processMessage(data []byte) {
	// Split message by newlines if multiple messages
	messages := string(data)
	for _, msgStr := range splitByNewline(messages) {
		if msgStr == "" {
			continue
		}
		
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(msgStr), &msg); err != nil {
			log.Println("error parsing message:", err)
			continue
		}
		
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "state":
				if data, ok := msg["data"].(map[string]interface{}); ok {
					fmt.Println("Received game state:")
					if score, ok := data["score"].(float64); ok {
						fmt.Printf("  Score: %.0f\n", score)
					}
					if level, ok := data["level"].(float64); ok {
						fmt.Printf("  Level: %.0f\n", level)
					}
					if lines, ok := data["lines"].(float64); ok {
						fmt.Printf("  Lines: %.0f\n", lines)
					}
					if state, ok := data["state"].(string); ok {
						fmt.Printf("  Game State: %s\n", state)
					}
				}
			case "error":
				if data, ok := msg["data"].(map[string]interface{}); ok {
					if errorMsg, ok := data["error"].(string); ok {
						fmt.Printf("Error: %s\n", errorMsg)
					}
				}
			case "game_over":
				fmt.Println("Game Over!")
				if data, ok := msg["data"].(map[string]interface{}); ok {
					if score, ok := data["score"].(float64); ok {
						fmt.Printf("Final Score: %.0f\n", score)
					}
				}
			}
		}
	}
}

// splitByNewline splits a string by newlines
func splitByNewline(s string) []string {
	var lines []string
	line := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}