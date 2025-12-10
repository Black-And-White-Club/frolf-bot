package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// CSVParser parses CSV scorecard files
type CSVParser struct{}

// NewCSVParser creates a new CSV parser
func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// Parse parses CSV data and returns a ParsedScorecard
func (p *CSVParser) Parse(data []byte) (*roundtypes.ParsedScorecard, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))

	var records [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV: %w", err)
		}
		// Skip empty rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Find par row and validate structure
	parRowIndex, parScores, err := findParRow(records)
	if err != nil {
		return nil, err
	}

	if parRowIndex < 0 {
		return nil, fmt.Errorf("no par row found in CSV")
	}

	// Extract player rows (skip header and par row)
	playerScores, err := extractPlayerScores(records, parRowIndex, len(parScores))
	if err != nil {
		return nil, err
	}

	return &roundtypes.ParsedScorecard{
		ParScores:    parScores,
		PlayerScores: playerScores,
	}, nil
}

// findParRow identifies the par row and extracts par values
func findParRow(records [][]string) (int, []int, error) {
	for i, record := range records {
		if len(record) == 0 {
			continue
		}

		// Check if first column is "Par" (case-insensitive)
		if strings.EqualFold(strings.TrimSpace(record[0]), "Par") {
			// Extract numeric values from remaining columns
			parScores, err := parseScoreRow(record[1:])
			if err != nil {
				return -1, nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
			}
			return i, parScores, nil
		}

		// Alternative: check if entire row is numeric (all scores)
		parScores, err := parseScoreRow(record)
		if err == nil && len(parScores) >= 9 {
			// Assume this is the par row if it has at least 9 numeric values
			// and the first column isn't a player name
			if !isLikelyPlayerName(record[0]) {
				return i, parScores, nil
			}
		}
	}

	return -1, nil, nil
}

// isLikelyPlayerName checks if a string looks like a player name
func isLikelyPlayerName(s string) bool {
	s = strings.TrimSpace(s)
	// Player names typically have letters and spaces, not just numbers
	_, err := strconv.Atoi(s)
	return err != nil // If it's not a number, it's likely a name
}

// parseScoreRow converts string values to integers, skipping non-numeric values
func parseScoreRow(record []string) ([]int, error) {
	var scores []int
	for _, val := range record {
		val = strings.TrimSpace(val)
		if val == "" || val == "-" {
			// Skip empty or dash values
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

// extractPlayerScores extracts all player score rows
func extractPlayerScores(records [][]string, parRowIndex int, numHoles int) ([]roundtypes.PlayerScoreRow, error) {
	var players []roundtypes.PlayerScoreRow

	for i, record := range records {
		// Skip par row and header rows
		if i == parRowIndex || i == 0 {
			continue
		}

		if len(record) == 0 {
			continue
		}

		// First column is player name
		playerName := strings.TrimSpace(record[0])
		if playerName == "" {
			continue
		}

		// Parse scores
		scores, err := parseScoreRow(record[1:])
		if err != nil {
			return nil, fmt.Errorf("invalid scores for player %q at line %d: %w", playerName, i+1, err)
		}

		// Validate hole count
		if len(scores) < numHoles {
			// Allow short hole counts (player didn't play all holes)
			// but pad with zeros for missing holes
			for len(scores) < numHoles {
				scores = append(scores, 0)
			}
		} else if len(scores) > numHoles {
			// If more scores than holes, the last score might be a total
			// Try to use first numHoles as hole scores
			scores = scores[:numHoles]
		}

		// Calculate total
		total := 0
		for _, score := range scores {
			total += score
		}

		// Check if the record has an explicit total column
		if len(record) > numHoles+1 {
			lastCol := strings.TrimSpace(record[len(record)-1])
			if providedTotal, err := strconv.Atoi(lastCol); err == nil {
				// Validate provided total matches calculated
				if providedTotal != total && providedTotal > 0 {
					// Log warning but don't fail; use calculated total
					// fmt.Printf("Warning: player %s total mismatch: provided=%d, calculated=%d\n", playerName, providedTotal, total)
				}
			}
		}

		players = append(players, roundtypes.PlayerScoreRow{
			PlayerName: playerName,
			HoleScores: scores,
			Total:      total,
		})
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("no player score rows found in CSV")
	}

	return players, nil
}
