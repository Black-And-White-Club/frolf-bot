package scorequeries

import (
	"context"
	"fmt"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
)

// ScoreQueryService defines the methods for querying score data.
type ScoreQueryService interface {
	GetScore(ctx context.Context, query *GetScoreQuery) (*scoredb.Score, error)
}

// scoreQueryService implements ScoreQueryService.
type scoreQueryService struct {
	repo scoredb.ScoreDB
}

// NewScoreQueryService creates a new ScoreQueryService.
func NewScoreQueryService(repo scoredb.ScoreDB) ScoreQueryService {
	return &scoreQueryService{repo: repo}
}

// GetScore retrieves a score based on the given query.
func (s *scoreQueryService) GetScore(ctx context.Context, query *GetScoreQuery) (*scoredb.Score, error) {
	// Assuming GetScoreQuery has DiscordID and RoundID
	score, err := s.repo.GetScore(ctx, query.DiscordID, query.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get score: %w", err)
	}
	return score, nil
}
