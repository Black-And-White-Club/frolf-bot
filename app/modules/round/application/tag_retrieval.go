package roundservice

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Service Functions for Tag Retrieval Flow --

// RequestTagNumber initiates the tag number retrieval process.
func (s *RoundService) RequestTagNumber(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.TagNumberRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagNumberRequestPayload: %w", err)
	}

	// Publish "round.tag.number.request" event
	if err := s.publishEvent(msg, roundevents.RoundTagNumberRequest, roundevents.TagNumberRequestPayload{
		UserID: eventPayload.UserID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.request event", map[string]interface{}{})
		return fmt.Errorf("failed to publish round.tag.number.request event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.request event", map[string]interface{}{"user_id": eventPayload.UserID})
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
		UserID: eventPayload.UserID,
	}

	// Publish the request to the leaderboard service using publishEvent
	if err := s.publishEvent(msg, roundevents.LeaderboardGetTagNumberRequest, leaderboardRequestPayload); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish GetTagNumberRequest to leaderboard", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish GetTagNumberRequest to leaderboard: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published GetTagNumberRequest to leaderboard", map[string]interface{}{"user_id": eventPayload.UserID})
	return nil
}

// HandleTagNumberResponse handles the leaderboard.tag.number.response event.
func (s *RoundService) TagNumberResponse(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.GetTagNumberResponsePayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagNumberResponsePayload: %w", err)
	}

	roundIDStr := msg.Metadata.Get("RoundID")
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to convert RoundID to int64: %w", err)
	}

	if eventPayload.Error != "" {
		// Handle error (publish round.tag.retrieval.error or similar)
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Received error from leaderboard", map[string]interface{}{"error": eventPayload.Error})
		// Publish round.tag.number.notfound event
		if err := s.publishEvent(msg, roundevents.RoundTagNumberNotFound, roundevents.RoundTagNumberNotFoundPayload{
			UserID: eventPayload.UserID,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.notfound event", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("failed to publish round.tag.number.notfound event: %w", err)
		}
		return nil
	}

	// If tag number is found, add participant to the round
	if eventPayload.TagNumber != nil {
		if err := s.publishEvent(msg, roundevents.RoundTagNumberFound, roundevents.RoundTagNumberFoundPayload{
			RoundID:   roundtypes.ID(roundID),
			UserID:    eventPayload.UserID,
			TagNumber: eventPayload.TagNumber,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.found event", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("failed to publish round.tag.number.found event: %w", err)
		}
	} else {
		// Handle case where tag number is not found
		if err := s.publishEvent(msg, roundevents.RoundTagNumberNotFound, roundevents.RoundTagNumberNotFoundPayload{
			UserID: eventPayload.UserID,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.notfound event", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("failed to publish round.tag.number.notfound event: %w", err)
		}
	}

	return nil
}
