package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// CSVParser implements the Parser interface for CSV scorecard files.
type CSVParser struct{}

// NewCSVParser creates a new CSV parser instance.
func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

// Parse reads CSV data and returns a ParsedScorecard.
// Expected format: first row contains column headers including "round_relative_score".
// If round_relative_score column is found, use it directly; otherwise fall back to hole-by-hole parsing.
func (p *CSVParser) Parse(fileData []byte, fileName string) (*roundtypes.ParsedScorecard, error) {
	reader := csv.NewReader(strings.NewReader(string(fileData)))

	var rows [][]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV: %w", err)
		}
		rows = append(rows, record)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV must contain at least header and one data row")
	}

	// Parse header to check for UDisc format (round_relative_score column)
	headerRow := rows[0]
	relativeScoreColIdx := -1
	playerNameColIdx := -1

	// Look for round_relative_score and player name columns
	for i, col := range headerRow {
		colLower := strings.ToLower(strings.TrimSpace(col))
		// Remove common separators to match variations like "round_relative_score", "Round Relative Score", "RoundRelativeScore"
		colNormalized := strings.ReplaceAll(strings.ReplaceAll(colLower, " ", ""), "_", "")

		if colNormalized == "roundrelativescore" || colLower == "round_relative_score" || colLower == "relative score" {
			relativeScoreColIdx = i
		}
		if colLower == "playername" || colLower == "player_name" || colLower == "name" {
			playerNameColIdx = i
		}
	}

	// If we found the relative score column, use UDisc format parsing
	if relativeScoreColIdx >= 0 {
		// Log that we detected UDisc format for debugging
		fmt.Printf("CSV Parser: Detected UDisc format with round_relative_score at column %d\n", relativeScoreColIdx)
		return p.parseUDiscFormat(rows, relativeScoreColIdx, playerNameColIdx, fileName)
	}

	// Fall back to legacy hole-by-hole format
	fmt.Printf("CSV Parser: Using legacy format (no round_relative_score column found). Header: %v\n", headerRow)
	return p.parseLegacyFormat(rows, fileName)
}

// parseUDiscFormat parses CSV with round_relative_score column.
// It extracts hole-by-hole scores for logging but uses round_relative_score as the authoritative score.
func (p *CSVParser) parseUDiscFormat(rows [][]string, relativeScoreColIdx, playerNameColIdx int, fileName string) (*roundtypes.ParsedScorecard, error) {
	var playerScores []PlayerScore
	headerRow := rows[0]

	// Find hole score columns (columns that look like hole numbers: 1, 2, 3, etc.)
	var holeColIndices []int
	for i, col := range headerRow {
		colTrimmed := strings.TrimSpace(col)
		if _, err := strconv.Atoi(colTrimmed); err == nil {
			holeColIndices = append(holeColIndices, i)
		}
	}

	// Start from row 1 (skip header)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= relativeScoreColIdx {
			continue
		}

		// Extract player name
		playerName := ""
		if playerNameColIdx >= 0 && playerNameColIdx < len(row) {
			playerName = strings.TrimSpace(row[playerNameColIdx])
		} else if len(row) > 0 {
			playerName = strings.TrimSpace(row[0]) // Default to first column
		}

		if playerName == "" {
			continue
		}

		// Extract relative score (this is the authoritative score)
		relativeScoreStr := strings.TrimSpace(row[relativeScoreColIdx])
		if relativeScoreStr == "" {
			continue
		}

		relativeScore, err := strconv.Atoi(relativeScoreStr)
		if err != nil {
			continue // Skip invalid scores
		}

		fmt.Printf("CSV Parser: Extracted player '%s' with relative score %d (from column value '%s')\n", playerName, relativeScore, relativeScoreStr)

		// Extract hole scores for informational purposes (if available)
		var holeScores []int
		if len(holeColIndices) > 0 {
			for _, holeIdx := range holeColIndices {
				if holeIdx < len(row) {
					holeScoreStr := strings.TrimSpace(row[holeIdx])
					if holeScore, err := strconv.Atoi(holeScoreStr); err == nil && holeScore > 0 {
						holeScores = append(holeScores, holeScore)
					}
				}
			}
		}

		// Store relative score as single value in HoleScores[0] - this signals to import service to use it directly
		playerScores = append(playerScores, PlayerScore{
			PlayerName: playerName,
			HoleScores: []int{relativeScore}, // Single value at index 0 means "use this as relative score"
			Total:      relativeScore,
		})
	}

	if len(playerScores) == 0 {
		return nil, fmt.Errorf("no valid player scores found in UDisc CSV")
	}

	return &roundtypes.ParsedScorecard{
		PlayerScores: convertPlayerScores(playerScores),
		ParScores:    []int{}, // No par info needed when we have relative scores
	}, nil
}

// parseLegacyFormat parses CSV with hole-by-hole scores and par row.
func (p *CSVParser) parseLegacyFormat(rows [][]string, fileName string) (*roundtypes.ParsedScorecard, error) {
	headerRow := rows[0]
	if len(headerRow) < 2 {
		return nil, fmt.Errorf("header row must have at least player name column and one hole")
	}

	// Find par row (might be second row or within data)
	var parScores []int
	var playerScores []PlayerScore
	parRowIdx := -1

	// Check if second row contains par information
	if len(rows) > 1 {
		secondRow := rows[1]
		if isPARRow(secondRow[0]) {
			parRowIdx = 1
			parScores = extractParScores(secondRow[1:])
		}
	}

	// Parse player scores
	startIdx := 1
	if parRowIdx >= 0 {
		startIdx = parRowIdx + 1
	}

	for i := startIdx; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 {
			continue // Skip empty or malformed rows
		}

		playerName := strings.TrimSpace(row[0])
		if playerName == "" {
			continue // Skip rows without player names
		}

		scores := make([]int, 0, len(row)-1)
		total := 0

		for j := 1; j < len(row); j++ {
			scoreStr := strings.TrimSpace(row[j])
			if scoreStr == "" {
				continue
			}

			score, err := strconv.Atoi(scoreStr)
			if err != nil {
				// Skip non-numeric values
				continue
			}

			scores = append(scores, score)
			total += score
		}

		if len(scores) > 0 {
			playerScores = append(playerScores, PlayerScore{
				PlayerName: playerName,
				HoleScores: scores,
				Total:      total,
			})
		}
	}

	if len(playerScores) == 0 {
		return nil, fmt.Errorf("no valid player scores found in CSV")
	}

	// If no par row was found, infer or set empty
	if len(parScores) == 0 {
		parScores = make([]int, len(playerScores[0].HoleScores))
		// Try to infer from any row, or leave as zeros
		for i := range parScores {
			parScores[i] = 3 // Default assumption for disc golf is par 3
		}
	}

	return &roundtypes.ParsedScorecard{
		ImportID:     "",                    // Set by caller
		RoundID:      sharedtypes.RoundID{}, // Set by caller
		GuildID:      "",                    // Set by caller
		ParScores:    parScores,
		PlayerScores: convertPlayerScores(playerScores),
	}, nil
}

// isPARRow checks if a row header indicates it's a PAR row.
func isPARRow(cellValue string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(cellValue))
	return normalized == "PAR" || normalized == "PARS" || normalized == "P"
}

// extractParScores extracts par values from a row.
func extractParScores(row []string) []int {
	var pars []int
	for _, cell := range row {
		val := strings.TrimSpace(cell)
		if par, err := strconv.Atoi(val); err == nil && par > 0 {
			pars = append(pars, par)
		}
	}
	return pars
}

// convertPlayerScores converts PlayerScore to roundtypes.PlayerScoreRow.
func convertPlayerScores(scores []PlayerScore) []roundtypes.PlayerScoreRow {
	rows := make([]roundtypes.PlayerScoreRow, 0, len(scores))
	for _, ps := range scores {
		rows = append(rows, roundtypes.PlayerScoreRow{
			PlayerName: ps.PlayerName,
			HoleScores: ps.HoleScores,
			Total:      ps.Total,
		})
	}
	return rows
}
