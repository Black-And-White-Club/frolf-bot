package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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

	// Handle business failure
	if result.Failure != nil {
		err := *result.Failure
		return []handlerwrapper.Result{{
			Topic: roundevents.RoundRetrievalFailedV1,
			Payload: &roundevents.RoundRetrievalFailedPayloadV1{
				GuildID: payload.GuildID,
				RoundID: payload.RoundID,
				Error:   err.Error(),
			},
		}}, nil
	}

	// Handle success
	if result.Success != nil {
		roundPtr := *result.Success
		roundVal := *roundPtr // Create a value copy to safely modify and embed

		// Ensure GuildID is populated from the request context
		roundVal.GuildID = payload.GuildID

		return []handlerwrapper.Result{{
			Topic: roundevents.RoundRetrievedV1,
			Payload: &roundevents.RoundRetrievedPayloadV1{
				Round: roundVal,
			},
		}}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from GetRound service"}
}
