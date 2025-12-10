package parsers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// XLSXParser implements the Parser interface for XLSX scorecard files.
type XLSXParser struct{}

// NewXLSXParser creates a new XLSX parser instance.
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

// Parse reads XLSX data and returns a ParsedScorecard.
// It extracts scores from the first sheet, assuming standard UDisc XLSX format.
func (p *XLSXParser) Parse(fileData []byte, fileName string) (*roundtypes.ParsedScorecard, error) {
	f, err := excelize.OpenReader(strings.NewReader(string(fileData)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XLSX: %w", err)
	}
	defer f.Close()

	// Get sheet names
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("XLSX file contains no sheets")
	}

	// Use first sheet (usually contains scorecard data)
	sheetName := sheetList[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %s: %w", sheetName, err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("XLSX must contain at least header and one data row")
	}

	var parScores []int
	var playerScores []PlayerScore
	parRowIdx := -1

	// Look for PAR row (usually near the top)
	for i := 0; i < len(rows) && i < 5; i++ { // Check first 5 rows for PAR
		if len(rows[i]) > 0 && isPARRow(rows[i][0]) {
			parRowIdx = i
			parScores = extractParScoresFromRow(rows[i][1:])
			break
		}
	}

	// If no PAR row found, assume it's the first row
	if parRowIdx == -1 {
		// First row might be headers with hole numbers
		headerRow := rows[0]
		if len(headerRow) > 1 {
			parScores = extractParScoresFromRow(headerRow[1:])
		}
		parRowIdx = 0
	}

	// Parse player scores starting after PAR row
	startIdx := parRowIdx + 1
	for i := startIdx; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 {
			continue
		}

		playerName := strings.TrimSpace(row[0])
		if playerName == "" || playerName == "Player" || playerName == "Name" {
			continue // Skip header rows or empty rows
		}

		scores := make([]int, 0, len(row)-1)
		total := 0

		for j := 1; j < len(row); j++ {
			cellVal := strings.TrimSpace(row[j])
			if cellVal == "" {
				continue
			}

			// Try to parse as integer
			score, err := strconv.Atoi(cellVal)
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
		return nil, fmt.Errorf("no valid player scores found in XLSX")
	}

	// If no par row was found or extracted, set defaults
	if len(parScores) == 0 {
		parScores = make([]int, len(playerScores[0].HoleScores))
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

// extractParScoresFromRow extracts par values from a row.
func extractParScoresFromRow(row []string) []int {
	var pars []int
	for _, cell := range row {
		val := strings.TrimSpace(cell)
		// Try to parse as integer
		if par, err := strconv.Atoi(val); err == nil && par > 0 && par < 20 {
			pars = append(pars, par)
		}
	}
	return pars
}
