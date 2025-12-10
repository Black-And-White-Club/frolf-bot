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
// Expected format: first row contains hole numbers or "Par", followed by player rows with player name and scores.
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

	// Parse header to extract par scores
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
