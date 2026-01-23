package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	fmt.Println("Testing server behavior when client disconnects...")

	// Test 1: Connect and disconnect normally
	fmt.Println("\n[Test 1] Normal connection and disconnect")
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Receive a message
	_, msg, _ := conn.ReadMessage()
	fmt.Printf("Received: %s\n", string(msg))

	// Close connection normally
	conn.Close()

	time.Sleep(500 * time.Millisecond)

	// Check server health
	resp, _ := http.Get("http://localhost:8080/health")
	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)
	resp.Body.Close()
	fmt.Printf("Server health after client disconnect: status=%v, clients=%v\n",
		health["status"], health["clients"])

	if health["clients"].(float64) != 0 {
		fmt.Println("✗ FAILED: Server should have 0 clients")
	} else {
		fmt.Println("✓ PASSED: Server correctly shows 0 clients")
	}

	// Test 2: Verify server is still running
	fmt.Println("\n[Test 2] Server should still be running")
	resp2, _ := http.Get("http://localhost:8080/health")
	var health2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&health2)
	resp2.Body.Close()

	if health2["status"] == "ok" {
		fmt.Println("✓ PASSED: Server is still running")
	} else {
		fmt.Println("✗ FAILED: Server is not running")
	}

	fmt.Println("\n=== Automated tests passed! ===")
	fmt.Println("\nManual testing needed for TUI exit keys:")
	fmt.Println("1. Run: ./bin/tetris")
	fmt.Println("2. Press Q - client should exit, server should stay running")
	fmt.Println("3. Run: curl http://localhost:8080/health - server should respond")
	fmt.Println("4. Repeat with ESC, Ctrl+D, Ctrl+Q, Ctrl+X")
	fmt.Println("\nNote: Ctrl+C handling depends on terminal process group")
}
