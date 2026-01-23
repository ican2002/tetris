#!/bin/bash
# Test script to verify all exit keys work correctly
# This script starts the server and then provides instructions for manual testing

set -e

echo "=== Tetris Exit Keys Test ==="
echo ""
echo "This test will:"
echo "1. Start the server in background"
echo "2. Provide instructions for manual testing"
echo "3. Clean up when done"
echo ""

# Kill any existing server
pkill -f "bin/server" 2>/dev/null || true
sleep 1

# Start server
echo "Starting server on :8080..."
./bin/server -addr :8080 > /tmp/tetris-server.log 2>&1 &
SERVER_PID=$!
sleep 2

# Verify server started
if ! ps -p $SERVER_PID > /dev/null; then
    echo "ERROR: Server failed to start"
    cat /tmp/tetris-server.log
    exit 1
fi

echo "Server started (PID: $SERVER_PID)"
echo ""

# Check health
HEALTH=$(curl -s http://localhost:8080/health)
echo "Server health: $HEALTH"
echo ""

echo "=== Manual Testing Instructions ==="
echo ""
echo "The server is running in background."
echo "Please open TWO separate terminals and run:"
echo ""
echo "  Terminal 1: ./bin/tetris"
echo "  Terminal 2: ./bin/tetris"
echo ""
echo "Then test the following exit keys in each client:"
echo ""
echo "  1. Q key - Should exit client only"
echo "  2. ESC key - Should exit client only"
echo "  3. Ctrl+D - Should exit client only"
echo "  4. Ctrl+Q - Should exit client only"
echo "  5. Ctrl+X - Should exit client only"
echo "  6. Ctrl+C - Should exit client only (NOT the server!)"
echo ""
echo "After each test, verify:"
echo "  - Client exits"
echo "  - Server continues running (check: curl http://localhost:8080/health)"
echo ""
echo "Press Enter when you're done testing to stop the server..."
read

# Check if server is still running
if ps -p $SERVER_PID > /dev/null; then
    echo "Server is still running, stopping..."
    kill $SERVER_PID
    sleep 1
fi

# Final check
if ps -p $SERVER_PID > /dev/null; then
    echo "WARNING: Server did not stop gracefully, killing..."
    kill -9 $SERVER_PID
fi

echo ""
echo "=== Test Complete ==="
echo "Server log saved to /tmp/tetris-server.log"
