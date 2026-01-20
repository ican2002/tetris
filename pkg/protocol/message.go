package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/ican2002/tetris/pkg/board"
	"github.com/ican2002/tetris/pkg/game"
	"github.com/ican2002/tetris/pkg/piece"
)

// MessageType represents the type of message
type MessageType string

const (
	// Client to Server messages
	MessageTypeMoveLeft  MessageType = "move_left"
	MessageTypeMoveRight MessageType = "move_right"
	MessageTypeMoveDown  MessageType = "move_down"
	MessageTypeRotate    MessageType = "rotate"
	MessageTypeHardDrop  MessageType = "hard_drop"
	MessageTypePause     MessageType = "pause"
	MessageTypeResume    MessageType = "resume"
	MessageTypePong      MessageType = "pong"

	// Server to Client messages
	MessageTypeState    MessageType = "state"
	MessageTypeError    MessageType = "error"
	MessageTypePing     MessageType = "ping"
	MessageTypeGameOver MessageType = "game_over"
)

// Message represents a WebSocket message
type Message struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// ControlMessage represents a control command from client
type ControlMessage struct {
	Type MessageType `json:"type"`
}

// StateMessage represents the game state sent to client
type StateMessage struct {
	Board        [][]string `json:"board"`
	CurrentPiece PieceData  `json:"current_piece"`
	NextPiece    PieceData  `json:"next_piece"`
	State        string     `json:"state"`
	Score        int        `json:"score"`
	Level        int        `json:"level"`
	Lines        int        `json:"lines"`
	DropInterval int        `json:"drop_interval_ms"`
}

// PieceData represents piece information for serialization
type PieceData struct {
	Type     piece.Type  `json:"type"`
	Color    piece.Color `json:"color"`
	X        int         `json:"x"`
	Y        int         `json:"y"`
	Rotation int         `json:"rotation"`
}

// ErrorMessage represents an error message
type ErrorMessage struct {
	Error string `json:"error"`
	Code  int    `json:"code,omitempty"`
}

// PingMessage represents a ping message
type PingMessage struct {
	Timestamp int64 `json:"timestamp"`
}

// PongMessage represents a pong message
type PongMessage struct {
	Timestamp int64 `json:"timestamp"`
}

// GameOverMessage represents a game over message
type GameOverMessage struct {
	Score int `json:"score"`
	Level int `json:"level"`
	Lines int `json:"lines"`
}

// NewStateMessage creates a state message from game state
func NewStateMessage(g *game.Game) *Message {
	b := g.GetBoard()
	pieces := make([][]string, board.Height)

	for y := 0; y < board.Height; y++ {
		pieces[y] = make([]string, board.Width)
		for x := 0; x < board.Width; x++ {
			cell, _ := b.GetCell(x, y)
			if cell.Empty {
				pieces[y][x] = ""
			} else {
				pieces[y][x] = string(cell.Color)
			}
		}
	}

	current := g.GetCurrentPiece()
	next := g.GetNextPiece()

	state := StateMessage{
		Board:        pieces,
		CurrentPiece: pieceToData(current),
		NextPiece:    pieceToData(next),
		State:        g.GetState().String(),
		Score:        g.GetScore(),
		Level:        g.GetLevel(),
		Lines:        g.GetLines(),
		DropInterval: int(g.GetDropInterval().Milliseconds()),
	}

	return &Message{
		Type: MessageTypeState,
		Data: state,
	}
}

// pieceToData converts a piece to PieceData
func pieceToData(p *piece.Piece) PieceData {
	if p == nil {
		return PieceData{}
	}
	return PieceData{
		Type:     p.Type,
		Color:    p.Color,
		X:        p.X,
		Y:        p.Y,
		Rotation: p.Rotation,
	}
}

// NewErrorMessage creates an error message
func NewErrorMessage(err string, code int) *Message {
	return &Message{
		Type: MessageTypeError,
		Data: ErrorMessage{
			Error: err,
			Code:  code,
		},
	}
}

// NewPingMessage creates a ping message
func NewPingMessage(timestamp int64) *Message {
	return &Message{
		Type: MessageTypePing,
		Data: PingMessage{Timestamp: timestamp},
	}
}

// NewGameOverMessage creates a game over message
func NewGameOverMessage(g *game.Game) *Message {
	return &Message{
		Type: MessageTypeGameOver,
		Data: GameOverMessage{
			Score: g.GetScore(),
			Level: g.GetLevel(),
			Lines: g.GetLines(),
		},
	}
}

// ParseControlMessage parses a control message from JSON
func ParseControlMessage(data []byte) (MessageType, error) {
	var msg ControlMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", fmt.Errorf("invalid message format: %w", err)
	}

	if msg.Type == "" {
		return "", fmt.Errorf("missing message type")
	}

	return msg.Type, nil
}

// Serialize converts a message to JSON bytes
func (m *Message) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

// DeserializeMessage parses a message from JSON bytes
func DeserializeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("invalid message format: %w", err)
	}

	if msg.Type == "" {
		return nil, fmt.Errorf("missing message type")
	}

	return &msg, nil
}

// IsValidControlType checks if a message type is a valid control command
func IsValidControlType(t MessageType) bool {
	switch t {
	case MessageTypeMoveLeft, MessageTypeMoveRight, MessageTypeMoveDown,
		MessageTypeRotate, MessageTypeHardDrop, MessageTypePause, MessageTypeResume, MessageTypePong:
		return true
	default:
		return false
	}
}
