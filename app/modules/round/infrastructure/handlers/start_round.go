package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundStartRequested handles the minimal backend request to start a round.
// The handler uses the DB as the source of truth (service will fetch the round).
func (h *RoundHandlers) HandleRoundStartRequested(
	ctx context.Context,
	payload *roundevents.RoundStartRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.roundService.ProcessRoundStart(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "round start processing failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundStartFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		discordStartPayload, ok := result.Success.(*roundevents.DiscordRoundStartPayloadV1)
		if !ok {
			return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from ProcessRoundStart"}
		}

		return []handlerwrapper.Result{
			{Topic: roundevents.RoundStartedDiscordV1, Payload: discordStartPayload},
		}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected result from service during round start"}
}
