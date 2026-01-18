package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetRoundRequest handles requests to retrieve details for a specific round.
func (h *RoundHandlers) HandleGetRoundRequest(
	ctx context.Context,
	payload *roundevents.GetRoundRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Call the service function to fetch the round
	result, err := h.service.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundRetrievedV1,
		roundevents.RoundRetrievalFailedV1,
	), nil
}
