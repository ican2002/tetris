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
	// Connect to the admin WebSocket endpoint
	u := url.URL{Scheme: "ws", Host: "localhost:9292", Path: "/ws/admin"}
	fmt.Printf("Connecting to admin endpoint: %s\n", u.String())

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
			// Process admin message
			processAdminMessage(message)
		}
	}()

	// Keep connection open for a while to receive updates
	fmt.Println("Connected to admin endpoint. Waiting for 10 seconds to receive updates...")
	time.Sleep(10 * time.Second)

	// Connect a game client while admin is connected
	fmt.Println("Connecting a game client to test real-time updates...")
	gameClient := connectGameClient()
	if gameClient != nil {
		defer gameClient.Close()
		// Wait for updates
		time.Sleep(5 * time.Second)
		// Send some commands to generate activity
		sendGameCommands(gameClient)
		// Wait for updates
		time.Sleep(5 * time.Second)
	}

	// Close connection
	fmt.Println("Closing admin connection")
	if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		log.Println("write close:", err)
		return
	}

	select {
	case <-done:
	case <-time.After(time.Second):
	}

	fmt.Println("Admin test completed")
}

// connectGameClient connects a game client for testing
func connectGameClient() *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: "localhost:9292", Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("dial game client:", err)
		return nil
	}
	fmt.Println("Game client connected")
	return c
}

// sendGameCommands sends some game commands
func sendGameCommands(c *websocket.Conn) {
	commands := []string{"move_left", "rotate", "hard_drop"}
	for _, cmd := range commands {
		time.Sleep(1 * time.Second)
		fmt.Printf("Game client sending %s command\n", cmd)
		cmdData := map[string]string{"type": cmd}
		data, _ := json.Marshal(cmdData)
		c.WriteMessage(websocket.TextMessage, data)
	}
}

// processAdminMessage processes admin messages
func processAdminMessage(data []byte) {
	var adminData map[string]interface{}
	if err := json.Unmarshal(data, &adminData); err != nil {
		log.Println("error parsing admin message:", err)
		return
	}

	fmt.Println("\nReceived admin update:")
	
	// Print stats
	if currentClients, ok := adminData["currentClients"].(float64); ok {
		fmt.Printf("- Current Clients: %.0f\n", currentClients)
	}
	if totalClients, ok := adminData["totalClients"].(float64); ok {
		fmt.Printf("- Total Clients: %.0f\n", totalClients)
	}
	if peakClients, ok := adminData["peakClients"].(float64); ok {
		fmt.Printf("- Peak Clients: %.0f\n", peakClients)
	}

	// Print client details
	if clients, ok := adminData["clients"].([]interface{}); ok {
		fmt.Printf("- Connected Clients: %d\n", len(clients))
		for i, client := range clients {
			if clientInfo, ok := client.(map[string]interface{}); ok {
				fmt.Printf("  Client %d:\n", i+1)
				if id, ok := clientInfo["id"].(string); ok {
					fmt.Printf("    ID: %s\n", id)
				}
				if addr, ok := clientInfo["address"].(string); ok {
					fmt.Printf("    Address: %s\n", addr)
				}
				if gameState, ok := clientInfo["gameState"].(string); ok {
					fmt.Printf("    Game State: %s\n", gameState)
				}
				if score, ok := clientInfo["score"].(float64); ok {
					fmt.Printf("    Score: %.0f\n", score)
				}
				if level, ok := clientInfo["level"].(float64); ok {
					fmt.Printf("    Level: %.0f\n", level)
				}
				if lines, ok := clientInfo["lines"].(float64); ok {
					fmt.Printf("    Lines: %.0f\n", lines)
				}
			}
		}
	}
	
	// Print timestamp
	if timestamp, ok := adminData["timestamp"].(string); ok {
		fmt.Printf("- Timestamp: %s\n", timestamp)
	}
}
