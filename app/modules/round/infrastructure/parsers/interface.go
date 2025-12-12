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

// PlayerScore represents a player's scores from a scorecard.
type PlayerScore struct {
	PlayerName string
	HoleScores []int
	Total      int
}
