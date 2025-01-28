package roundservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/shared/logging"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Service Functions for Tag Retrieval Flow --

const defaultGetTagNumberTimeout = 5 * time.Second // Or whatever timeout you deem appropriate

// RequestTagNumber initiates the tag number retrieval process.
func (s *RoundService) RequestTagNumber(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.TagNumberRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagNumberRequestPayload: %w", err)
	}

	// Validate input, set defaults, etc.
	if eventPayload.Timeout == 0 {
		eventPayload.Timeout = defaultGetTagNumberTimeout
	}

	// Publish "round.tag.number.request" event
	if err := s.publishEvent(msg, roundevents.RoundTagNumberRequest, roundevents.TagNumberRequestPayload{
		DiscordID: eventPayload.DiscordID,
		Timeout:   eventPayload.Timeout,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.request event", map[string]interface{}{})
		return fmt.Errorf("failed to publish round.tag.number.request event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.request event", map[string]interface{}{"discord_id": eventPayload.DiscordID})
	return nil
}

// HandleTagNumberRequest handles the round.tag.number.request event.
func (s *RoundService) TagNumberRequest(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.TagNumberRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberRequestPayload: %w", err)
	}

	s.logger.Info("Handling round.tag.number.request event",
		slog.String("correlation_id", correlationID),
		slog.String("event", "round.tag.number.request"),
	)

	// Prepare the request payload for the leaderboard
	leaderboardRequestPayload := roundevents.TagNumberRequestPayload{
		DiscordID: eventPayload.DiscordID,
	}

	// Publish the request to the leaderboard service using publishEvent
	if err := s.publishEvent(msg, roundevents.LeaderboardGetTagNumberRequest, leaderboardRequestPayload); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish GetTagNumberRequest to leaderboard", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish GetTagNumberRequest to leaderboard: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published GetTagNumberRequest to leaderboard", map[string]interface{}{"discord_id": eventPayload.DiscordID})
	return nil
}

// HandleTagNumberResponse handles the leaderboard.tag.number.response event.
func (s *RoundService) TagNumberResponse(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.GetTagNumberResponsePayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagNumberResponsePayload: %w", err)
	}

	if eventPayload.Error != "" {
		// Handle error (publish round.tag.retrieval.error or similar)
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Received error from leaderboard", map[string]interface{}{
			"error": eventPayload.Error,
		})
		// ... publish a round.tag.retrieval.error event if needed ...
		return fmt.Errorf("error from leaderboard: %s", eventPayload.Error)
	}

	RoundID := msg.Metadata.Get("RoundID")
	if eventPayload.TagNumber != 0 {
		// Publish round.tag.number.found event
		if err := s.publishEvent(msg, roundevents.RoundTagNumberFound, roundevents.RoundTagNumberFoundPayload{
			RoundID:   RoundID,
			DiscordID: eventPayload.DiscordID,
			TagNumber: eventPayload.TagNumber,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.found event", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("failed to publish round.tag.number.found event: %w", err)
		}

		logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.found event", map[string]interface{}{
			"tag_number": eventPayload.TagNumber,
			"discord_id": eventPayload.DiscordID,
		})
	} else {
		// Publish round.tag.number.notfound event (or handle as needed)
		if err := s.publishEvent(msg, roundevents.RoundTagNumberNotFound, roundevents.RoundTagNumberNotFoundPayload{
			DiscordID: eventPayload.DiscordID,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.notfound event", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("failed to publish round.tag.number.notfound event: %w", err)
		}

		logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.notfound event", map[string]interface{}{
			"discord_id": eventPayload.DiscordID,
		})
	}

	return nil
}
