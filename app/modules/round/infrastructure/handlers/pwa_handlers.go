package roundhandlers

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundListRequest handles PWA requests for round list
func (h *RoundHandlers) HandleRoundListRequest(
	ctx context.Context,
	req *RoundListRequest,
) ([]handlerwrapper.Result, error) {
	// Fetch rounds from service
	rounds, err := h.service.GetRoundsForGuild(ctx, sharedtypes.GuildID(req.GuildID))
	if err != nil {
		return nil, err
	}

	// Return raw round data; clients can render fields they need
	response := struct {
		Rounds []*roundtypes.Round `json:"rounds"`
	}{Rounds: rounds}

	return []handlerwrapper.Result{
		{
			Topic:   "round.list.response.v1",
			Payload: &response,
		},
	}, nil
}
