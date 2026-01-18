package userhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScorecardParsed consumes a parsed scorecard and emits match confirmation or confirmed events.
// This handler uses a type switch because the service may return different success payload types
// depending on whether confirmation is required or all matches were resolved automatically.
func (h *UserHandlers) HandleScorecardParsed(
	ctx context.Context,
	payload *roundevents.ParsedScorecardPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.MatchParsedScorecard(ctx, *payload)
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
