package leaderboardhandlers

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

	result, err := h.service.ExecuteBatchTagAssignment(
		ctx,
		payload.GuildID,
		requests,
		payload.RoundID,
		sharedtypes.ServiceUpdateSourceProcessScores,
	)
	if err != nil {
		return nil, err
	}

	if result.IsFailure() {
		var swapErr *leaderboardservice.TagSwapNeededError
		if errors.As(*result.Failure, &swapErr) {
			intentErr := h.sagaCoordinator.ProcessIntent(ctx, saga.SwapIntent{
				UserID:     swapErr.RequestorID,
				CurrentTag: swapErr.CurrentTag,
				TargetTag:  swapErr.TargetTag,
				GuildID:    payload.GuildID,
			})
			return []handlerwrapper.Result{}, intentErr
		}
		return nil, *result.Failure
	}

	updatedData := *result.Success
	// Build success events from returned leaderboard data
	assignments := make([]leaderboardevents.TagAssignmentInfoV1, 0, len(updatedData))
	for _, entry := range updatedData {
		assignments = append(assignments, leaderboardevents.TagAssignmentInfoV1{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		})
	}

	leaderboardData := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID, len(assignments))
	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(assignments))
	for _, entry := range assignments {
		leaderboardData[entry.TagNumber] = entry.UserID
		changedTags[entry.UserID] = entry.TagNumber
	}

	results := []handlerwrapper.Result{
		{
			Topic: leaderboardevents.LeaderboardUpdatedV1,
			Payload: &leaderboardevents.LeaderboardUpdatedPayloadV1{
				GuildID:         payload.GuildID,
				RoundID:         payload.RoundID,
				LeaderboardData: leaderboardData,
			},
		},
		{
			Topic: sharedevents.SyncRoundsTagRequestV1,
			Payload: &sharedevents.SyncRoundsTagRequestPayloadV1{
				GuildID:     payload.GuildID,
				Source:      "leaderboard_update",
				UpdatedAt:   time.Now().UTC(),
				ChangedTags: changedTags,
			},
		},
	}

	// Add both legacy GuildID and internal ClubUUID scoped versions for PWA/NATS transition
	results = h.addParallelIdentityResults(ctx, results, leaderboardevents.LeaderboardUpdatedV1, payload.GuildID)

	return results, nil
}
