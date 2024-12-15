// In app/modules/round/queries/get_score_for_participant.go

package roundqueries

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

// GetScoreForParticipantQuery represents the query to get the score for a participant in a round.
type GetScoreForParticipantQuery struct {
	RoundID       int64  `json:"round_id"`
	ParticipantID string `json:"participant_id"`
}

// GetScoreForParticipantHandler handles the GetScoreForParticipantQuery.
type GetScoreForParticipantHandler interface {
	Handle(ctx context.Context, query GetScoreForParticipantQuery) (*rounddb.Score, error)
}

type getScoreForParticipantHandler struct {
	roundDB rounddb.RoundDB
}

// NewGetScoreForParticipantHandler creates a new getScoreForParticipantHandler.
func NewGetScoreForParticipantHandler(roundDB rounddb.RoundDB) *getScoreForParticipantHandler {
	return &getScoreForParticipantHandler{roundDB: roundDB}
}

// Handle processes the GetScoreForParticipantQuery.
func (h *getScoreForParticipantHandler) Handle(ctx context.Context, query GetScoreForParticipantQuery) (*rounddb.Score, error) {
	score, err := h.roundDB.GetScoreForParticipant(ctx, query.RoundID, query.ParticipantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get score for participant: %w", err)
	}
	return score, nil
}
