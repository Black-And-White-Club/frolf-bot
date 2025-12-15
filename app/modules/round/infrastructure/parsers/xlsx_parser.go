package parsers

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// ================ XLSX Parser ================

type XLSXParser struct{}

func NewXLSXParser() *XLSXParser {
	return &XLSXParser{}
}

func (p *XLSXParser) Parse(fileData []byte, fileName string) (*roundtypes.ParsedScorecard, error) {
	f, err := excelize.OpenReader(bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XLSX: %w", err)
	}
	defer f.Close()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("XLSX file contains no sheets")
	}

	rows, err := f.GetRows(sheetList[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("XLSX must contain at least header and one data row")
	}

	header := rows[0]

	// Find the columns we care about
	playerNameIdx := findColumn(header, []string{"username", "name", "player"})
	relativeScoreIdx := findColumn(header, []string{"round_relative_score", "roundrelativescore"})

	if relativeScoreIdx < 0 {
		return nil, fmt.Errorf("XLSX missing required 'round_relative_score' column")
	}

	// Default to first column for player name if not found
	if playerNameIdx < 0 {
		playerNameIdx = 0
	}

	// Find hole columns for logging (optional)
	holeColumns := findHoleColumns(header)

	var playerScores []roundtypes.PlayerScoreRow

	// Parse player scores (skip header)
	for i := 1; i < len(rows); i++ {
		row := rows[i]

		if playerNameIdx >= len(row) || relativeScoreIdx >= len(row) {
			continue
		}

		playerName := strings.TrimSpace(row[playerNameIdx])
		if playerName == "" {
			continue
		}

		// This is the ONLY value we actually care about for scoring
		relativeScoreStr := strings.TrimSpace(row[relativeScoreIdx])
		if relativeScoreStr == "" {
			continue
		}

		relativeScore, err := strconv.Atoi(relativeScoreStr)
		if err != nil {
			continue // Skip invalid relative scores
		}

		// Extract hole scores for logging only (may be empty/incomplete - that's fine)
		var holeScores []int
		if len(holeColumns) > 0 {
			holeScores = extractScores(row, holeColumns)
		}

		// Total is just the relative score
		playerScores = append(playerScores, roundtypes.PlayerScoreRow{
			PlayerName: playerName,
			HoleScores: holeScores, // May be empty or incomplete - just for logging
			Total:      relativeScore,
		})
	}

	if len(playerScores) == 0 {
		return nil, fmt.Errorf("no valid player scores found in XLSX")
	}

	return &roundtypes.ParsedScorecard{
		PlayerScores: playerScores,
		ParScores:    []int{}, // XLSX events don't include par info
	}, nil
}
