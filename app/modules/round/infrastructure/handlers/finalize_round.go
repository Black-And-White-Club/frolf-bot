package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleAllScoresSubmitted(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleAllScoresSubmitted",
		&roundevents.AllScoresSubmittedPayload{},
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

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					finalizeResult.Failure,
					roundevents.RoundFinalizationError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			h.logger.InfoContext(ctx, "Backend round finalization successful", attr.CorrelationIDFromMsg(msg))

			// Use the round data from the payload instead of fetching again
			fetchedRound := &allScoresSubmittedPayload.RoundData

			// Create Discord finalization message
			discordFinalizationPayload := roundevents.RoundFinalizedEmbedUpdatePayload{
				RoundID:        allScoresSubmittedPayload.RoundID,
				Title:          fetchedRound.Title,
				StartTime:      fetchedRound.StartTime,
				Location:       fetchedRound.Location,
				Participants:   allScoresSubmittedPayload.Participants,
				EventMessageID: fetchedRound.EventMessageID,
			}

			discordFinalizedMsg, err := h.helpers.CreateResultMessage(
				msg,
				&discordFinalizationPayload,
				roundevents.DiscordRoundFinalized,
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to create DiscordRoundFinalized message",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create DiscordRoundFinalized message: %w", err)
			}

			// CREATE BACKEND FINALIZED MESSAGE WITH PARTICIPANTS FROM PAYLOAD
			backendFinalizationPayload := roundevents.RoundFinalizedPayload{
				GuildID: allScoresSubmittedPayload.GuildID,
				RoundID: allScoresSubmittedPayload.RoundID,
				RoundData: roundtypes.Round{
					ID:             fetchedRound.ID,
					Title:          fetchedRound.Title,
					Description:    fetchedRound.Description,
					Location:       fetchedRound.Location,
					StartTime:      fetchedRound.StartTime,
					EventMessageID: fetchedRound.EventMessageID,
					CreatedBy:      fetchedRound.CreatedBy,
					State:          fetchedRound.State,
					// USE PARTICIPANTS FROM THE PAYLOAD, NOT FROM fetchedRound
					Participants: allScoresSubmittedPayload.Participants, // This contains the scores!
				},
			}

			backendFinalizedMsg, err := h.helpers.CreateResultMessage(
				msg,
				&backendFinalizationPayload,
				roundevents.RoundFinalized, // This triggers HandleRoundFinalized
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to create RoundFinalized message",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create RoundFinalized message: %w", err)
			}

			h.logger.InfoContext(ctx, "Publishing parallel messages for round finalization",
				attr.CorrelationIDFromMsg(msg),
				attr.String("discord_topic", roundevents.DiscordRoundFinalized),
				attr.String("backend_topic", roundevents.RoundFinalized),
				attr.Int("participants_with_scores", len(allScoresSubmittedPayload.Participants)),
			)

			// Return BOTH messages
			return []*message.Message{discordFinalizedMsg, backendFinalizedMsg}, nil
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

				// Create success message to publish - this should contain the ProcessRoundScoresRequestPayload
				successPayload, ok := result.Success.(*roundevents.ProcessRoundScoresRequestPayload)
				if !ok {
					h.logger.ErrorContext(ctx, "Unexpected success payload type from NotifyScoreModule",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, fmt.Errorf("unexpected success payload type: %T", result.Success)
				}

				// Log the payload data to debug the issue
				h.logger.InfoContext(ctx, "Publishing ProcessRoundScoresRequest",
					attr.CorrelationIDFromMsg(msg),
					attr.String("round_id", successPayload.RoundID.String()),
					attr.Int("num_scores", len(successPayload.Scores)),
				)

				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					roundevents.ProcessRoundScoresRequest,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from NotifyScoreModule service (neither success nor failure)",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service (neither success nor failure)")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
