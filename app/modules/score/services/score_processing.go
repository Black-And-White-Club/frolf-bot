package scoreservice

import (
	"sort"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
)

// ScoresProcessingService handles business logic for processing scores.
type ScoresProcessingService struct{}

// NewScoresProcessingService creates a new ScoresProcessingService instance.
func NewScoresProcessingService() *ScoresProcessingService {
	return &ScoresProcessingService{}
}

// SortScores sorts the scores in ascending order, keeping tags associated with participants.
func (s *ScoresProcessingService) SortScores(scores []scoredb.Score) ([]scoredb.Score, error) {
	// Sort scores by the Score field (ascending order for disc golf)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	return scores, nil
}
