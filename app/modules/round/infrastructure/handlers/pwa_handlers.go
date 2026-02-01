package roundhandlers

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
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

	// Collect all user IDs from participants
	userIDs := h.collectUserIDsFromRounds(rounds)

	// Lookup profiles (best-effort, don't fail if this errors)
	profiles := make(map[sharedtypes.DiscordID]*usertypes.UserProfile)
	if len(userIDs) > 0 {
		result, _ := h.userService.LookupProfiles(ctx, userIDs)
		if result.IsSuccess() {
			profiles = *result.Success
		}
	}

	// Return raw round data + profiles
	response := struct {
		Rounds   []*roundtypes.Round                              `json:"rounds"`
		Profiles map[sharedtypes.DiscordID]*usertypes.UserProfile `json:"profiles"`
	}{
		Rounds:   rounds,
		Profiles: profiles,
	}

	// Check for reply_to subject for Request-Reply pattern
	topic := "round.list.response.v1"
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{
		{
			Topic:   topic,
			Payload: &response,
		},
	}, nil
}

func (h *RoundHandlers) collectUserIDsFromRounds(rounds []*roundtypes.Round) []sharedtypes.DiscordID {
	seen := make(map[sharedtypes.DiscordID]bool)
	var userIDs []sharedtypes.DiscordID

	for _, r := range rounds {
		// Add creator
		if !seen[r.CreatedBy] {
			seen[r.CreatedBy] = true
			userIDs = append(userIDs, r.CreatedBy)
		}
		// Add participants
		for _, p := range r.Participants {
			if !seen[p.UserID] {
				seen[p.UserID] = true
				userIDs = append(userIDs, p.UserID)
			}
		}
	}

	return userIDs
}
