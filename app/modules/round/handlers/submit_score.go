package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
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
	var cmd roundcommands.SubmitScoreRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal SubmitScoreCommand: %w", err)
	}

	// Fetch the round from the database
	round, err := h.RoundDB.GetRound(ctx, cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if the round is finalized
	if round.State == rounddb.RoundStateFinalized {
		return fmt.Errorf("cannot submit score for a finalized round")
	}

	// Update the score in the PendingScores slice
	found := false
	for i, score := range round.PendingScores {
		if score.ParticipantID == cmd.ParticipantID {
			round.PendingScores[i].Score = cmd.Score
			found = true
			break
		}
	}
	if !found {
		round.PendingScores = append(round.PendingScores, rounddb.Score{
			ParticipantID: cmd.ParticipantID,
			Score:         cmd.Score,
		})
	}

	// Update the score in the PendingScores slice (using SubmitScore)
	if err := h.RoundDB.SubmitScore(ctx, cmd.RoundID, cmd.ParticipantID, cmd.Score); err != nil {
		return fmt.Errorf("failed to submit score: %w", err)
	}

	// Check if all participants have submitted scores
	allScoresSubmitted := true
	for _, participant := range round.Participants {
		scoreFound := false
		for _, score := range round.PendingScores { // Access PendingScores from the updated round
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
		if err := h.PubSuber.Publish(finalizeEvent.Topic(), message.NewMessage(watermill.NewUUID(), finalizePayload)); err != nil {
			return fmt.Errorf("failed to publish FinalizeRoundEvent: %w", err)
		}
	}

	return nil
}
