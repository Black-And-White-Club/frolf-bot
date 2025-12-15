package parsers

import (
	"fmt"
	"strconv"
	"strings"
)

// Factory creates the appropriate parser based on file extension.
type Factory struct{}

// NewFactory creates a new parser factory.
func NewFactory() *Factory {
	return &Factory{}
}

// GetParser returns a parser for the given file name.
func (f *Factory) GetParser(fileName string) (Parser, error) {
	fileName = strings.ToLower(fileName)

	if strings.HasSuffix(fileName, ".csv") {
		return NewCSVParser(), nil
	}

	if strings.HasSuffix(fileName, ".xlsx") || strings.HasSuffix(fileName, ".xls") {
		return NewXLSXParser(), nil
	}

	return nil, fmt.Errorf("unsupported file type: %s (must be .csv or .xlsx)", fileName)
}

// ================ Shared Utilities ================

// findColumn searches for a column by multiple possible names (case-insensitive)
func findColumn(header []string, possibleNames []string) int {
	for i, col := range header {
		colLower := strings.ToLower(strings.TrimSpace(col))
		// Also check normalized version (no spaces/underscores)
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

	// Pre-allocate with capacity but return actual scores found
	scores := make([]int, 0, len(columnIndices))

	for _, idx := range columnIndices {
		if idx >= len(row) {
			// Column exists in header but not in this row - skip
			continue
		}

		val := strings.TrimSpace(row[idx])
		if val == "" {
			// Empty cell - skip (don't add nil/0, just skip)
			continue
		}

		score, err := strconv.Atoi(val)
		if err != nil {
			// Invalid score - skip
			continue
		}

		// Only add valid scores (sanity check: disc golf scores are typically 1-15)
		if score > 0 && score < 20 {
			scores = append(scores, score)
		}
	}

	// Return nil instead of empty slice if no scores found
	if len(scores) == 0 {
		return nil
	}

	return scores
}
