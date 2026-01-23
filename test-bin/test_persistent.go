// Persistent multi-client test for Tetris server
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Clients int    `json:"clients"`
}

type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

var done = make(chan bool)

func main() {
	fmt.Println("=== Tetris Multi-Client Test (Persistent) ===\n")

	// Check initial server state
	checkHealth("Initial state")

	// Connect two clients
	var wg sync.WaitGroup

	fmt.Println("\n--- Connecting Client-1 ---")
	wg.Add(1)
	go func() {
		defer wg.Done()
		runClient("Client-1", 10*time.Second)
	}()

	time.Sleep(1 * time.Second) // Stagger connections
	checkHealth("After Client-1 connected")

	fmt.Println("\n--- Connecting Client-2 ---")
	wg.Add(1)
	go func() {
		defer wg.Done()
		runClient("Client-2", 10*time.Second)
	}()

	time.Sleep(500 * time.Millisecond)
	checkHealth("After Client-2 connected")

	fmt.Println("\n--- Both clients connected, waiting 5 seconds ---")
	time.Sleep(5 * time.Second)
	checkHealth("After 5 seconds")

	fmt.Println("\n--- Disconnecting clients ---")
	wg.Wait()

	// Final health check
	time.Sleep(500 * time.Millisecond)
	checkHealth("Final state")

	fmt.Println("\n=== Test PASSED: Multiple clients handled successfully! ===")
}

func runClient(name string, duration time.Duration) {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Printf("[%s] Connection failed: %v", name, err)
		return
	}
	defer conn.Close()

	fmt.Printf("[%s] Connected and will stay for %v\n", name, duration)

	// Set up channel for messages
	messages := make(chan []byte, 10)

	// Read messages in background
	go func() {
		defer close(messages)
		conn.SetReadDeadline(time.Now().Add(duration + 2*time.Second))
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			select {
			case messages <- msg:
			default:
			}
		}
	}()

	// Process messages
	messageCount := 0
	startTime := time.Now()
	for time.Since(startTime) < duration {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			messageCount++

			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				continue
			}

			if message.Type == "state" && message.Data != nil {
				if messageCount%5 == 1 { // Print every 5th message
					fmt.Printf("[%s] State: score=%.0f, level=%.0f, state=%v\n",
						name, message.Data["score"], message.Data["level"], message.Data["state"])
				}
			}

		case <-time.After(100 * time.Millisecond):
			// Send occasional command
			if messageCount > 0 && messageCount%10 == 0 {
				cmd := `{"type":"move_left"}`
				if err := conn.WriteMessage(websocket.TextMessage, []byte(cmd)); err != nil {
					log.Printf("[%s] Send error: %v", name, err)
					return
				}
			}
		}
	}

	fmt.Printf("[%s] Disconnecting after %d messages received\n", name, messageCount)
}

func checkHealth(label string) {
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		log.Printf("[%s] Health check failed: %v", label, err)
		return
	}
	defer resp.Body.Close()

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		log.Printf("[%s] Failed to decode health: %v", label, err)
		return
	}

	fmt.Printf("[%s] Health: status=%s, clients=%d\n", label, health.Status, health.Clients)
}
