package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetRoundRequest handles requests to retrieve details for a specific round.
func (h *RoundHandlers) HandleGetRoundRequest(
	ctx context.Context,
	payload *roundevents.GetRoundRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Call the service function to fetch the round
	result, err := h.roundService.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		h.logger.WarnContext(ctx, "get round request failed",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundRetrievalFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		round, ok := result.Success.(*roundtypes.Round)
		if !ok {
			return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from GetRound service"}
		}

		return []handlerwrapper.Result{
			{Topic: roundevents.RoundRetrievedV1, Payload: round},
		}, nil
	}

	h.logger.ErrorContext(ctx, "unexpected empty result from GetRound service")
	return nil, sharedtypes.ValidationError{Message: "unexpected result from service"}
}
