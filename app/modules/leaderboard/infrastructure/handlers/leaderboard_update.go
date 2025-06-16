package leaderboardhandlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// extractChangedTagsMap converts tag assignments to a simple map for cross-module communication
func extractChangedTagsMap(assignments []sharedtypes.TagAssignmentRequest) map[string]int {
	result := make(map[string]int, len(assignments))
	for _, assignment := range assignments {
		result[string(assignment.UserID)] = int(assignment.TagNumber)
	}
	return result
}

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
// This is for score processing after round completion - updates leaderboard with new participant tags.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateRequested(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper("HandleLeaderboardUpdateRequested", &leaderboardevents.LeaderboardUpdateRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			requestPayload := payload.(*leaderboardevents.LeaderboardUpdateRequestedPayload)

			h.logger.InfoContext(ctx, "Received LeaderboardUpdateRequested event from score processing",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", requestPayload.RoundID),
				attr.Int("participant_count", len(requestPayload.SortedParticipantTags)),
			)

			// Parse tag:userID format
			assignments := make([]sharedtypes.TagAssignmentRequest, len(requestPayload.SortedParticipantTags))
			for i, tagUserPair := range requestPayload.SortedParticipantTags {
				parts := strings.Split(tagUserPair, ":")
				if len(parts) != 2 {
					h.logger.ErrorContext(ctx, "Invalid tag format in score processing",
						attr.CorrelationIDFromMsg(msg),
						attr.String("tag_pair", tagUserPair),
					)
					return nil, fmt.Errorf("invalid tag format: %s", tagUserPair)
				}

				tagNumber, err := strconv.Atoi(parts[0])
				if err != nil {
					h.logger.ErrorContext(ctx, "Failed to parse tag number",
						attr.CorrelationIDFromMsg(msg),
						attr.String("tag_number", parts[0]),
						attr.Any("error", err),
					)
					return nil, fmt.Errorf("failed to parse tag number: %w", err)
				}

				assignments[i] = sharedtypes.TagAssignmentRequest{
					UserID:    sharedtypes.DiscordID(parts[1]),
					TagNumber: sharedtypes.TagNumber(tagNumber),
				}
			}

			// Use the public interface method
			result, err := h.leaderboardService.ProcessTagAssignments(
				ctx,
				sharedtypes.ServiceUpdateSourceProcessScores,
				assignments,
				nil,                               // System operation
				uuid.UUID(requestPayload.RoundID), // Use roundID as operation ID
				uuid.New(),                        // Generate batch ID
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Service failed to handle leaderboard update",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", requestPayload.RoundID),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to process score-based tag assignments: %w", err)
			}

			// Handle failure response
			if result.Failure != nil {
				failureMsg, err := h.Helpers.CreateResultMessage(msg, result.Failure, leaderboardevents.LeaderboardUpdateFailed)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Handle success response
			if result.Success != nil {
				// Existing LeaderboardUpdated publication
				successMsg, err := h.Helpers.CreateResultMessage(msg, result.Success, leaderboardevents.LeaderboardUpdated)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// NEW: Also publish TagUpdateForScheduledRounds to notify round module
				tagUpdatePayload := map[string]interface{}{
					"changed_tags": extractChangedTagsMap(assignments),
					"updated_at":   time.Now().UTC(),
					"source":       "leaderboard_update",
					"round_id":     requestPayload.RoundID,
				}

				tagUpdateMsg, err := h.Helpers.CreateResultMessage(msg, tagUpdatePayload, leaderboardevents.TagUpdateForScheduledRounds)
				if err != nil {
					h.logger.WarnContext(ctx, "Failed to create tag update message for scheduled rounds", attr.Error(err))
					// Still return the leaderboard success even if this fails
					return []*message.Message{successMsg}, nil
				}

				h.logger.InfoContext(ctx, "Publishing tag updates to scheduled rounds",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", requestPayload.RoundID),
					attr.Int("changed_tags", len(assignments)),
				)

				return []*message.Message{successMsg, tagUpdateMsg}, nil
			}

			return nil, fmt.Errorf("unexpected service result: neither success nor failure set")
		},
	)(msg)
}
