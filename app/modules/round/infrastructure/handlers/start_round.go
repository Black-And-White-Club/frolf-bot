package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundStarted processes the transition of a round to the started state.
func (h *RoundHandlers) HandleRoundStarted(
	ctx context.Context,
	payload *roundevents.RoundStartedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.roundService.ProcessRoundStart(ctx, *payload)
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
