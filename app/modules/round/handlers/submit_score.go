package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubmitScoreHandler handles the SubmitScore command.
type SubmitScoreHandler struct {
	RoundDB  rounddb.RoundDB
	PubSuber watermillutil.PubSuber
}

// NewSubmitScoreHandler creates a new SubmitScoreHandler.
func NewSubmitScoreHandler(roundDB rounddb.RoundDB, pubsuber watermillutil.PubSuber) *SubmitScoreHandler {
	return &SubmitScoreHandler{
		RoundDB:  roundDB,
		PubSuber: pubsuber,
	}
}

// Handle processes the SubmitScore command.
func (h *SubmitScoreHandler) Handle(ctx context.Context, msg *message.Message) error {
	var dto rounddto.SubmitScoreInput // Use the DTO
	if err := json.Unmarshal(msg.Payload, &dto); err != nil {
		return fmt.Errorf("failed to unmarshal SubmitScoreCommand: %w", err)
	}

	// Fetch the round from the database
	round, err := h.RoundDB.GetRound(ctx, dto.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if the round is finalized
	if round.State == rounddb.RoundStateFinalized {
		return fmt.Errorf("cannot submit score for a finalized round")
	}

	// Update the score in the PendingScores slice (using SubmitScore)
	if err := h.RoundDB.SubmitScore(ctx, dto.RoundID, dto.DiscordID, dto.Score); err != nil {
		return fmt.Errorf("failed to submit score: %w", err)
	}

	// Re-fetch the round to get updated PendingScores
	round, err = h.RoundDB.GetRound(ctx, dto.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if all participants have submitted scores
	allScoresSubmitted := true
	for _, participant := range round.Participants {
		scoreFound := false
		for _, score := range round.PendingScores {
			if score.ParticipantID == participant.DiscordID {
				scoreFound = true
				break
			}
		}
		if !scoreFound {
			allScoresSubmitted = false
			break
		}
	}

	if allScoresSubmitted {
		// Publish an event to trigger the FinalizeAndProcessScoresHandler
		finalizeEvent := &FinalizeRoundEvent{
			RoundID: round.ID,
		}
		finalizePayload, err := json.Marshal(finalizeEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal FinalizeRoundEvent: %w", err)
		}
		if err := h.PubSuber.Publish(TopicRoundFinalized, message.NewMessage(watermill.NewUUID(), finalizePayload)); err != nil {
			return fmt.Errorf("failed to publish FinalizeRoundEvent: %w", err)
		}
	}

	return nil
}
