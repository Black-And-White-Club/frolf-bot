package leaderboardhandlers

import (
	"context"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func (h *LeaderboardHandlers) HandleBatchTagAssignmentRequested(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleBatchTagAssignmentRequested",
		&sharedevents.BatchTagAssignmentRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			batchPayload := payload.(*sharedevents.BatchTagAssignmentRequestedPayload)

			h.logger.InfoContext(ctx, "Received BatchTagAssignmentRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("batch_id", batchPayload.BatchID),
				attr.String("requesting_user", string(batchPayload.RequestingUserID)),
				attr.Int("assignment_count", len(batchPayload.Assignments)),
			)

			// Convert assignments to the expected format
			assignments := make([]sharedtypes.TagAssignmentRequest, len(batchPayload.Assignments))
			for i, assignment := range batchPayload.Assignments {
				assignments[i] = sharedtypes.TagAssignmentRequest{
					UserID:    assignment.UserID,
					TagNumber: assignment.TagNumber,
				}
			}

			batchID, err := uuid.Parse(batchPayload.BatchID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Invalid batch ID format", attr.Error(err))
				return nil, fmt.Errorf("invalid batch ID format: %w", err)
			}

			// Call service - propagate guildID
			result, err := h.leaderboardService.ProcessTagAssignments(
				ctx,
				sharedtypes.GuildID(batchPayload.GuildID), // Pass guildID explicitly, cast to correct type
				batchPayload, // Pass the entire payload for source determination
				assignments,
				&batchPayload.RequestingUserID,
				uuid.New(),
				batchID,
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Service failed to handle batch assignment", attr.Error(err))
				return nil, fmt.Errorf("failed to process batch tag assignments: %w", err)
			}

			var resultMessages []*message.Message

			// Handle failure response
			if result.Failure != nil {
				failureMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Failure,
					leaderboardevents.LeaderboardBatchTagAssignmentFailed,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Handle success response
			if result.Success != nil {
				// Always create the primary success message
				successMsg, err := h.Helpers.CreateResultMessage(
					msg,
					result.Success,
					leaderboardevents.LeaderboardBatchTagAssigned,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
				resultMessages = append(resultMessages, successMsg)

				// For Discord single assignments, also create individual response for backward compatibility
				if msg.Metadata.Get("single_assignment") == "true" &&
					msg.Metadata.Get("source") == "discord_claim" &&
					len(assignments) == 1 {

					assignment := assignments[0]
					// Parse the batchID string to RoundID
					roundID, err := uuid.Parse(batchPayload.BatchID)
					if err != nil {
						h.logger.WarnContext(ctx, "Failed to parse batch ID for individual response", attr.Error(err))
					} else {
						individualPayload := &leaderboardevents.TagAssignedPayload{
							UserID:       assignment.UserID,
							TagNumber:    &assignment.TagNumber,
							AssignmentID: sharedtypes.RoundID(roundID), // Convert properly
							Source:       msg.Metadata.Get("source"),
						}

						// Use the correct event constant that exists in your events file
						individualMsg, err := h.Helpers.CreateResultMessage(
							msg,
							individualPayload,
							leaderboardevents.LeaderboardTagAssignmentSuccess, // This exists in your events
						)
						if err != nil {
							h.logger.WarnContext(ctx, "Failed to create individual response message", attr.Error(err))
						} else {
							// Copy Discord metadata to individual message
							discordFields := []string{"user_id", "requestor_id", "channel_id", "message_id", "correlation_id"}
							for _, field := range discordFields {
								if value := msg.Metadata.Get(field); value != "" {
									individualMsg.Metadata.Set(field, value)
								}
							}
							resultMessages = append(resultMessages, individualMsg)
						}
					}
				}

				// Always publish tag updates for scheduled rounds
				changedTags := make(map[string]interface{})
				for _, assignment := range assignments {
					changedTags[string(assignment.UserID)] = assignment.TagNumber
				}

				// Determine source for tag update
				tagUpdateSource := "batch_assignment"
				if msg.Metadata.Get("source") == "discord_claim" && len(assignments) == 1 {
					tagUpdateSource = "individual_assignment"
				} else if msg.Metadata.Get("source") == "round_completion" {
					tagUpdateSource = "score_processing"
				} else if msg.Metadata.Get("source") == "admin_assign" {
					tagUpdateSource = "admin_batch"
				}

				tagUpdatePayload := map[string]interface{}{
					"changed_tags":       changedTags,
					"updated_at":         time.Now().UTC(),
					"source":             tagUpdateSource,
					"batch_id":           batchPayload.BatchID, // Keep as string
					"requesting_user_id": string(batchPayload.RequestingUserID),
				}

				tagUpdateMsg, err := h.Helpers.CreateResultMessage(msg, tagUpdatePayload, sharedevents.TagUpdateForScheduledRounds)
				if err != nil {
					h.logger.WarnContext(ctx, "Failed to create tag update message for scheduled rounds", attr.Error(err))
					return resultMessages, nil // Return what we have so far
				}

				h.logger.InfoContext(ctx, "Publishing tag updates to scheduled rounds",
					attr.CorrelationIDFromMsg(msg),
					attr.String("batch_id", batchPayload.BatchID),
					attr.Int("changed_tags", len(assignments)),
				)

				resultMessages = append(resultMessages, tagUpdateMsg)
			}

			return resultMessages, nil
		},
	)

	return wrappedHandler(msg)
}
