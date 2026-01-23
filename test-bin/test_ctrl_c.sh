#!/bin/bash
# Test Ctrl+C exit functionality

echo "=== Testing Ctrl+C Exit Functionality ==="
echo ""

# Start server
pkill -f "bin/server" 2>/dev/null || true
sleep 1
./bin/server -addr :8080 > /tmp/tetris-server.log 2>&1 &
SERVER_PID=$!
sleep 2

if ! ps -p $SERVER_PID > /dev/null; then
    echo "ERROR: Server failed to start"
    cat /tmp/tetris-server.log
    exit 1
fi

echo "✓ Server started (PID: $SERVER_PID)"
echo ""
echo "Manual test instructions:"
echo ""
echo "1. Run: ./bin/tetris"
echo "2. Press Ctrl+C"
echo "3. Expected: TUI client exits gracefully"
echo "4. Verify server is still running: curl http://localhost:8080/health"
echo ""
echo "Other exit keys to test:"
echo "  - Q key"
echo "  - ESC key"
echo "  - Ctrl+D"
echo "  - Ctrl+Q"
echo "  - Ctrl+X"
echo ""
echo "Press Enter to stop server and exit..."
read

# Stop server
if ps -p $SERVER_PID > /dev/null; then
    kill $SERVER_PID
    sleep 1
fi

echo ""
echo "=== Test Complete ==="
