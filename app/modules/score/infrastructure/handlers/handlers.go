package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/application"
	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreHandlers struct to hold dependencies
type ScoreHandlers struct {
	ScoreService scoreservice.Service
	EventBus     shared.EventBus
	logger       *slog.Logger
}

// NewScoreHandlers creates a new ScoreHandlers instance.
func NewScoreHandlers(service scoreservice.Service, eventBus shared.EventBus, logger *slog.Logger) *ScoreHandlers {
	return &ScoreHandlers{
		ScoreService: service,
		EventBus:     eventBus,
		logger:       logger,
	}
}

// HandleScoresReceived handles the ScoresReceivedEvent.
func (h *ScoreHandlers) HandleScoresReceived(ctx context.Context, msg *message.Message) error {
	// Logging
	h.logger.Info("HandleScoresReceived handler invoked")
	h.logger.Debug("Message received", slog.Any("msg", msg))
	h.logger.Info("HandleScoresReceived started", "contextErr", ctx.Err())
	h.logger.Info("Processing ScoresReceived", "payload", string(msg.Payload))
	h.logger.Info("Processing ScoresReceived", "message_id", msg.UUID)

	// Unmarshal the event
	var event scoreevents.ScoresReceivedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal ScoresReceivedEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal ScoresReceivedEvent: %w", err)
	}

	// Process the scores using the service
	if err := h.ScoreService.ProcessRoundScores(msg.Context(), event); err != nil {
		h.logger.Error("Failed to process round scores", "error", err, "message_id", msg.UUID, "round_id", event.RoundID)

		// Publish a ProcessedScoresEvent with an error status
		if err := h.publishEvent(ctx, scoreevents.ProcessedScoresEventSubject, &scoreevents.ProcessedScoresEvent{
			RoundID: event.RoundID,
			Scores:  event.Scores,
			Error:   err.Error(),
		}); err != nil {
			return fmt.Errorf("failed to publish ProcessedScoresEvent (error): %w", err)
		}

		return fmt.Errorf("failed to process round scores: %w", err)
	}

	// Publish a ProcessedScoresEvent with a success status
	if err := h.publishEvent(ctx, scoreevents.ProcessedScoresEventSubject, &scoreevents.ProcessedScoresEvent{
		RoundID: event.RoundID,
		Scores:  event.Scores,
		Success: true,
	}); err != nil {
		return fmt.Errorf("failed to publish ProcessedScoresEvent: %w", err)
	}

	// Logging and acknowledgment
	h.logger.Info("HandleScoresReceived completed", "message_id", msg.UUID)
	msg.Ack()
	return nil
}

// HandleScoreCorrected handles the ScoreCorrectedEvent.
func (h *ScoreHandlers) HandleScoreCorrected(ctx context.Context, msg *message.Message) error {
	// Logging
	h.logger.Info("HandleScoreCorrected handler invoked")
	h.logger.Debug("Message received", slog.Any("msg", msg))
	h.logger.Info("HandleScoreCorrected started", "contextErr", ctx.Err())
	h.logger.Info("Processing ScoreCorrected", "payload", string(msg.Payload))
	h.logger.Info("Processing ScoreCorrected", "message_id", msg.UUID)

	// Unmarshal the event
	var event scoreevents.ScoreCorrectedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal ScoreCorrectedEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal ScoreCorrectedEvent: %w", err)
	}

	// Correct the score using the service
	if err := h.ScoreService.CorrectScore(msg.Context(), event); err != nil {
		h.logger.Error("Failed to correct score", "error", err, "message_id", msg.UUID, "round_id", event.RoundID, "discord_id", event.DiscordID)

		// Publish a ScoreCorrectedEvent with an error status
		errorEvent := &scoreevents.ScoreCorrectedEvent{
			RoundID:   event.RoundID,
			DiscordID: event.DiscordID,
			NewScore:  event.NewScore,
			TagNumber: event.TagNumber,
			Error:     err.Error(),
		}
		if err := h.publishEvent(ctx, scoreevents.ScoreCorrectedEventSubject, errorEvent); err != nil {
			return fmt.Errorf("failed to publish ScoreCorrectedEvent (error): %w", err)
		}

		return fmt.Errorf("failed to correct score: %w", err)
	}

	// Publish a ScoreCorrectedEvent with a success status
	successEvent := &scoreevents.ScoreCorrectedEvent{
		RoundID:   event.RoundID,
		DiscordID: event.DiscordID,
		NewScore:  event.NewScore,
		TagNumber: event.TagNumber,
		Success:   true,
	}
	if err := h.publishEvent(ctx, scoreevents.ScoreCorrectedEventSubject, successEvent); err != nil {
		return fmt.Errorf("failed to publish ScoreCorrectedEvent (success): %w", err)
	}

	// Logging and acknowledgment
	h.logger.Info("HandleScoreCorrected completed", "message_id", msg.UUID)
	msg.Ack()
	return nil
}

func (h *ScoreHandlers) publishEvent(ctx context.Context, subject string, payload interface{}) error {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadData)
	msg.Metadata.Set("subject", subject)

	h.logger.Info("Publishing event", "subject", subject, "payload", string(payloadData))

	if err := h.EventBus.Publish(ctx, scoreevents.ScoreStreamName, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
