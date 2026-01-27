package userhandlers

import (
	"context"
	"errors"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScorecardParsed consumes a parsed scorecard and emits match confirmation events.
func (h *UserHandlers) HandleScorecardParsed(
	ctx context.Context,
	payload *roundevents.ParsedScorecardPayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil || payload.ParsedData == nil {
		return nil, errors.New("payload or parsed data cannot be nil")
	}

	// Build player names slice
	playerNames := make([]string, 0, len(payload.ParsedData.PlayerScores))
	for _, row := range payload.ParsedData.PlayerScores {
		playerNames = append(playerNames, row.PlayerName)
	}

	// Call the service
	result, err := h.service.MatchParsedScorecard(ctx, payload.GuildID, payload.UserID, playerNames)
	if err != nil {
		return nil, err // infrastructure error
	}

	if result.IsFailure() {
		// Domain failure — could log or emit if desired, returning nil for now
		return nil, nil
	}

	if result.Success == nil {
		return nil, errors.New("MatchParsedScorecard returned nil success result")
	}

	// Dereference pointer to access MatchResult fields
	matchResult := *result.Success

	var res []handlerwrapper.Result

	// Emit confirmed mappings
	for _, mapping := range matchResult.Mappings {
		res = append(res, handlerwrapper.Result{
			Topic:   userevents.UDiscMatchConfirmedV1,
			Payload: mapping,
		})
	}

	// Emit confirmation-required event for unmatched players (all in one payload)
	if len(matchResult.Unmatched) > 0 {
		res = append(res, handlerwrapper.Result{
			Topic: userevents.UDiscMatchConfirmationRequiredV1,
			Payload: userevents.UDiscMatchConfirmationRequiredPayloadV1{
				GuildID:          payload.GuildID,
				RoundID:          payload.RoundID,
				ImportID:         payload.ImportID,
				UserID:           payload.UserID,
				ChannelID:        payload.ChannelID,
				UnmatchedPlayers: matchResult.Unmatched,
				Timestamp:        time.Now(),
			},
		})
	}

	return res, nil
}
