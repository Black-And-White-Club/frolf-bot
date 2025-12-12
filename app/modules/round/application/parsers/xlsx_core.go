package parsers

import (
	"bytes"
	"fmt"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/xuri/excelize/v2"
)

// parseXLSXCore contains the core logic for parsing XLSX data without fallback.
// This is used by both XLSXParser (as primary) and CSVParser (as fallback for misnamed files).
func parseXLSXCore(data []byte) (*roundtypes.ParsedScorecard, error) {
	// Create a reader from the byte data
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX file: %w", err)
	}
	defer f.Close()

	// Get sheet names
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("XLSX file has no sheets")
	}

	// Use the first sheet
	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %q: %w", sheetName, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %q is empty", sheetName)
	}

	// Find par row and validate structure
	parRowIndex, headerRowIndex, nameColIndex, holeStartColIdx, _, parScores, err := findParRowXLSX(rows)
	if err != nil {
		return nil, err
	}
	if len(parScores) == 0 {
		return nil, fmt.Errorf("no par row found in XLSX")
	}

	// Parse player scores
	playerScores, err := parsePlayerScoresXLSX(rows, parRowIndex, headerRowIndex, nameColIndex, holeStartColIdx, parScores)
	if err != nil {
		return nil, err
	}

	return &roundtypes.ParsedScorecard{
		ParScores:    parScores,
		PlayerScores: playerScores,
	}, nil
}
