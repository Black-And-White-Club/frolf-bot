package parsers

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// XLSXParser implements the Parser interface for XLSX scorecard files.
type XLSXParser struct{}

// NewXLSXParser creates a new XLSX parser instance.
func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

// Parse reads XLSX data and returns a ParsedScorecard.
// It extracts scores from the first sheet. Checks for UDisc format with round_relative_score column first.
func (p *XLSXParser) Parse(fileData []byte, fileName string) (*roundtypes.ParsedScorecard, error) {
	f, err := excelize.OpenReader(bytes.NewReader(fileData))
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

	// Check first row for UDisc format (round_relative_score column)
	headerRow := rows[0]
	relativeScoreColIdx := -1
	playerNameColIdx := -1

	for i, col := range headerRow {
		colLower := strings.ToLower(strings.TrimSpace(col))
		colNormalized := strings.ReplaceAll(strings.ReplaceAll(colLower, " ", ""), "_", "")

		if colNormalized == "roundrelativescore" || colLower == "round_relative_score" || colLower == "relative score" {
			relativeScoreColIdx = i
		}
		// XLSX uses "username" column
		if colLower == "username" || colLower == "user_name" || colLower == "name" {
			playerNameColIdx = i
		}
	}

	// If we found the relative score column, use UDisc format parsing
	if relativeScoreColIdx >= 0 {
		fmt.Printf("XLSX Parser: Detected UDisc format with round_relative_score at column %d\n", relativeScoreColIdx)
		return p.parseUDiscFormat(rows, relativeScoreColIdx, playerNameColIdx)
	}

	// Fall back to legacy hole-by-hole format
	fmt.Printf("XLSX Parser: Using legacy format (no round_relative_score column found). Header: %v\n", headerRow)
	return p.parseLegacyFormat(rows)
}

// parseUDiscFormat parses XLSX with round_relative_score column.
func (p *XLSXParser) parseUDiscFormat(rows [][]string, relativeScoreColIdx, playerNameColIdx int) (*roundtypes.ParsedScorecard, error) {
	var playerScores []PlayerScore

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
			playerName = strings.TrimSpace(row[0])
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

		fmt.Printf("XLSX Parser: Extracted player '%s' with relative score %d (from column value '%s')\n", playerName, relativeScore, relativeScoreStr)

		// Store relative score as single value - signals to import service to use directly
		playerScores = append(playerScores, PlayerScore{
			PlayerName: playerName,
			HoleScores: []int{relativeScore}, // Single value: use as relative score
			Total:      relativeScore,
		})
	}

	if len(playerScores) == 0 {
		return nil, fmt.Errorf("no valid player scores found in UDisc XLSX")
	}

	return &roundtypes.ParsedScorecard{
		PlayerScores: convertPlayerScores(playerScores),
		ParScores:    []int{}, // No par info needed when we have relative scores
	}, nil
}

// parseLegacyFormat parses XLSX with hole-by-hole scores and par row.
func (p *XLSXParser) parseLegacyFormat(rows [][]string) (*roundtypes.ParsedScorecard, error) {
	var parScores []int
	var playerScores []PlayerScore
	parRowIdx := -1

	// Look for PAR row (usually near the top)
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
