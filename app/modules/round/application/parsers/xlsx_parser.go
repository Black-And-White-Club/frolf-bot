package parsers

import (
	"fmt"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// XLSXParser parses XLSX scorecard files
type XLSXParser struct{}

// NewXLSXParser creates a new XLSX parser
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

// Parse parses XLSX data and returns a ParsedScorecard
func (p *XLSXParser) Parse(data []byte) (*roundtypes.ParsedScorecard, error) {
	// Try to parse as XLSX first using the core logic
	parsed, err := parseXLSXCore(data)
	if err == nil {
		// Determine Mode based on whether we found any teams
		for _, row := range parsed.PlayerScores {
			if row.IsTeam {
				parsed.Mode = sharedtypes.RoundModeDoubles
				break
			}
		}
		if parsed.Mode == "" {
			parsed.Mode = sharedtypes.RoundModeSingles
		}

		return parsed, nil
	}

	// Fallback: Try to parse as CSV if XLSX parsing fails with a zip error
	// This handles cases where a CSV file is incorrectly named with an .xlsx extension
	if strings.Contains(err.Error(), "zip: not a valid zip file") {
		// Check if it looks like a CSV (text file)
		// Simple heuristic: check if it contains commas and newlines, and no null bytes in the first chunk
		isBinary := false
		checkLen := 512
		if len(data) < checkLen {
			checkLen = len(data)
		}
		for _, b := range data[:checkLen] {
			if b == 0 {
				isBinary = true
				break
			}
		}

		if !isBinary {
			csvParser := NewCSVParser()
			parsed, csvErr := csvParser.Parse(data)
			if csvErr == nil {
				return parsed, nil
			}
		}

		// If CSV parsing also fails, return the original error with a hint
		return nil, fmt.Errorf("failed to open XLSX file: %w. (Hint: If this is a CSV file, please ensure it has a .csv extension)", err)
	}

	return nil, err
}
