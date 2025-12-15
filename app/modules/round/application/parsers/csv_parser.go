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
	// Check for XLSX signature (PK\x03\x04) to handle XLSX files incorrectly named as .csv
	if len(data) > 4 && string(data[:4]) == "PK\x03\x04" {
		parsed, err := parseXLSXCore(data)
		if err != nil {
			return nil, fmt.Errorf("file appears to be XLSX (zip signature) but failed to parse: %w", err)
		}
		return parsed, nil
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1 // Allow variable number of fields

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
	parRowIndex, nameColIndex, parScores, err := findParRow(records)
	if err != nil {
		return nil, err
	}

	if parRowIndex < 0 {
		return nil, fmt.Errorf("no par row found in CSV")
	}

	// Detect relative score column (for disc golf "+/-" scores)
	relativeScoreColIndex := detectRelativeScoreColumn(records, nameColIndex)

	// Extract player rows (skip header and par row)
	playerScores, err := extractPlayerScoresWithRelative(records, parRowIndex, nameColIndex, len(parScores), parScores, relativeScoreColIndex)
	if err != nil {
		return nil, err
	}

	return &roundtypes.ParsedScorecard{
		ParScores:    parScores,
		PlayerScores: playerScores,
	}, nil
}

// findParRow identifies the par row and extracts par values
// Returns: rowIndex, colIndex, parScores, error
func findParRow(records [][]string) (int, int, []int, error) {
	// First, try to detect a UDisc-style header row to find column indices
	headerRowIdx, nameColIdx, holeStartColIdx := detectLayoutCSV(records)

	if headerRowIdx != -1 {
		// We found a header. Look for the Par row relative to this layout.
		// The Par row usually has "Par" in the name column.
		for i := headerRowIdx + 1; i < len(records); i++ {
			row := records[i]
			if len(row) <= nameColIdx {
				continue
			}

			nameVal := strings.TrimSpace(row[nameColIdx])
			if strings.Contains(strings.ToLower(nameVal), "par") {
				// Found Par row. Parse scores starting from holeStartColIdx.
				if len(row) <= holeStartColIdx {
					return -1, -1, nil, fmt.Errorf("par row too short")
				}

				parScores, err := parseScoreRow(row[holeStartColIdx:])
				if err != nil {
					return -1, -1, nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
				}
				return i, nameColIdx, parScores, nil
			}
		}
	}

	// Fallback: Use the old heuristic search
	for i, record := range records {
		if len(record) == 0 {
			continue
		}

		// Find first non-empty column
		firstColIdx := -1
		for c, val := range record {
			if strings.TrimSpace(val) != "" {
				firstColIdx = c
				break
			}
		}

		if firstColIdx == -1 {
			continue // Empty row
		}

		firstVal := strings.TrimSpace(record[firstColIdx])

		// Check if first non-empty column is "Par" (case-insensitive)
		if strings.Contains(strings.ToLower(firstVal), "par") {
			// Extract numeric values from remaining columns
			parScores, err := parseScoreRow(record[firstColIdx+1:])
			if err != nil {
				return -1, -1, nil, fmt.Errorf("invalid par row at line %d: %w", i+1, err)
			}
			return i, firstColIdx, parScores, nil
		}

		// Alternative: check if entire row is numeric (all scores)
		parScores, err := parseScoreRow(record[firstColIdx:])
		if err == nil && len(parScores) >= 9 {
			// Assume this is the par row if it has at least 9 numeric values
			if !isLikelyPlayerName(firstVal) {
				return i, firstColIdx, parScores, nil
			}
		}
	}

	return -1, -1, nil, nil
}

// detectLayoutCSV attempts to find the header row and column indices.
// Returns: headerRowIndex, nameColIndex, holeStartColIndex
// Returns -1s if not found.
func detectLayoutCSV(rows [][]string) (int, int, int) {
	for i, row := range rows {
		nameIdx := -1
		holeStartIdx := -1

		for c, val := range row {
			val = strings.ToLower(strings.TrimSpace(val))
			if nameIdx == -1 && (val == "playername" || val == "name" || val == "player") {
				nameIdx = c
			}
			if holeStartIdx == -1 && (val == "hole1" || val == "hole 1" || val == "hole_1" || val == "1") {
				holeStartIdx = c
			}
		}

		if nameIdx != -1 && holeStartIdx != -1 && holeStartIdx > nameIdx {
			return i, nameIdx, holeStartIdx
		}
	}
	return -1, -1, -1
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
			// If we hit a non-numeric value
			if len(scores) > 0 {
				// We already have some scores, assume this is the start of summary columns (Total, +/-)
				// Stop parsing and return what we have
				break
			}
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
func extractPlayerScores(records [][]string, parRowIndex int, nameColIndex int, numHoles int) ([]roundtypes.PlayerScoreRow, error) {
	var players []roundtypes.PlayerScoreRow

	for i, record := range records {
		// Skip par row and header rows
		if i == parRowIndex || i == 0 {
			continue
		}

		if len(record) <= nameColIndex {
			continue
		}

		// Player name is at the same column index as "Par"
		playerName := strings.TrimSpace(record[nameColIndex])
		if playerName == "" {
			continue
		}

		// Parse scores
		if len(record) <= nameColIndex+1 {
			continue
		}
		scores, err := parseScoreRow(record[nameColIndex+1:])
		if err != nil {
			// Skip rows that don't look like player rows (e.g. headers)
			continue
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

// detectRelativeScoreColumn looks for a "+/-" or "round_relative_score" column in the CSV header.
// Returns the column index if found, or -1 if not found.
func detectRelativeScoreColumn(records [][]string, nameColIndex int) int {
	// Look for a header row (usually row 0 or close to it)
	for i := 0; i < len(records) && i < 3; i++ { // Check first few rows for headers
		row := records[i]
		for j, cell := range row {
			cellLower := strings.ToLower(strings.TrimSpace(cell))
			if cellLower == "+/-" || cellLower == "round_relative_score" || cellLower == "relative_score" || cellLower == "to_par" {
				return j
			}
		}
	}
	return -1
}

// extractPlayerScoresWithRelative extracts player scores, attempting to use relative score column if available
func extractPlayerScoresWithRelative(records [][]string, parRowIndex int, nameColIndex int, numHoles int, parScores []int, relativeScoreColIndex int) ([]roundtypes.PlayerScoreRow, error) {
	var players []roundtypes.PlayerScoreRow

	for i, record := range records {
		// Skip par row and header rows
		if i == parRowIndex || i == 0 {
			continue
		}

		if len(record) <= nameColIndex {
			continue
		}

		// Player name is at the same column index as "Par"
		playerName := strings.TrimSpace(record[nameColIndex])
		if playerName == "" {
			continue
		}

		// Parse scores
		if len(record) <= nameColIndex+1 {
			continue
		}
		scores, err := parseScoreRow(record[nameColIndex+1:])
		if err != nil {
			// Skip rows that don't look like player rows (e.g. headers)
			continue
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

		// Try to extract relative score from the dedicated column if available
		total := 0
		if relativeScoreColIndex != -1 && relativeScoreColIndex < len(record) {
			relativeScoreStr := strings.TrimSpace(record[relativeScoreColIndex])
			// Try to parse the relative score directly
			if relativeScoreStr != "" && relativeScoreStr != "-" && relativeScoreStr != "+" {
				// Remove leading + if present
				relativeScoreStr = strings.TrimPrefix(relativeScoreStr, "+")
				if val, err := strconv.Atoi(relativeScoreStr); err == nil {
					total = val
				} else {
					// Fall back to calculating from holes and par
					total = calculateRelativeScore(scores, parScores)
				}
			} else {
				// Fall back to calculating from holes and par
				total = calculateRelativeScore(scores, parScores)
			}
		} else {
			// No relative score column, calculate from holes and par
			total = calculateRelativeScore(scores, parScores)
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

// calculateRelativeScore calculates the relative score (to par) from hole scores and par
func calculateRelativeScore(holeScores []int, parScores []int) int {
	if len(holeScores) == 0 || len(parScores) == 0 {
		return 0
	}

	// Calculate strokes and par
	totalStrokes := 0
	for _, s := range holeScores {
		totalStrokes += s
	}

	totalPar := 0
	for _, p := range parScores {
		totalPar += p
	}

	// Relative score is strokes - par
	return totalStrokes - totalPar
}
