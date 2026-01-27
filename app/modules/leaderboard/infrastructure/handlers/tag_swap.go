package leaderboardhandlers

import (
	"context"
	"errors"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	"github.com/google/uuid"
)

// HandleTagSwapRequested handles manual intent to swap tags between two users.
func (h *LeaderboardHandlers) HandleTagSwapRequested(
	ctx context.Context,
	payload *leaderboardevents.TagSwapRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// 1. We still need to know WHAT tag the Target currently holds.
	// The service helper 'GetTagByUserID' is perfect for this.
	targetTag, err := h.service.GetTagByUserID(ctx, payload.GuildID, payload.TargetID)
	if err != nil {
		// If Target has no tag, we can't swap.
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.TagSwapFailedV1,
			Payload: &leaderboardevents.TagSwapFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  "target_user_has_no_tag",
			},
		}}, nil
	}

	// 2. Identify the requestor's current tag for the Saga record.
	// We ignore the error here; if they don't have a tag, CurrentTag is just 0.
	requestorTag, _ := h.service.GetTagByUserID(ctx, payload.GuildID, payload.RequestorID)

	// 3. Attempt the Funnel
	resultData, err := h.service.ExecuteBatchTagAssignment(
		ctx,
		payload.GuildID,
		[]sharedtypes.TagAssignmentRequest{
			{UserID: payload.RequestorID, TagNumber: targetTag},
		},
		sharedtypes.RoundID(uuid.New()),
		sharedtypes.ServiceUpdateSourceTagSwap,
	)

	// 4. Traffic Cop: Handle Saga Redirection
	if err != nil {
		var swapErr *leaderboardservice.TagSwapNeededError
		if errors.As(err, &swapErr) {
			// This is a "Partial Success" - the intent is recorded.
			intentErr := h.sagaCoordinator.ProcessIntent(ctx, saga.SwapIntent{
				UserID:     payload.RequestorID,
				CurrentTag: requestorTag,
				TargetTag:  targetTag,
				GuildID:    payload.GuildID,
			})
			if intentErr != nil {
				return nil, intentErr
			}

			return []handlerwrapper.Result{{
				Topic: leaderboardevents.TagSwapProcessedV1,
				Payload: &leaderboardevents.TagSwapProcessedPayloadV1{
					GuildID:     payload.GuildID,
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
				},
			}}, nil
		}
		return nil, err
	}

	// 5. Success Path (Immediate Swap)
	return h.mapSuccessResults(payload.GuildID, payload.RequestorID, "manual-swap", resultData, "tag_swap"), nil
}
