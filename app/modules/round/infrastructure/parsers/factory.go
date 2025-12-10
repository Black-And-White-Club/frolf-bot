package parsers

import (
	"fmt"
	"strings"
)

// Factory creates the appropriate parser based on file extension.
type Factory struct{}

// NewFactory creates a new parser factory.
func NewFactory() *Factory {
	return &Factory{}
}

// GetParser returns a parser for the given file name.
func (f *Factory) GetParser(fileName string) (Parser, error) {
	fileName = strings.ToLower(fileName)

	if strings.HasSuffix(fileName, ".csv") {
		return NewCSVParser(), nil
	}

	if strings.HasSuffix(fileName, ".xlsx") || strings.HasSuffix(fileName, ".xls") {
		return NewXLSXParser(), nil
	}

	return nil, fmt.Errorf("unsupported file type: %s (must be .csv or .xlsx)", fileName)
}
