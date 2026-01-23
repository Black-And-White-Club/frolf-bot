package parsers

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// ================ CSV Parser ================

type CSVParser struct{}

func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

func (p *CSVParser) Parse(fileData []byte) (*roundtypes.ParsedScorecard, error) {
	reader := csv.NewReader(strings.NewReader(string(fileData)))

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV must contain at least header and one data row")
	}

	header := rows[0]

	// Find the columns we care about
	playerNameIdx := findColumn(header, []string{"playername", "player name", "player"})
	relativeScoreIdx := findColumn(header, []string{"+/-"})

	if relativeScoreIdx < 0 {
		return nil, fmt.Errorf("CSV missing required '+/-' column")
	}

	// Default to first column for player name if not found
	if playerNameIdx < 0 {
		playerNameIdx = 0
	}

	// Find hole columns for logging only
	holeColumns := findHoleColumns(header)

	var playerScores []roundtypes.PlayerScoreRow
	var parScores []int

	// Check if first data row is Par
	dataStartRow := 1
	if len(rows) > 1 && isPARRow(rows[1][playerNameIdx]) {
		if len(holeColumns) > 0 {
			parScores = extractScores(rows[1], holeColumns)
		}
		dataStartRow = 2
	}

	// Parse player scores
	for i := dataStartRow; i < len(rows); i++ {
		row := rows[i]

		if playerNameIdx >= len(row) || relativeScoreIdx >= len(row) {
			continue
		}

		playerName := strings.TrimSpace(row[playerNameIdx])
		if playerName == "" {
			continue
		}

		// CSV files ONLY extract from the +/- column
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

		// Total is the relative score from the +/- column
		// If the player name represents a team (e.g., "Alec + Jess"), split into individuals
		names := SplitPlayerNames(playerName)
		if len(names) <= 1 {
			playerScores = append(playerScores, roundtypes.PlayerScoreRow{
				PlayerName: playerName,
				HoleScores: holeScores,
				Total:      relativeScore,
			})
		} else {
			// This row represents a team/doubles entry. Create one PlayerScoreRow per teammate
			// and set TeamMembers so downstream logic can detect a doubles-derived row.
			for _, n := range names {
				playerScores = append(playerScores, roundtypes.PlayerScoreRow{
					PlayerName: n,
					HoleScores: holeScores,
					Total:      relativeScore,
				})
			}
		}
	}

	if len(playerScores) == 0 {
		return nil, fmt.Errorf("no valid player scores found in CSV")
	}

	return &roundtypes.ParsedScorecard{
		PlayerScores: playerScores,
		ParScores:    parScores, // May be empty - just for logging
	}, nil
}
