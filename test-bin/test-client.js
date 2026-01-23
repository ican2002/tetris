// Simple WebSocket client for testing the Tetris server
const WebSocket = require('ws');

// Connect to the server
const ws = new WebSocket('ws://localhost:9292/ws');

// Connection opened
ws.on('open', function open() {
  console.log('Connected to server');
  
  // Send some test commands
  setTimeout(() => {
    console.log('Sending move_left command');
    ws.send(JSON.stringify({ type: 'move_left' }));
  }, 1000);
  
  setTimeout(() => {
    console.log('Sending move_right command');
    ws.send(JSON.stringify({ type: 'move_right' }));
  }, 2000);
  
  setTimeout(() => {
    console.log('Sending rotate command');
    ws.send(JSON.stringify({ type: 'rotate' }));
  }, 3000);
  
  // Keep the connection open for a while to be visible in admin UI
  setTimeout(() => {
    console.log('Closing connection');
    ws.close();
  }, 10000);
});

// Listen for messages
ws.on('message', function incoming(data) {
  try {
    const messages = data.split('\n');
    messages.forEach(msgStr => {
      if (msgStr.trim()) {
        const msg = JSON.parse(msgStr);
        if (msg.type === 'state') {
          console.log('Received game state:');
          console.log('  Score:', msg.data.score);
          console.log('  Level:', msg.data.level);
          console.log('  Lines:', msg.data.lines);
        } else if (msg.type === 'error') {
          console.error('Error:', msg.data.error);
        }
      }
    });
  } catch (e) {
    console.error('Error parsing message:', e);
  }
});

// Connection closed
ws.on('close', function close() {
  console.log('Disconnected from server');
});

// Connection error
ws.on('error', function error(err) {
  console.error('WebSocket error:', err);
});