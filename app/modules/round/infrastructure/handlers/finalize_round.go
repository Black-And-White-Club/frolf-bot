package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleAllScoresSubmitted(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleAllScoresSubmitted",
		&roundevents.AllScoresSubmittedPayload{}, // Expecting payload with Participants
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			allScoresSubmittedPayload, ok := payload.(*roundevents.AllScoresSubmittedPayload)
			if !ok {
				h.logger.ErrorContext(ctx, "Invalid payload type for HandleAllScoresSubmitted",
					attr.Any("payload", payload),
				)
				return nil, fmt.Errorf("invalid payload type for HandleAllScoresSubmitted")
			}

			h.logger.InfoContext(ctx, "Received AllScoresSubmitted event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", allScoresSubmittedPayload.RoundID.String()),
			)

			// Call the service function for backend finalization steps
			finalizeResult, finalizeErr := h.roundService.FinalizeRound(ctx, *allScoresSubmittedPayload)
			if finalizeErr != nil {
				h.logger.ErrorContext(ctx, "Failed during backend FinalizeRound service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", finalizeErr),
				)
				return nil, fmt.Errorf("failed during backend FinalizeRound service call: %w", finalizeErr)
			}

			if finalizeResult.Failure != nil {
				h.logger.InfoContext(ctx, "Backend round finalization failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", finalizeResult.Failure),
				)
				// Decide if a failure message should be published here
				return nil, fmt.Errorf("backend round finalization failed: %v", finalizeResult.Failure)
			}

			h.logger.InfoContext(ctx, "Backend round finalization successful", attr.CorrelationIDFromMsg(msg))

			// Fetch round details needed for the Discord finalization embed payload
			_, err := h.roundService.GetRound(ctx, allScoresSubmittedPayload.RoundID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to get round details for Discord finalization payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
					attr.RoundID("round_id", allScoresSubmittedPayload.RoundID),
				)
				return nil, fmt.Errorf("failed to get round details for Discord finalization payload: %w", err)
			}

			// Construct and publish the event to trigger Discord embed finalization display
			discordFinalizationPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        allScoresSubmittedPayload.RoundID,
				Title:          allScoresSubmittedPayload.RoundData.Title,          // Populate from fetched round data
				StartTime:      allScoresSubmittedPayload.RoundData.StartTime,      // Populate from fetched round data
				Location:       allScoresSubmittedPayload.RoundData.Location,       // Populate from fetched round data
				Participants:   allScoresSubmittedPayload.Participants,             // Use the list from the incoming event
				EventMessageID: allScoresSubmittedPayload.RoundData.EventMessageID, // Use the EventMessageID from fetched round data
			}

			discordFinalizedMsg, err := h.helpers.CreateResultMessage(
				msg,
				&discordFinalizationPayload,       // Publish the pointer
				roundevents.DiscordRoundFinalized, // Topic for the Discord App handler
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to create DiscordRoundFinalized message",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create DiscordRoundFinalized message: %w", err)
			}

			h.logger.InfoContext(ctx, "Published DiscordRoundFinalized event to trigger frontend update",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", allScoresSubmittedPayload.RoundID),
				attr.String("discord_message_id", discordFinalizationPayload.EventMessageID),
				attr.String("discord_channel_id", discordFinalizationPayload.DiscordChannelID),
			)

			// Decide what messages to return. This handler triggers backend finalization
			// and the Discord frontend update. You might return just the Discord update message
			// or also a backend RoundFinalized message if FinalizeRound published one.

			messagesToReturn := []*message.Message{discordFinalizedMsg}

			// If backend FinalizeRound service returned a success payload to be published as RoundFinalized
			// if finalizeResult.Success != nil {
			//      backendFinalizedMsg, err := h.helpers.CreateResultMessage(msg, finalizeResult.Success, roundevents.RoundFinalized)
			//      if err == nil {
			//          messagesToReturn = append(messagesToReturn, backendFinalizedMsg)
			//      } else {
			//           h.logger.ErrorContext(ctx, "Failed to create backend RoundFinalized message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
			//      }
			// }

			return messagesToReturn, nil
		},
	)
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundFinalized",
		&roundevents.RoundFinalizedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundFinalizedPayload := payload.(*roundevents.RoundFinalizedPayload)

			h.logger.InfoContext(ctx, "Received RoundFinalized event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", roundFinalizedPayload.RoundID.String()),
			)

			// Call the service function to handle the event
			result, err := h.roundService.NotifyScoreModule(ctx, *roundFinalizedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundFinalized event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundFinalized event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Notify Score Module failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundFinalizationError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Notify Score Module successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.ProcessRoundScoresRequest,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from NotifyScoreModule service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
