package userhandlers

import (
	"context"
	"errors"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error) {
	// Add explicit debug logging to stdout
	fmt.Printf("DEBUG: HandleUserSignupRequest() called with message UUID: %s\n", msg.UUID)
	h.logger.DebugContext(context.Background(), "HandleUserSignupRequest called",
		attr.String("message_uuid", msg.UUID),
		attr.String("payload", string(msg.Payload)),
	)
	wrappedHandler := h.handlerWrapper(
		"HandleUserSignupRequest",
		&userevents.UserSignupRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			fmt.Printf("DEBUG: Inside handler wrapper for UserSignupRequest\n")
			userSignupPayload := payload.(*userevents.UserSignupRequestedPayloadV1)

			userID := userSignupPayload.UserID
			guildID := userSignupPayload.GuildID

			fmt.Printf("DEBUG: Processing signup for UserID: %s, GuildID: %s\n", userID, guildID)
			h.logger.InfoContext(ctx, "Received UserSignupRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)
			if userSignupPayload.TagNumber != nil {
				tagNumber := *userSignupPayload.TagNumber

				h.logger.InfoContext(ctx, "Tag availability check requested",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Int("tag_number", int(tagNumber)),
				)

				ctx, span := h.tracer.Start(ctx, "TagAvailabilityCheck")
				defer span.End()

				eventPayload := &leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
					GuildID:   guildID,
					TagNumber: userSignupPayload.TagNumber,
					UserID:    userID,
				}

				tagAvailabilityMsg, err := h.helpers.CreateResultMessage(
					msg,
					eventPayload,
					leaderboardevents.TagAvailabilityCheckRequestedV1,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create tag availability check message: %w", err)
				}

				h.metrics.RecordTagAvailabilityCheck(ctx, true, tagNumber)

				return []*message.Message{tagAvailabilityMsg}, nil
			}

			ctx, span := h.tracer.Start(ctx, "CreateUser")
			defer span.End()

			result, err := h.userService.CreateUser(ctx, guildID, userID, nil, userSignupPayload.UDiscUsername, userSignupPayload.UDiscName)
			fmt.Printf("DEBUG: CreateUser returned: result=%#v, err=%v\n", result, err)

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.UserCreationFailedPayloadV1)
				if !ok {
					span.RecordError(errors.New("unexpected type for failure payload"))
					return nil, errors.New("unexpected type for failure payload")
				}

				h.logger.InfoContext(ctx, "User creation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				// Create failure message for user stream
				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					failedPayload,
					userevents.UserCreationFailedV1,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				fmt.Printf("DEBUG: Created failureMsg for user_id: %s, guild_id: %s, reason: %s\n", userID, guildID, failedPayload.Reason)

				// Create Discord-specific failure message
				discordFailurePayload := &userevents.UserSignupFailedPayloadV1{
					GuildID: guildID,
					Reason:  failedPayload.Reason,
				}
				discordFailureMsg, err := h.helpers.CreateResultMessage(
					msg,
					discordFailurePayload,
					userevents.UserSignupFailedV1,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create discord failure message: %w", err)
				}
				fmt.Printf("DEBUG: Created discordFailureMsg for user_id: %s, guild_id: %s, reason: %s\n", userID, guildID, failedPayload.Reason)

				h.metrics.RecordUserCreationFailure(ctx, failedPayload.Reason, "failed")

				fmt.Printf("DEBUG: Returning 2 messages from HandleUserSignupRequest (failure case)\n")
				return []*message.Message{failureMsg, discordFailureMsg}, nil
			}

			// Now check for service error after handling any failure payload
			if err != nil {
				span.RecordError(err)
				h.logger.ErrorContext(ctx, "Failed to call CreateUser service",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process UserSignupRequest service call: %w", err)
			}

			if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.UserCreatedPayloadV1)
				if !ok {
					span.RecordError(errors.New("unexpected type for success payload"))
					return nil, errors.New("unexpected type for success payload")
				}

				h.logger.InfoContext(ctx, "User creation succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
				)

				// Create the main user created message for the user stream
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.UserCreatedV1,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
				fmt.Printf("DEBUG: Created successMsg for user_id: %s, guild_id: %s\n", userID, guildID)

				// Create a Discord-specific signup success message for the discord stream
				discordSuccessMsg, err := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.UserSignupSucceededV1,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create discord success message: %w", err)
				}
				fmt.Printf("DEBUG: Created discordSuccessMsg for user_id: %s, guild_id: %s\n", userID, guildID)

				h.metrics.RecordUserCreationSuccess(ctx, string(successPayload.UserID), "discord")

				fmt.Printf("DEBUG: Returning 2 messages from HandleUserSignupRequest (success case)\n")
				return []*message.Message{successMsg, discordSuccessMsg}, nil
			}

			h.logger.WarnContext(ctx, "CreateUser returned no success or failure payload when error was nil",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)
			fmt.Printf("DEBUG: CreateUser returned no success or failure payload, err=%v\n", err)
			return nil, errors.New("user creation service returned unexpected result")
		},
	)

	return wrappedHandler(msg)
}
