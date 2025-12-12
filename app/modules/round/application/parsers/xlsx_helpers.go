package parsers

import (
	"fmt"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// findParRowXLSX identifies the par row and extracts par values.
// Returns: rowIndex, nameColIndex, holeStartColIdx, parScores, error
func findParRowXLSX(rows [][]string) (int, int, int, []int, error) {
	// First, try to detect a UDisc-style header row to find column indices
	headerRowIdx, nameColIdx, holeStartColIdx, layoutType := detectLayout(rows)

	if headerRowIdx != -1 {
		// If it's a Leaderboard (XLSX), it might not have a Par row.
		// We return 0s for ParScores since we can't know the hole pars,
		// but we need to return the correct number of holes so player scores can be parsed.
		if layoutType == "leaderboard" {
			// Count holes starting from holeStartColIdx
			numHoles := 0
			headerRow := rows[headerRowIdx]
			for i := holeStartColIdx; i < len(headerRow); i++ {
				val := strings.ToLower(strings.TrimSpace(headerRow[i]))
				// Check if it looks like a hole header (e.g. "Hole 1", "1", etc.)
				// We assume hole columns are contiguous.
				if strings.HasPrefix(val, "hole") || (len(val) <= 2 && isNumeric(val)) {
					numHoles++
				} else {
					// If we hit a non-hole column, stop counting?
					// In some exports, holes are followed by other data.
					// But usually holes are the last thing or contiguous.
					// If we encounter an empty string, skip it?
					if val == "" {
						continue
					}
					// If it's clearly not a hole (e.g. "Round Rating"), stop.
					if !strings.HasPrefix(val, "hole") && !isNumeric(val) {
						break
					}
					// If it's numeric but not a hole? (e.g. "1000" rating).
					// "1" to "18" are holes. "1000" is not.
					if isNumeric(val) {
						n, _ := strconv.Atoi(val)
						if n > 100 { // Arbitrary cutoff for hole number
							break
						}
					}
					numHoles++
				}
			}

			// If we found holes, return 0s for par (unknown)
			if numHoles > 0 {
				parScores := make([]int, numHoles)
				// Return headerRowIdx as parRowIndex so it gets skipped during player parsing
				return headerRowIdx, nameColIdx, holeStartColIdx, parScores, nil
			}
		}

		// For Scorecard layout, look for the Par row relative to this layout.
		// The Par row usually has "Par" in the name column.
		for i := headerRowIdx + 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) <= nameColIdx {
				continue
			}

			nameVal := strings.TrimSpace(row[nameColIdx])
			if strings.Contains(strings.ToLower(nameVal), "par") {
				// Found Par row. Parse scores starting from holeStartColIdx.
				if len(row) <= holeStartColIdx {
					return -1, -1, -1, nil, fmt.Errorf("par row too short")
				}

				parScores, err := parseScoreRowXLSX(row[holeStartColIdx:])
				if err != nil {
					return -1, -1, -1, nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
				}
				return i, nameColIdx, holeStartColIdx, parScores, nil
			}
		}

		// If no labeled Par row found, look for a numeric row that matches the hole count/structure
		// This handles cases where the par row exists but isn't labeled "Par" (e.g. just numbers)
		for i := headerRowIdx + 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) <= holeStartColIdx {
				continue
			}

			// Check if the "name" column is NOT a likely player name (e.g. it's a number or empty)
			// If it looks like a player name, we shouldn't treat it as a par row
			nameVal := strings.TrimSpace(row[nameColIdx])
			if isLikelyPlayerNameXLSX(nameVal) {
				continue
			}

			// Check if the rest are scores
			parScores, err := parseScoreRowXLSX(row[holeStartColIdx:])
			if err == nil && len(parScores) > 0 {
				return i, nameColIdx, holeStartColIdx, parScores, nil
			}
		}
	}

	// Fallback: Use the old heuristic search if no clear header was found
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}

		// Find first non-empty column
		firstColIdx := -1
		for c, val := range row {
			if strings.TrimSpace(val) != "" {
				firstColIdx = c
				break
			}
		}

		if firstColIdx == -1 {
			continue // Empty row
		}

		firstVal := strings.TrimSpace(row[firstColIdx])

		// Check if first non-empty column is "Par" (case-insensitive)
		if strings.Contains(strings.ToLower(firstVal), "par") {
			// Extract numeric values from remaining columns
			parScores, err := parseScoreRowXLSX(row[firstColIdx+1:])
			if err != nil {
				// If parsing fails immediately after "Par", it might be because of intermediate columns (UDisc style)
				// Try skipping columns until we find numbers?
				// For now, just return error as before, unless we want to be smarter here too.
				return -1, -1, -1, nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
			}
			return i, firstColIdx, firstColIdx + 1, parScores, nil
		}

		// Alternative: check if entire row is numeric (all scores)
		parScores, err := parseScoreRowXLSX(row[firstColIdx:])
		if err == nil && len(parScores) >= 9 {
			if !isLikelyPlayerNameXLSX(firstVal) {
				return i, firstColIdx, firstColIdx, parScores, nil
			}
		}
	}

	return -1, -1, -1, nil, nil
}

// detectLayout attempts to find the header row and column indices.
// Returns: headerRowIndex, nameColIndex, holeStartColIndex, layoutType
// Returns -1s if not found.
func detectLayout(rows [][]string) (int, int, int, string) {
	for i, row := range rows {
		nameIdx := -1
		usernameIdx := -1
		holeStartIdx := -1
		isLeaderboard := false

		for c, val := range row {
			val = strings.ToLower(strings.TrimSpace(val))

			// Check for Scorecard headers
			if nameIdx == -1 && (val == "playername" || val == "name" || val == "player") {
				nameIdx = c
			}

			// Check for Username header (preferred for Leaderboards)
			if usernameIdx == -1 && (val == "username" || val == "user name") {
				usernameIdx = c
			}

			// Check for Leaderboard headers
			if val == "division" || val == "position" {
				isLeaderboard = true
			}

			// Check for Hole 1
			if holeStartIdx == -1 && (val == "hole1" || val == "hole 1" || val == "hole_1" || val == "1") {
				holeStartIdx = c
			}
		}

		// If it's a leaderboard and we found a username column, use that as the name source
		if isLeaderboard && usernameIdx != -1 {
			nameIdx = usernameIdx
		}

		if nameIdx != -1 && holeStartIdx != -1 && holeStartIdx > nameIdx {
			layout := "scorecard"
			if isLeaderboard {
				layout = "leaderboard"
			}
			return i, nameIdx, holeStartIdx, layout
		}
	}
	return -1, -1, -1, ""
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func isLikelyPlayerNameXLSX(s string) bool {
	s = strings.TrimSpace(s)
	_, err := strconv.Atoi(s)
	return err != nil
}

// parseScoreRowXLSX converts string values to integers
// It stops parsing when it encounters a non-numeric value after finding at least one score,
// to handle trailing columns like "Total" or "+/-".
func parseScoreRowXLSX(row []string) ([]int, error) {
	var scores []int
	for _, val := range row {
		val = strings.TrimSpace(val)
		if val == "" || val == "-" {
			continue
		}
		score, err := strconv.Atoi(val)
		if err != nil {
			// If we hit a non-numeric value
			if len(scores) > 0 {
				// We already have some scores, assume this is the start of summary columns (Total, +/-)
				// Stop parsing and return what we have
				break
			}
			// If we haven't found any scores yet, this is an error (garbage in score columns)
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
func extractPlayerScoresXLSX(rows [][]string, parRowIndex int, nameColIndex int, holeStartColIdx int, numHoles int) ([]roundtypes.PlayerScoreRow, error) {
	var players []roundtypes.PlayerScoreRow

	for i, row := range rows {
		// Skip par row
		if i == parRowIndex || i == 0 {
			continue
		}

		if len(row) <= nameColIndex {
			continue
		}

		// Player name is at the same column index as "Par"
		playerName := strings.TrimSpace(row[nameColIndex])
		if playerName == "" {
			continue
		}

		// Parse scores starting at holeStartColIdx
		if len(row) <= holeStartColIdx {
			continue
		}
		scores, err := parseScoreRowXLSX(row[holeStartColIdx:])
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

	if len(players) == 0 {
		return nil, fmt.Errorf("no player scores found")
	}

	return players, nil
}

// parsePlayerScoresXLSX is a wrapper for extractPlayerScoresXLSX to match the signature used in xlsx_core.go
func parsePlayerScoresXLSX(rows [][]string, parRowIndex int, nameColIndex int, holeStartColIdx int, parScores []int) ([]roundtypes.PlayerScoreRow, error) {
	return extractPlayerScoresXLSX(rows, parRowIndex, nameColIndex, holeStartColIdx, len(parScores))
}
