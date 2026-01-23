package parsers

import (
	"strconv"
	"strings"
)

// findColumn searches for a column by multiple possible names (case-insensitive)
func findColumn(header []string, possibleNames []string) int {
	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		colNorm := strings.ReplaceAll(strings.ReplaceAll(colLower, " ", ""), "_", "")

		for _, name := range possibleNames {
			nameLower := strings.ToLower(name)
			nameNorm := strings.ReplaceAll(strings.ReplaceAll(nameLower, " ", ""), "_", "")

			if colLower == nameLower || colNorm == nameNorm {
				return i
			}
		}
	}
	return -1
}

// findHoleColumns finds all columns that represent holes (for logging only)
func findHoleColumns(header []string) []int {
	var holeColumns []int

	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		colNorm := strings.ReplaceAll(colLower, " ", "")

		// Matches "hole1", "hole_1", "hole 1", or just "1"
		if strings.HasPrefix(colNorm, "hole") {
			holeNumStr := strings.TrimPrefix(colNorm, "hole")
			holeNumStr = strings.TrimPrefix(holeNumStr, "_")
			if _, err := strconv.Atoi(holeNumStr); err == nil {
				holeColumns = append(holeColumns, i)
			}
		} else if _, err := strconv.Atoi(strings.TrimSpace(col)); err == nil {
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
