package parsers

import (
	"fmt"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

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
			// Skip rows that don't look like player rows (e.g. headers)
			continue
		}

		// Validate and pad hole count
		if len(scores) < numHoles {
			for len(scores) < numHoles {
				scores = append(scores, 0)
			}
		} else if len(scores) > numHoles {
			scores = scores[:numHoles]
		}

		players = append(players, roundtypes.PlayerScoreRow{
			PlayerName: playerName,
			HoleScores: scores,
		})
	}

	return players, nil
}

// parsePlayerScoresXLSX is a wrapper for extractPlayerScoresXLSX to match the signature used in xlsx_core.go
func parsePlayerScoresXLSX(rows [][]string, parRowIndex int, parScores []int) ([]roundtypes.PlayerScoreRow, error) {
	return extractPlayerScoresXLSX(rows, parRowIndex, len(parScores))
}
