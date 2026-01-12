package leaderboardhandlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
// This is for score processing after round completion - updates leaderboard with new participant tags.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateRequested(
	ctx context.Context,
	payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received LeaderboardUpdateRequested event",
		attr.ExtractCorrelationID(ctx),
		attr.RoundID("round_id", payload.RoundID),
		attr.Int("participant_count", len(payload.SortedParticipantTags)),
	)

	// Parse tag:user pairs into assignments
	assignments := make([]sharedtypes.TagAssignmentRequest, len(payload.SortedParticipantTags))
	for i, tagUserPair := range payload.SortedParticipantTags {
		parts := strings.Split(tagUserPair, ":")
		if len(parts) != 2 {
			h.logger.ErrorContext(ctx, "Invalid tag format",
				attr.ExtractCorrelationID(ctx),
				attr.String("tag_pair", tagUserPair),
			)
			return nil, fmt.Errorf("invalid tag format: %s", tagUserPair)
		}

		tagNumber, err := strconv.Atoi(parts[0])
		if err != nil {
			h.logger.ErrorContext(ctx, "Invalid tag number",
				attr.ExtractCorrelationID(ctx),
				attr.String("tag_number_str", parts[0]),
				attr.Error(err),
			)
			return nil, fmt.Errorf("invalid tag number: %w", err)
		}

		assignments[i] = sharedtypes.TagAssignmentRequest{
			UserID:    sharedtypes.DiscordID(parts[1]),
			TagNumber: sharedtypes.TagNumber(tagNumber),
		}
	}

	// Call service to process tag assignments
	result, err := h.leaderboardService.ProcessTagAssignments(
		ctx,
		payload.GuildID,
		sharedtypes.ServiceUpdateSourceProcessScores,
		assignments,
		nil,
		uuid.UUID(payload.RoundID),
		uuid.New(),
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to process tag assignments",
			attr.ExtractCorrelationID(ctx),
			attr.Error(err),
		)
		return nil, err
	}

	// Handle failure outcome
	if result.Failure != nil {
		failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}
		h.logger.InfoContext(ctx, "Leaderboard update failed",
			attr.ExtractCorrelationID(ctx),
			attr.Any("failure_payload", failurePayload),
		)

		return []handlerwrapper.Result{
			{Topic: leaderboardevents.LeaderboardUpdateFailedV1, Payload: failurePayload},
		}, nil
	}

	// Handle success outcome - returns multiple results
	if result.Success != nil {
		successPayload, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}
		h.logger.InfoContext(ctx, "Leaderboard updated successfully",
			attr.ExtractCorrelationID(ctx),
		)

		// Create tag update notification payload
		tagPayload := leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
			GuildID:     payload.GuildID,
			RoundID:     payload.RoundID,
			Source:      "leaderboard_update",
			UpdatedAt:   time.Now().UTC(),
			ChangedTags: extractChangedTags(assignments),
		}

		return []handlerwrapper.Result{
			{Topic: leaderboardevents.LeaderboardUpdatedV1, Payload: successPayload},
			{Topic: leaderboardevents.TagUpdateForScheduledRoundsV1, Payload: tagPayload},
		}, nil
	}

	return nil, errors.New("leaderboard update service returned unexpected result")
}

// extractChangedTags converts tag assignments to a map for tag update notifications
func extractChangedTags(assignments []sharedtypes.TagAssignmentRequest) map[sharedtypes.DiscordID]sharedtypes.TagNumber {
	out := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(assignments))
	for _, a := range assignments {
		out[a.UserID] = a.TagNumber
	}
	return out
}
