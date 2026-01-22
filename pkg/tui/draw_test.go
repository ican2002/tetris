package tui

import (
	"testing"

	"github.com/ican2002/tetris/pkg/piece"
	"github.com/ican2002/tetris/pkg/protocol"
)

// TestGetPieceShape verifies that getPieceShape correctly handles all piece types
// including TypeI which has value 0
func TestGetPieceShape(t *testing.T) {
	tests := []struct {
		name     string
		pieceType piece.Type
	}{
		{"TypeI", piece.TypeI}, // TypeI = 0, has 1 row
		{"TypeO", piece.TypeO},
		{"TypeT", piece.TypeT},
		{"TypeS", piece.TypeS},
		{"TypeZ", piece.TypeZ},
		{"TypeJ", piece.TypeJ},
		{"TypeL", piece.TypeL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pieceData := protocol.PieceData{
				Type:     tt.pieceType,
				Color:    piece.Color("#00FFFF"),
				X:        0,
				Y:        0,
				Rotation: 0,
			}

			shape := getPieceShape(pieceData)

			if shape == nil {
				t.Errorf("getPieceShape() returned nil for %v (value=%d)", tt.pieceType, tt.pieceType)
			}
		})
	}
}

// TestGetPieceShapeInvalid verifies that getPieceShape handles invalid piece types
func TestGetPieceShapeInvalid(t *testing.T) {
	// Use an invalid piece type (not one of the 7 valid types)
	pieceData := protocol.PieceData{
		Type:     piece.Type(99), // Invalid type
		Color:    "",
		X:        0,
		Y:        0,
		Rotation: 0,
	}

	shape := getPieceShape(pieceData)

	if shape != nil {
		t.Errorf("getPieceShape() should return nil for invalid piece type, got %v", shape)
	}
}

// TestTypeIZeroValue documents the bug: TypeI has value 0, which is the zero value
// for piece.Type. Using "!= 0" to check validity will incorrectly reject TypeI.
func TestTypeIZeroValue(t *testing.T) {
	// This test documents that TypeI equals 0 (the zero value)
	if piece.TypeI != 0 {
		t.Errorf("TypeI should be 0, got %d", piece.TypeI)
	}

	// The bug in DrawBoard and DrawPiecePreview:
	// They use "if currentPiece.Type != 0" to check validity
	// This will skip TypeI pieces since TypeI = 0

	// Correct validation should check against valid types:
	validTypes := map[piece.Type]bool{
		piece.TypeI: true,
		piece.TypeO: true,
		piece.TypeT: true,
		piece.TypeS: true,
		piece.TypeZ: true,
		piece.TypeJ: true,
		piece.TypeL: true,
	}

	if !validTypes[piece.TypeI] {
		t.Error("TypeI should be a valid type")
	}
}

// TestIsValidPieceType verifies that isValidPieceType correctly identifies valid types
func TestIsValidPieceType(t *testing.T) {
	validTypes := []piece.Type{
		piece.TypeI, // TypeI = 0
		piece.TypeO,
		piece.TypeT,
		piece.TypeS,
		piece.TypeZ,
		piece.TypeJ,
		piece.TypeL,
	}

	invalidTypes := []piece.Type{
		piece.Type(99),
		piece.Type(-1),
		piece.Type(100),
	}

	for _, ttype := range validTypes {
		if !isValidPieceType(ttype) {
			t.Errorf("Type %v should be valid", ttype)
		}
	}

	for _, ttype := range invalidTypes {
		if isValidPieceType(ttype) {
			t.Errorf("Type %v should be invalid", ttype)
		}
	}
}
