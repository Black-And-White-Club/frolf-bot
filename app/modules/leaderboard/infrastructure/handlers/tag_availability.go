package leaderboardhandlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleTagAvailabilityCheckRequested handles the TagAvailabilityCheckRequested event.
func (h *LeaderboardHandlers) HandleTagAvailabilityCheckRequested(msg *message.Message) ([]*message.Message, error) {
	// DEBUG: Handler entry
	fmt.Println("DEBUG: Entered HandleTagAvailabilityCheckRequested handler")
	h.logger.Info("DEBUG: Entered HandleTagAvailabilityCheckRequested handler")

	wrappedHandler := h.handlerWrapper(
		"HandleTagAvailabilityCheckRequested",
		&leaderboardevents.TagAvailabilityCheckRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// DEBUG: Handler wrapper entry
			fmt.Println("DEBUG: Inside handlerWrapper for TagAvailabilityCheckRequested")
			defer func() {
				if r := recover(); r != nil {
					h.logger.ErrorContext(ctx, "Panic in HandleTagAvailabilityCheckRequested", attr.Any("panic", r))
					fmt.Println("DEBUG: Panic in HandleTagAvailabilityCheckRequested", r)
				}
			}()

			tagAvailabilityCheckRequestedPayload := payload.(*leaderboardevents.TagAvailabilityCheckRequestedPayload)

			h.logger.InfoContext(ctx, "Received TagAvailabilityCheckRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(tagAvailabilityCheckRequestedPayload.UserID)),
				attr.Int("tag_number", int(*tagAvailabilityCheckRequestedPayload.TagNumber)),
			)

			// DEBUG: About to call CheckTagAvailability
			fmt.Println("DEBUG: About to call CheckTagAvailability", tagAvailabilityCheckRequestedPayload.GuildID, tagAvailabilityCheckRequestedPayload.UserID, *tagAvailabilityCheckRequestedPayload.TagNumber)
			h.logger.InfoContext(ctx, "DEBUG: About to call CheckTagAvailability",
				attr.String("guild_id", string(tagAvailabilityCheckRequestedPayload.GuildID)),
				attr.String("user_id", string(tagAvailabilityCheckRequestedPayload.UserID)),
				attr.Int("tag_number", int(*tagAvailabilityCheckRequestedPayload.TagNumber)),
			)

			result, failure, err := h.leaderboardService.CheckTagAvailability(ctx, tagAvailabilityCheckRequestedPayload.GuildID, *tagAvailabilityCheckRequestedPayload)

			// DEBUG: After CheckTagAvailability
			fmt.Println("DEBUG: Returned from CheckTagAvailability", result, failure, err)
			h.logger.InfoContext(ctx, "DEBUG: Returned from CheckTagAvailability",
				attr.Any("result", result),
				attr.Any("failure", failure),
				attr.Any("error", err),
			)

			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle TagAvailabilityCheckRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle TagAvailabilityCheckRequested event: %w", err)
			}

			if failure != nil {
				failure.GuildID = tagAvailabilityCheckRequestedPayload.GuildID // Patch: propagate guild_id
				h.logger.InfoContext(ctx, "Tag availability check failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", failure),
				)

				// Create failure message
				failureMsg, errMsg := h.Helpers.CreateResultMessage(
					msg,
					failure,
					leaderboardevents.TagAvailableCheckFailure,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			h.logger.InfoContext(ctx, "Tag availability check successful", attr.CorrelationIDFromMsg(msg))

			// Create success message to publish
			if result.Available {
				result.GuildID = tagAvailabilityCheckRequestedPayload.GuildID // Patch: propagate guild_id
				h.logger.InfoContext(ctx, "Tag is available",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(result.UserID)),
					attr.Int("tag_number", int(*result.TagNumber)),
				)

				// DEBUG: Before CreateResultMessage for User
				fmt.Println("DEBUG: Before CreateResultMessage for User", result)
				h.logger.InfoContext(ctx, "DEBUG: Before CreateResultMessage for User", attr.Any("result", result))

				createUser, err := h.Helpers.CreateResultMessage(
					msg,
					result,
					leaderboardevents.TagAvailable,
				)

				// DEBUG: After CreateResultMessage for User
				fmt.Println("DEBUG: After CreateResultMessage for User", createUser, err)
				h.logger.InfoContext(ctx, "DEBUG: After CreateResultMessage for User", attr.Any("createUser", createUser), attr.Any("error", err))

				if err != nil {
					h.logger.ErrorContext(ctx, "Error in CreateResultMessage for User", attr.Any("error", err))
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// DEBUG: Before CreateResultMessage for AssignTag
				fmt.Println("DEBUG: Before CreateResultMessage for AssignTag")
				h.logger.InfoContext(ctx, "DEBUG: Before CreateResultMessage for AssignTag")

				assignTag, err := h.Helpers.CreateResultMessage(
					msg,
					&sharedevents.BatchTagAssignmentRequestedPayload{
						// Ensure GuildID is propagated for downstream leaderboard processing
						ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: tagAvailabilityCheckRequestedPayload.GuildID},
						RequestingUserID: "system", // User creation is system-initiated
						BatchID:          uuid.New().String(),
						Assignments: []sharedevents.TagAssignmentInfo{
							{
								UserID:    result.UserID,
								TagNumber: *result.TagNumber,
							},
						},
					},
					sharedevents.LeaderboardBatchTagAssignmentRequested,
				)

				// DEBUG: After CreateResultMessage for AssignTag
				fmt.Println("DEBUG: After CreateResultMessage for AssignTag", assignTag, err)
				h.logger.InfoContext(ctx, "DEBUG: After CreateResultMessage for AssignTag", attr.Any("assignTag", assignTag), attr.Any("error", err))

				if err != nil {
					h.logger.ErrorContext(ctx, "Error in CreateResultMessage for AssignTag", attr.Any("error", err))
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{createUser, assignTag}, nil
			} else {
				// Patch: propagate guild_id in TagUnavailablePayload
				tagUnavailable := &leaderboardevents.TagUnavailablePayload{
					UserID:    result.UserID,
					TagNumber: result.TagNumber,
					Reason:    result.Reason,
					GuildID:   tagAvailabilityCheckRequestedPayload.GuildID,
				}
				h.logger.InfoContext(ctx, "Tag is not available",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(result.UserID)),
					attr.Int("tag_number", int(*result.TagNumber)),
				)

				// Create tag not available message
				tagNotAvailableMsg, err := h.Helpers.CreateResultMessage(
					msg,
					tagUnavailable,
					leaderboardevents.TagUnavailable,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create tag not available message: %w", err)
				}

				return []*message.Message{tagNotAvailableMsg}, nil
			}
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
