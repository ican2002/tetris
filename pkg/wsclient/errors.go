package wsclient

import "errors"

var (
	// ErrNotConnected is returned when trying to send data while disconnected
	ErrNotConnected = errors.New("websocket client is not connected")
)
