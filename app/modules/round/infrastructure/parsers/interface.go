package parsers

import (
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// Parser defines the interface for scorecard parsers.
type Parser interface {
	// Parse reads scorecard data and returns a ParsedScorecard.
	// fileData should contain the raw file bytes.
	// fileName is optional and used for validation.
	Parse(fileData []byte, fileName string) (*roundtypes.ParsedScorecard, error)
}

// ScoreFormat indicates whether we have relative scores or hole-by-hole
type ScoreFormat int

const (
	FormatUnknown    ScoreFormat = iota
	FormatRelative               // UDisc format with relative scores
	FormatHoleByHole             // Legacy format with individual hole scores
)

// ParsedScorecard represents the result with explicit format indication
type ParsedScorecardInternal struct {
	Format       ScoreFormat
	PlayerScores []PlayerScore
	ParScores    []int
}

type PlayerScore struct {
	PlayerName    string
	RelativeScore *int  // Pointer so we can tell if it's set
	HoleScores    []int // Empty if format is relative-only
	Total         int
}
