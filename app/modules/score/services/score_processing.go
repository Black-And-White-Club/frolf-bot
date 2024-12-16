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
	// Sort scores by Score (ascending for golf scores) and then by TagNumber
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].TagNumber < scores[j].TagNumber
		}
		return scores[i].Score < scores[j].Score
	})

	return scores, nil
}
