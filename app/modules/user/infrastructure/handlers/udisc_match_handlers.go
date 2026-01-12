package userhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScorecardParsed consumes a parsed scorecard and emits match confirmation or confirmed events.
func (h *UserHandlers) HandleScorecardParsed(
	ctx context.Context,
	payload *roundevents.ParsedScorecardPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Convert V1 payload to service payload format
	servicePayload := roundevents.ParsedScorecardPayload{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		ImportID:       payload.ImportID,
		EventMessageID: payload.EventMessageID,
		UserID:         payload.UserID,
		ChannelID:      payload.ChannelID,
		ParsedData:     payload.ParsedData,
		Timestamp:      payload.Timestamp,
	}
	result, err := h.userService.MatchParsedScorecard(ctx, servicePayload)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return nil, err
	}

	switch v := result.Success.(type) {
	case *userevents.UDiscMatchConfirmationRequiredPayloadV1:
		return []handlerwrapper.Result{
			{Topic: userevents.UDiscMatchConfirmationRequiredV1, Payload: v},
		}, nil
	case *userevents.UDiscMatchConfirmedPayloadV1:
		return []handlerwrapper.Result{
			{Topic: userevents.UDiscMatchConfirmedV1, Payload: v},
		}, nil
	}

	return nil, nil
}
