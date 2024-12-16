package scorequeries

import (
	"context"
	"errors"
	"fmt"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
)

// GetScoreQuery represents a query to retrieve a score.
type GetScoreQuery struct {
	DiscordID string `json:"discord_id"`
	RoundID   string `json:"round_id"`
}

// GetScoreHandler defines the method for handling the GetScoreQuery.
type GetScoreHandler interface {
	Handle(ctx context.Context, query GetScoreQuery) (scoredb.Score, error)
}

// getScoreHandler implements GetScoreHandler.
type getScoreHandler struct {
	repo scoredb.ScoreDB
}

// NewGetScoreHandler creates a new getScoreHandler.
func NewGetScoreHandler(repo scoredb.ScoreDB) *getScoreHandler {
	return &getScoreHandler{repo: repo}
}

// Handle retrieves a score based on the given query.
func (h *getScoreHandler) Handle(ctx context.Context, query GetScoreQuery) (scoredb.Score, error) {
	score, err := h.repo.GetScore(ctx, query.DiscordID, query.RoundID)
	if err != nil {
		return scoredb.Score{}, fmt.Errorf("failed to get score: %w", err)
	}
	if score == nil {
		return scoredb.Score{}, errors.New("score not found")
	}

	return *score, nil
}
