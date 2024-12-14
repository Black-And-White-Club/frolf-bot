package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetTagNumberRequest represents the request to get a user's tag number.
type GetTagNumberRequest struct {
	DiscordID string `json:"user_id"`
}

// GetTagNumberResponse represents the response to a GetTagNumberRequest.
type GetTagNumberResponse struct {
	TagNumber *int `json:"tag_number"`
}

// JoinRoundHandler handles the JoinRoundRequest command.
type JoinRoundHandler struct {
	roundDB  rounddb.RoundDB
	eventBus watermillutil.PubSuber
}

// NewJoinRoundHandler creates a new JoinRoundHandler.
func NewJoinRoundHandler(roundDB rounddb.RoundDB, eventBus watermillutil.PubSuber) *JoinRoundHandler {
	return &JoinRoundHandler{
		roundDB:  roundDB,
		eventBus: eventBus,
	}
}

// Handle processes the JoinRoundRequest command.
func (h *JoinRoundHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd roundcommands.JoinRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal JoinRoundRequest: %w", err)
	}

	// 1. Validate the command (e.g., ensure the round exists)
	// ... (Your validation logic here) ...

	// 2. Publish a GetTagNumberRequest event to the leaderboard module
	getTagNumberRequest := GetTagNumberRequest{
		RoundID:   cmd.RoundID, // Include the RoundID
		DiscordID: cmd.DiscordID,
	}

	// 3. Subscribe to the response topic from the leaderboard module
	responseChan, err := h.eventBus.Subscribe(ctx, "tag.number.response")
	if err != nil {
		return fmt.Errorf("failed to subscribe to tag number response: %w", err)
	}

	// 4. Wait for the response (with a timeout)
	var tagNumber *int
	select {
	case responseMsg := <-responseChan:
		var response GetTagNumberResponse
		if err := json.Unmarshal(responseMsg.Payload, &response); err != nil {
			return fmt.Errorf("failed to unmarshal GetTagNumberResponse: %w", err)
		}
		tagNumber = response.TagNumber
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for tag number response")
	}

	// 5. Add the user as a participant to the round in the database
	participant := rounddb.Participant{
		DiscordID: cmd.DiscordID,
		TagNumber: tagNumber,
		Response:  cmd.Response,
	}
	err = h.roundDB.AddParticipantToRound(ctx, cmd.RoundID, participant)
	if err != nil {
		return fmt.Errorf("failed to add participant to round: %w", err)
	}

	// 6. Publish a ParticipantJoinedRound event
	event := ParticipantJoinedRoundEvent{
		RoundID:     cmd.RoundID,
		Participant: participant,
	}
	payload, err = json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ParticipantJoinedRoundEvent: %w", err)
	}
	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish ParticipantJoinedRoundEvent: %w", err)
	}

	return nil
}
