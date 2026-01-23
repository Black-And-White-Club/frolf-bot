package parsers

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// findColumn searches for a column by multiple possible names (case-insensitive)
// Removes spaces, underscores, and hyphens for normalization
func findColumn(header []string, possibleNames []string) int {
	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		colNorm := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(colLower, " ", ""), "_", ""), "-", "")

		for _, name := range possibleNames {
			nameLower := strings.ToLower(name)
			nameNorm := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(nameLower, " ", ""), "_", ""), "-", "")

			if colLower == nameLower || colNorm == nameNorm {
				return i
			}
		}
	}
	return -1
}

// findHoleColumns finds all columns that represent holes (for logging only)
// Matches patterns: "hole1", "hole_1", "hole 1", "h1", "H1", or just "1"
func findHoleColumns(header []string) []int {
	var holeColumns []int

	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		colNorm := strings.ReplaceAll(colLower, " ", "")

		// Matches "hole1", "hole_1", "hole 1"
		if strings.HasPrefix(colNorm, "hole") {
			holeNumStr := strings.TrimPrefix(colNorm, "hole")
			holeNumStr = strings.TrimPrefix(holeNumStr, "_")
			if _, err := strconv.Atoi(holeNumStr); err == nil {
				holeColumns = append(holeColumns, i)
			}
		} else if strings.HasPrefix(colNorm, "h") && len(colNorm) > 1 {
			// Matches "h1", "h2", etc.
			holeNumStr := strings.TrimPrefix(colNorm, "h")
			holeNumStr = strings.TrimPrefix(holeNumStr, "_")
			if _, err := strconv.Atoi(holeNumStr); err == nil {
				holeColumns = append(holeColumns, i)
			}
		} else if _, err := strconv.Atoi(strings.TrimSpace(col)); err == nil {
			// Matches just "1", "2", etc.
			holeColumns = append(holeColumns, i)
		}
	}

	return holeColumns
}

// isPARRow checks if a row represents par values
func isPARRow(cellValue string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(cellValue))
	return normalized == "PAR" || normalized == "PARS" || normalized == "P"
}

// extractScores extracts scores from specific columns (for logging - may have gaps)
func extractScores(row []string, columnIndices []int) []int {
	if len(columnIndices) == 0 {
		return nil
	}

	scores := make([]int, 0, len(columnIndices))

	for _, idx := range columnIndices {
		if idx >= len(row) {
			continue
		}

		val := strings.TrimSpace(row[idx])
		if val == "" {
			continue
		}

		score, err := strconv.Atoi(val)
		if err != nil {
			continue
		}

		// Sanity check: disc golf scores are typically 1-15
		if score > 0 && score < 20 {
			scores = append(scores, score)
		}
	}

	if len(scores) == 0 {
		return nil
	}

	return scores
}

// preprocessCSVData cleans CSV data and auto-detects delimiter
// Returns: cleaned string, delimiter rune, error
func preprocessCSVData(data []byte) (string, rune, error) {
	if len(data) == 0 {
		return "", ',', fmt.Errorf("empty CSV data")
	}

	// Strip UTF-8 BOM if present (0xEF, 0xBB, 0xBF)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Replace all \r\n with \n
	cleaned := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	cleanedStr := string(cleaned)

	// Auto-detect delimiter: count commas vs tabs in first 5 lines
	lines := strings.Split(cleanedStr, "\n")
	sampleSize := 5
	if len(lines) < sampleSize {
		sampleSize = len(lines)
	}

	commaCount := 0
	tabCount := 0
	for i := 0; i < sampleSize; i++ {
		commaCount += strings.Count(lines[i], ",")
		tabCount += strings.Count(lines[i], "\t")
	}

	delimiter := ','
	if tabCount > commaCount {
		delimiter = '\t'
	}

	return cleanedStr, delimiter, nil
}

// detectHeaderRow scans the first 5 rows to find the header
// Returns the index of the header row, or -1 if not found
func detectHeaderRow(rows [][]string) int {
	if len(rows) == 0 {
		return -1
	}

	// Scan up to first 5 rows
	maxRows := 5
	if len(rows) < maxRows {
		maxRows = len(rows)
	}

	// Known header column names (normalized)
	knownColumns := []string{
		"playername", "player", "hole1", "1", "+/-", "plusminus",
		"total", "score", "relative", "name", "h1",
	}

	bestScore := 0
	bestRow := -1

	for rowIdx := 0; rowIdx < maxRows; rowIdx++ {
		row := rows[rowIdx]
		score := 0

		for _, cell := range row {
			cellNorm := strings.ToLower(strings.TrimSpace(cell))
			cellNorm = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(cellNorm, " ", ""), "_", ""), "-", "")

			for _, known := range knownColumns {
				knownNorm := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(known, " ", ""), "_", ""), "-", "")
				if cellNorm == knownNorm {
					score++
					break
				}
			}
		}

		// Need at least 2 recognized columns to consider it a header
		if score >= 2 && score > bestScore {
			bestScore = score
			bestRow = rowIdx
		}
	}

	return bestRow
}

// SplitPlayerNames attempts to split a player/team name into individual player names.
// It handles common separators used on UDisc scorecards like "+", "&", "/", and the word "and".
func SplitPlayerNames(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Normalize common separators to a single delimiter
	normalized := s
	// word " and " -> +
	normalized = strings.ReplaceAll(normalized, " and ", " + ")
	// ampersand variants
	normalized = strings.ReplaceAll(normalized, "&", " + ")
	// slashes
	normalized = strings.ReplaceAll(normalized, "/", " + ")
	// commas sometimes separate names
	normalized = strings.ReplaceAll(normalized, ",", " + ")

	parts := strings.Split(normalized, "+")
	var out []string
	for _, p := range parts {
		n := strings.TrimSpace(p)
		if n == "" {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}
