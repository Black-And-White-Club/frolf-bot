package leaderboardhandlers

import (
	"context"
	"errors"
	"fmt"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	"github.com/google/uuid"
)

func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(
	ctx context.Context,
	payload *sharedevents.BatchTagAssignmentRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	requests := make([]sharedtypes.TagAssignmentRequest, len(payload.Assignments))
	for i, a := range payload.Assignments {
		requests[i] = sharedtypes.TagAssignmentRequest{
			UserID:    a.UserID,
			TagNumber: a.TagNumber,
		}
	}

	batchUUID, err := uuid.Parse(payload.BatchID)
	if err != nil {
		return nil, fmt.Errorf("invalid batch_id format: %w", err)
	}

	result, err := h.service.ExecuteBatchTagAssignment(
		ctx,
		payload.GuildID,
		requests,
		sharedtypes.RoundID(batchUUID),
		sharedtypes.ServiceUpdateSourceAdminBatch,
	)

	if err != nil {
		var swapErr *leaderboardservice.TagSwapNeededError
		if errors.As(err, &swapErr) {
			intentErr := h.sagaCoordinator.ProcessIntent(ctx, saga.SwapIntent{
				UserID:     swapErr.RequestorID,
				CurrentTag: swapErr.CurrentTag,
				TargetTag:  swapErr.TargetTag,
				GuildID:    payload.GuildID,
			})
			return []handlerwrapper.Result{}, intentErr
		}
		return nil, err
	}

	return h.mapSuccessResults(payload.GuildID, payload.RequestingUserID, payload.BatchID, result, "batch_assignment"), nil
}
