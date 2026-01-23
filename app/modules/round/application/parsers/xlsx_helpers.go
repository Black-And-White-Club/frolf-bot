package parsers

import (
	"fmt"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// findParRowXLSX identifies the par row and extracts par values.
// Returns: parRowIndex, headerRowIndex, nameColIndex, holeStartColIdx, layoutType, parScores, error
func findParRowXLSX(rows [][]string) (int, int, int, int, string, []int, error) {
	headerRowIdx, nameColIdx, holeStartColIdx, layoutType := detectLayout(rows)

	if headerRowIdx != -1 {
		if layoutType == "leaderboard" {
			// Count holes starting from the detected hole column
			numHoles := 0
			headerRow := rows[headerRowIdx]
			for i := holeStartColIdx; i < len(headerRow); i++ {
				val := strings.ToLower(strings.TrimSpace(headerRow[i]))
				if strings.HasPrefix(val, "hole") || (len(val) <= 2 && isNumeric(val)) {
					numHoles++
					continue
				}
				if val == "" {
					continue
				}
				if !strings.HasPrefix(val, "hole") && !isNumeric(val) {
					break
				}
				if isNumeric(val) {
					n, _ := strconv.Atoi(val)
					if n > 100 {
						break
					}
				}
				numHoles++
			}

			if numHoles > 0 {
				parScores := make([]int, numHoles)
				return headerRowIdx, headerRowIdx, nameColIdx, holeStartColIdx, layoutType, parScores, nil
			}
		}

		for i := headerRowIdx + 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) <= nameColIdx {
				continue
			}

			nameVal := strings.TrimSpace(row[nameColIdx])
			if strings.Contains(strings.ToLower(nameVal), "par") {
				if len(row) <= holeStartColIdx {
					return -1, -1, -1, -1, "", nil, fmt.Errorf("par row too short")
				}

				parScores, err := parseScoreRowXLSX(row[holeStartColIdx:])
				if err != nil {
					return -1, -1, -1, -1, "", nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
				}
				return i, headerRowIdx, nameColIdx, holeStartColIdx, layoutType, parScores, nil
			}
		}

		for i := headerRowIdx + 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) <= holeStartColIdx {
				continue
			}

			nameVal := strings.TrimSpace(row[nameColIdx])
			if isLikelyPlayerNameXLSX(nameVal) {
				continue
			}

			parScores, err := parseScoreRowXLSX(row[holeStartColIdx:])
			if err == nil && len(parScores) > 0 {
				return i, headerRowIdx, nameColIdx, holeStartColIdx, layoutType, parScores, nil
			}
		}
	}

	// Fallback: Use the old heuristic search if no clear header was found
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}

		firstColIdx := -1
		for c, val := range row {
			if strings.TrimSpace(val) != "" {
				firstColIdx = c
				break
			}
		}

		if firstColIdx == -1 {
			continue
		}

		firstVal := strings.TrimSpace(row[firstColIdx])
		if strings.Contains(strings.ToLower(firstVal), "par") {
			parScores, err := parseScoreRowXLSX(row[firstColIdx+1:])
			if err != nil {
				return -1, -1, -1, -1, "", nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
			}
			return i, -1, firstColIdx, firstColIdx + 1, "scorecard", parScores, nil
		}

		parScores, err := parseScoreRowXLSX(row[firstColIdx:])
		if err == nil && len(parScores) >= 9 && !isLikelyPlayerNameXLSX(firstVal) {
			return i, -1, firstColIdx, firstColIdx, "scorecard", parScores, nil
		}
	}

	return -1, -1, -1, -1, "", nil, nil
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

			// Check for Hole 1 (supports "hole1", "hole 1", "hole_1", "h1", "H1", "1")
			if holeStartIdx == -1 && (val == "hole1" || val == "hole 1" || val == "hole_1" || val == "h1" || val == "1") {
				holeStartIdx = c
			}
		}

		// Prefer username column if it exists and sits before the hole columns
		if usernameIdx != -1 && holeStartIdx != -1 && holeStartIdx > usernameIdx {
			nameIdx = usernameIdx
			isLeaderboard = true
		} else if isLeaderboard && usernameIdx != -1 {
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
func extractPlayerScoresXLSX(rows [][]string, parRowIndex int, headerRowIndex int, nameColIndex int, holeStartColIdx int, numHoles int) ([]roundtypes.PlayerScoreRow, error) {
	var players []roundtypes.PlayerScoreRow

	// Detect relative score column
	relativeScoreColIndex := detectRelativeScoreColumnXLSX(rows, nameColIndex)

	// Get par scores if available
	var parScores []int
	if parRowIndex >= 0 && parRowIndex < len(rows) {
		if holeStartColIdx >= 0 && holeStartColIdx < len(rows[parRowIndex]) {
			parScores, _ = parseScoreRowXLSX(rows[parRowIndex][holeStartColIdx:])
		}
	}

	for i, row := range rows {
		// Skip headers and par rows
		if i == parRowIndex || (headerRowIndex >= 0 && i == headerRowIndex) || i == 0 {
			continue
		}

		if len(row) <= nameColIndex {
			continue
		}

		playerName := strings.TrimSpace(row[nameColIndex])
		if playerName == "" {
			continue
		}

		// Parse scores
		if len(row) <= holeStartColIdx {
			continue
		}
		scores, err := parseScoreRowXLSX(row[holeStartColIdx:])
		if err != nil {
			continue // Skip invalid rows
		}

		// Validate hole count
		if len(scores) < numHoles {
			for len(scores) < numHoles {
				scores = append(scores, 0)
			}
		} else if len(scores) > numHoles {
			scores = scores[:numHoles]
		}

		// Calculate Total
		total := 0
		if relativeScoreColIndex != -1 && relativeScoreColIndex < len(row) {
			relativeScoreStr := strings.TrimSpace(row[relativeScoreColIndex])
			if relativeScoreStr != "" && relativeScoreStr != "-" {
				relativeScoreStr = strings.TrimPrefix(relativeScoreStr, "+")
				if val, err := strconv.Atoi(relativeScoreStr); err == nil {
					total = val
				} else {
					total = calculateRelativeScoreXLSX(scores, parScores)
				}
			} else {
				total = calculateRelativeScoreXLSX(scores, parScores)
			}
		} else {
			total = calculateRelativeScoreXLSX(scores, parScores)
		}

		// =========================================================
		// DOUBLES DETECTION LOGIC
		// =========================================================
		teamNames := SplitPlayerNames(playerName)
		isTeam := len(teamNames) > 1

		players = append(players, roundtypes.PlayerScoreRow{
			PlayerName: playerName, // Keep raw "Alec + Jess"
			HoleScores: scores,
			Total:      total,
			IsTeam:     isTeam,    // Flag this row
			TeamNames:  teamNames, // Store the split names for the service
		})
	}

	if len(players) == 0 {
		return nil, fmt.Errorf("no player scores found")
	}

	return players, nil
}

// parsePlayerScoresXLSX is a wrapper for extractPlayerScoresXLSX to match the signature used in xlsx_core.go
func parsePlayerScoresXLSX(rows [][]string, parRowIndex int, headerRowIndex int, nameColIndex int, holeStartColIdx int, parScores []int) ([]roundtypes.PlayerScoreRow, error) {
	return extractPlayerScoresXLSX(rows, parRowIndex, headerRowIndex, nameColIndex, holeStartColIdx, len(parScores))
}

// detectRelativeScoreColumnXLSX looks for a relative score column ("+/-", "round_relative_score", etc.)
// in the header row or the first few rows of an XLSX sheet.
func detectRelativeScoreColumnXLSX(rows [][]string, nameColIndex int) int {
	// Expanded list of possible column names
	possibleNames := []string{
		"+/-", "plusminus", "plus minus", "relative", "relative_score",
		"round_relative_score", "to_par", "topar", "net", "total", "score",
	}

	// Look for a header row (usually row 0 or close to it)
	for i := 0; i < len(rows) && i < 3; i++ {
		row := rows[i]
		for j, cell := range row {
			cellLower := strings.ToLower(strings.TrimSpace(cell))
			// Normalize: remove spaces, underscores, hyphens
			cellNorm := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(cellLower, " ", ""), "_", ""), "-", "")

			for _, name := range possibleNames {
				nameLower := strings.ToLower(name)
				nameNorm := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(nameLower, " ", ""), "_", ""), "-", "")

				if cellLower == nameLower || cellNorm == nameNorm {
					return j
				}
			}
		}
	}
	return -1
}

// calculateRelativeScoreXLSX calculates the relative score (to par) from hole scores and par
func calculateRelativeScoreXLSX(holeScores []int, parScores []int) int {
	if len(holeScores) == 0 {
		return 0
	}

	// Calculate strokes
	totalStrokes := 0
	for _, s := range holeScores {
		totalStrokes += s
	}

	// Calculate par if available
	if len(parScores) > 0 {
		totalPar := 0
		for _, p := range parScores {
			totalPar += p
		}
		// Relative score is strokes - par
		return totalStrokes - totalPar
	}

	// If no par available, return the sum (not ideal, but a fallback)
	return totalStrokes
}
