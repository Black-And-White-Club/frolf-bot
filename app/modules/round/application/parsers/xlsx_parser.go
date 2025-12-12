package parsers

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/xuri/excelize/v2"
)

// XLSXParser parses XLSX scorecard files
type XLSXParser struct{}

// NewXLSXParser creates a new XLSX parser
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

// Parse parses XLSX data and returns a ParsedScorecard
func (p *XLSXParser) Parse(data []byte) (*roundtypes.ParsedScorecard, error) {
	// Create a reader from the byte data
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		if strings.Contains(err.Error(), "zip: not a valid zip file") {
			return nil, fmt.Errorf("failed to open XLSX file: %w. (Hint: If this is a CSV file, please ensure it has a .csv extension)", err)
		}
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
	parRowIndex, parScores, err := findParRowXLSX(rows)
	if err != nil {
		return nil, err
	}

	if parRowIndex < 0 {
		return nil, fmt.Errorf("no par row found in XLSX")
	}

	// Extract player rows
	playerScores, err := extractPlayerScoresXLSX(rows, parRowIndex, len(parScores))
	if err != nil {
		return nil, err
	}

	return &roundtypes.ParsedScorecard{
		ParScores:    parScores,
		PlayerScores: playerScores,
	}, nil
}

// findParRowXLSX identifies the par row and extracts par values
func findParRowXLSX(rows [][]string) (int, []int, error) {
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}

		// Check if first column is "Par" (case-insensitive)
		if strings.EqualFold(strings.TrimSpace(row[0]), "Par") {
			// Extract numeric values from remaining columns
			parScores, err := parseScoreRowXLSX(row[1:])
			if err != nil {
				return -1, nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
			}
			return i, parScores, nil
		}

		// Alternative: check if entire row is numeric (all scores)
		parScores, err := parseScoreRowXLSX(row)
		if err == nil && len(parScores) >= 9 {
			// Assume this is the par row if it has at least 9 numeric values
			if !isLikelyPlayerNameXLSX(row[0]) {
				return i, parScores, nil
			}
		}
	}

	return -1, nil, nil
}

// isLikelyPlayerNameXLSX checks if a string looks like a player name
func isLikelyPlayerNameXLSX(s string) bool {
	s = strings.TrimSpace(s)
	_, err := strconv.Atoi(s)
	return err != nil
}

// parseScoreRowXLSX converts string values to integers
func parseScoreRowXLSX(row []string) ([]int, error) {
	var scores []int
	for _, val := range row {
		val = strings.TrimSpace(val)
		if val == "" || val == "-" {
			continue
		}
		score, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("non-numeric score value: %q", val)
		}
		if score < 0 {
			return nil, fmt.Errorf("negative score value: %d", score)
		}
		scores = append(scores, score)
	}
	return scores, nil
}

// extractPlayerScoresXLSX extracts all player score rows from XLSX
func extractPlayerScoresXLSX(rows [][]string, parRowIndex int, numHoles int) ([]roundtypes.PlayerScoreRow, error) {
	var players []roundtypes.PlayerScoreRow

	for i, row := range rows {
		// Skip par row
		if i == parRowIndex || i == 0 {
			continue
		}

		if len(row) == 0 {
			continue
		}

		// First column is player name
		playerName := strings.TrimSpace(row[0])
		if playerName == "" {
			continue
		}

		// Parse scores
		scores, err := parseScoreRowXLSX(row[1:])
		if err != nil {
			return nil, fmt.Errorf("invalid scores for player %q at line %d: %w", playerName, i+1, err)
		}

		// Validate and pad hole count
		if len(scores) < numHoles {
			for len(scores) < numHoles {
				scores = append(scores, 0)
			}
		} else if len(scores) > numHoles {
			scores = scores[:numHoles]
		}

		// Calculate total
		total := 0
		for _, score := range scores {
			total += score
		}

		players = append(players, roundtypes.PlayerScoreRow{
			PlayerName: playerName,
			HoleScores: scores,
			Total:      total,
		})
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("no player score rows found in XLSX")
	}

	return players, nil
}
