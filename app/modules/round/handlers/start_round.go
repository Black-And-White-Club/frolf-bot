package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// StartRoundHandler handles start round events.
type StartRoundHandler struct {
	RoundDB  rounddb.RoundDB
	PubSuber watermillutil.PubSuber
}

// NewStartRoundHandler creates a new StartRoundHandler.
func NewStartRoundHandler(roundDB rounddb.RoundDB, pubsuber watermillutil.PubSuber) *StartRoundHandler {
	return &StartRoundHandler{
		RoundDB:  roundDB,
		PubSuber: pubsuber,
	}
}

// Handle processes start round events.
func (h *StartRoundHandler) Handle(ctx context.Context, msg *message.Message) error {
	var event RoundStartEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal RoundStartEvent: %w", err)
	}

	// Fetch the round from the database using event.RoundID
	round, err := h.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if the round still exists and is in the upcoming state
	if round == nil || round.State != rounddb.RoundStateUpcoming {
		return nil // Ignore the start event if the round doesn't exist or is not upcoming
	}

	// 1. Update the round state to "in progress"
	if err := h.RoundDB.UpdateRoundState(ctx, round.ID, rounddb.RoundStateInProgress); err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// 2. Create blank score associations for participants with "Accepted" or "Tentative" status
	if round.Scores == nil {
		round.Scores = make(map[string]int)
	}
	for _, participant := range round.Participants {
		if participant.Response == rounddb.ResponseAccept || participant.Response == rounddb.ResponseTentative {
			// Create a blank score entry in the Scores map
			scoreKey := fmt.Sprintf("%d", *participant.TagNumber) // Assuming tag number is unique
			round.Scores[scoreKey] = 0
		}
	}

	// Update the round with the new Scores map
	if err := h.RoundDB.CreateRoundScores(ctx, round.ID, round.Scores); err != nil {
		return fmt.Errorf("failed to create round scores: %w", err)
	}

	return nil
}
