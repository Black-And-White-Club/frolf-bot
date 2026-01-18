package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundStartRequested handles the minimal backend request to start a round.
// The handler uses the DB as the source of truth (service will fetch the round).
func (h *RoundHandlers) HandleRoundStartRequested(
	ctx context.Context,
	payload *roundevents.RoundStartRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.ProcessRoundStart(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		roundevents.RoundStartedDiscordV1,
		roundevents.RoundStartFailedV1,
	), nil
}
