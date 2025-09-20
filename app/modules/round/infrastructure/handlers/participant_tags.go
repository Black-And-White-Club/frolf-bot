package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleScheduledRoundTagUpdate(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScheduledRoundTagUpdate",
		&map[string]interface{}{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagUpdateMap := payload.(*map[string]interface{})

			h.logger.InfoContext(ctx, "Received ScheduledRoundTagUpdate event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("source", getStringFromMap(tagUpdateMap, "source")),
				attr.String("batch_id", getStringFromMap(tagUpdateMap, "batch_id")),
			)

			// Extract guild_id if present
			var guildID sharedtypes.GuildID
			if gidRaw, ok := (*tagUpdateMap)["guild_id"]; ok {
				if gidStr, ok := gidRaw.(string); ok && gidStr != "" {
					guildID = sharedtypes.GuildID(gidStr)
				}
			}

			// Convert the map to the service payload format (changed tags)
			changedTags := make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber)

			if changedTagsRaw, ok := (*tagUpdateMap)["changed_tags"]; ok {
				if changedTagsMap, ok := changedTagsRaw.(map[string]interface{}); ok {
					for userID, tagNumberRaw := range changedTagsMap {
						switch v := tagNumberRaw.(type) {
						case float64:
							tagNumber := sharedtypes.TagNumber(v)
							changedTags[sharedtypes.DiscordID(userID)] = &tagNumber
						case int:
							tagNumber := sharedtypes.TagNumber(v)
							changedTags[sharedtypes.DiscordID(userID)] = &tagNumber
						default:
							h.logger.WarnContext(ctx, "Unexpected tag number type",
								attr.String("user_id", userID),
								attr.Any("tag_number", tagNumberRaw),
								attr.String("type", fmt.Sprintf("%T", tagNumberRaw)),
							)
						}
					}
				}
			}

			h.logger.InfoContext(ctx, "Converted changed tags",
				attr.CorrelationIDFromMsg(msg),
				attr.Int("changed_tags_count", len(changedTags)),
			)

			if len(changedTags) == 0 {
				h.logger.InfoContext(ctx, "No valid tag changes found, skipping update")
				return nil, nil
			}

			// Create the service payload
			servicePayload := roundevents.ScheduledRoundTagUpdatePayload{
				GuildID:     guildID,
				ChangedTags: changedTags,
			}

			if guildID == "" {
				h.logger.WarnContext(ctx, "ScheduledRoundTagUpdate received without guild_id; backend will treat as no-op",
					attr.CorrelationIDFromMsg(msg),
				)
			} else {
				h.logger.InfoContext(ctx, "Prepared service payload for scheduled round tag update",
					attr.CorrelationIDFromMsg(msg),
					attr.String("guild_id", string(guildID)),
					attr.Int("changed_tags_count", len(changedTags)),
				)
			}

			// Call the service function to handle the event
			result, err := h.roundService.UpdateScheduledRoundsWithNewTags(ctx, servicePayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ScheduledRoundTagUpdate event",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to handle ScheduledRoundTagUpdate event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Scheduled round tag update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				// FOLLOW THE SAME PATTERN AS REMINDER HANDLER - Extract and log the success payload details
				tagsUpdatedPayload := result.Success.(*roundevents.TagsUpdatedForScheduledRoundsPayload)

				h.logger.InfoContext(ctx, "Scheduled round tag update processed successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.Int("total_rounds_processed", tagsUpdatedPayload.Summary.TotalRoundsProcessed),
					attr.Int("rounds_updated", tagsUpdatedPayload.Summary.RoundsUpdated),
					attr.Int("participants_updated", tagsUpdatedPayload.Summary.ParticipantsUpdated),
				)

				// Log each round that will be updated (similar to reminder handler logging participants)
				for _, roundInfo := range tagsUpdatedPayload.UpdatedRounds {
					h.logger.InfoContext(ctx, "Round requires Discord embed update",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", roundInfo.RoundID),
						attr.String("round_title", string(roundInfo.Title)),
						attr.String("event_message_id", roundInfo.EventMessageID),
						attr.Int("total_participants", len(roundInfo.UpdatedParticipants)),
						attr.Int("participants_with_tag_changes", roundInfo.ParticipantsChanged),
					)
				}

				// Only publish Discord update if there are rounds to update
				if len(tagsUpdatedPayload.UpdatedRounds) > 0 {
					successMsg, err := h.helpers.CreateResultMessage(
						msg,
						tagsUpdatedPayload, // Pass the extracted payload, not result.Success
						roundevents.TagsUpdatedForScheduledRounds,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create success message: %w", err)
					}
					return []*message.Message{successMsg}, nil
				} else {
					// No rounds to update, but processing was successful
					h.logger.InfoContext(ctx, "Tag update processed but no rounds require Discord updates",
						attr.CorrelationIDFromMsg(msg),
						attr.Int("total_rounds_processed", tagsUpdatedPayload.Summary.TotalRoundsProcessed),
					)
					return []*message.Message{}, nil
				}
			}

			// This should never happen now that service always returns Success or Failure
			h.logger.ErrorContext(ctx, "Unexpected result from UpdateScheduledRoundsWithNewTags service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("service returned neither success nor failure")
		},
	)

	return wrappedHandler(msg)
}

// Helper function to safely extract string values from the map
func getStringFromMap(m *map[string]interface{}, key string) string {
	if value, ok := (*m)[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
