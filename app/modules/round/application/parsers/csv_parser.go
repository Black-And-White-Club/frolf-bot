package parsers

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ================ CSV Parser ================

type CSVParser struct{}

func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

func (p *CSVParser) Parse(fileData []byte) (*roundtypes.ParsedScorecard, error) {
	// 1. Preprocess: strip BOM, normalize line endings, detect delimiter
	cleanedData, delimiter, err := preprocessCSVData(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to preprocess CSV: %w", err)
	}

	// 2. Configure reader with detected delimiter
	reader := csv.NewReader(strings.NewReader(cleanedData))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1 // Allow variable-length rows
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV must contain at least header and one data row")
	}

	// 3. Detect header row (may not be first row due to title rows)
	headerRowIdx := detectHeaderRow(rows)
	if headerRowIdx < 0 {
		return nil, fmt.Errorf("no valid header row found in first 5 rows")
	}

	header := rows[headerRowIdx]

	// 4. Find columns with expanded alternatives
	playerNameIdx := findColumn(header, []string{"playername", "player name", "player", "name"})
	relativeScoreIdx := findColumn(header, []string{"+/-", "plusminus", "plus minus", "relative", "relative_score", "net", "to par", "topar"})
	totalScoreIdx := findColumn(header, []string{"total", "score"})

	// Prefer "+/-" if exists, fall back to "total"
	scoreIdx := relativeScoreIdx
	if scoreIdx < 0 {
		scoreIdx = totalScoreIdx
	}

	if scoreIdx < 0 {
		return nil, fmt.Errorf("CSV missing required score column (tried: +/-, total, score, relative)")
	}

	// Default to first column for player name if not found
	if playerNameIdx < 0 {
		playerNameIdx = 0
	}

	// Find hole columns for logging only (supports "1", "hole1", "h1" patterns)
	holeColumns := findHoleColumns(header)

	var playerScores []roundtypes.PlayerScoreRow
	var parScores []int

	// 5. Parse data rows starting after header
	dataStartRow := headerRowIdx + 1

	// Check if first data row after header is Par
	if dataStartRow < len(rows) && len(rows[dataStartRow]) > playerNameIdx &&
		isPARRow(rows[dataStartRow][playerNameIdx]) {
		if len(holeColumns) > 0 {
			parScores = extractScores(rows[dataStartRow], holeColumns)
		}
		dataStartRow++
	}

	// 6. Parse player scores
	for i := dataStartRow; i < len(rows); i++ {
		row := rows[i]

		if playerNameIdx >= len(row) || scoreIdx >= len(row) {
			continue
		}

		playerName := strings.TrimSpace(row[playerNameIdx])
		if playerName == "" {
			continue
		}

		// Extract score from detected column
		scoreStr := strings.TrimSpace(row[scoreIdx])
		if scoreStr == "" {
			continue
		}

		// Handle scores with explicit + sign (e.g., "+5")
		scoreStr = strings.TrimPrefix(scoreStr, "+")

		relativeScore, err := strconv.Atoi(scoreStr)
		if err != nil {
			continue // Skip invalid scores
		}

		// Extract hole scores for logging only (may be empty/incomplete - that's fine)
		var holeScores []int
		if len(holeColumns) > 0 {
			holeScores = extractScores(row, holeColumns)
		}

		// If the player name represents a team (e.g., "Alec + Jess"), split into individuals
		names := SplitPlayerNames(playerName)
		if len(names) <= 1 {
			playerScores = append(playerScores, roundtypes.PlayerScoreRow{
				PlayerName: playerName,
				HoleScores: holeScores,
				Total:      relativeScore,
			})
		} else {
			// This row represents a team/doubles entry
			playerScores = append(playerScores, roundtypes.PlayerScoreRow{
				PlayerName: playerName,
				HoleScores: holeScores,
				Total:      relativeScore,
				IsTeam:     true,
				TeamNames:  names,
			})
		}
	}

	if len(playerScores) == 0 {
		return nil, fmt.Errorf("no valid player scores found in CSV")
	}

	// Detect mode: if any player has TeamNames set, it's doubles
	mode := sharedtypes.RoundModeSingles
	for _, p := range playerScores {
		if len(p.TeamNames) > 1 || p.IsTeam {
			mode = sharedtypes.RoundModeDoubles
			break
		}
	}

	return &roundtypes.ParsedScorecard{
		PlayerScores: playerScores,
		ParScores:    parScores,
		Mode:         mode,
	}, nil
}
