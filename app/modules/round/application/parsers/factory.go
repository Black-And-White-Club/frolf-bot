package parsers

import (
	"fmt"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// Parser defines the interface for scorecard parsers
type Parser interface {
	Parse(data []byte) (*roundtypes.ParsedScorecard, error)
}

// ParserFactory defines the interface for creating parsers
type ParserFactory interface {
	GetParser(filename string) (Parser, error)
}

// Factory creates the appropriate parser based on file extension
type Factory struct{}

// NewFactory creates a new parser factory
func NewFactory() *Factory {
	return &Factory{}
}

// GetParser returns the appropriate parser for the given filename
func (f *Factory) GetParser(filename string) (Parser, error) {
	ext := strings.ToLower(getFileExtension(filename))

	switch ext {
	case ".csv":
		return NewCSVParser(), nil
	case ".xlsx", ".xls":
		return NewXLSXParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// getFileExtension extracts the file extension from a filename
func getFileExtension(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return filename[idx:]
}
