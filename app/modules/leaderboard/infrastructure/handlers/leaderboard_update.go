package leaderboardhandlers

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
)

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
// This is for score processing after round completion - updates leaderboard with new participant tags.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateRequested(
	ctx context.Context,
	payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Processing leaderboard update from scores",
		attr.ExtractCorrelationID(ctx),
		attr.String("round_id", payload.RoundID.String()))

	requests := make([]sharedtypes.TagAssignmentRequest, 0, len(payload.SortedParticipantTags))
	for _, tagUserPair := range payload.SortedParticipantTags {
		parts := strings.Split(tagUserPair, ":")
		if len(parts) != 2 {
			continue
		}
		tagNum, _ := strconv.Atoi(parts[0])
		requests = append(requests, sharedtypes.TagAssignmentRequest{
			UserID:    sharedtypes.DiscordID(parts[1]),
			TagNumber: sharedtypes.TagNumber(tagNum),
		})
	}

	result, err := h.leaderboardService.ExecuteBatchTagAssignment(
		ctx,
		payload.GuildID,
		requests,
		payload.RoundID,
		sharedtypes.ServiceUpdateSourceProcessScores,
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

	// Build success events
	leaderboardData := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID, len(result.Leaderboard))
	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(result.Leaderboard))
	for _, entry := range result.Leaderboard {
		leaderboardData[entry.TagNumber] = entry.UserID
		changedTags[entry.UserID] = entry.TagNumber
	}

	return []handlerwrapper.Result{
		{
			Topic: leaderboardevents.LeaderboardUpdatedV1,
			Payload: &leaderboardevents.LeaderboardUpdatedPayloadV1{
				GuildID:         payload.GuildID,
				RoundID:         payload.RoundID,
				LeaderboardData: leaderboardData,
			},
		},
		{
			Topic: sharedevents.TagUpdateForScheduledRoundsV1,
			Payload: &leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
				GuildID:     payload.GuildID,
				RoundID:     payload.RoundID,
				Source:      "leaderboard_update",
				UpdatedAt:   time.Now().UTC(),
				ChangedTags: changedTags,
			},
		},
	}, nil
}
