package leaderboardhandlers

import (
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleTagAssignmentRequested handles the TagAssignmentRequested event.
func (h *LeaderboardHandlers) HandleTagAssignmentRequested(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAssignmentRequestedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAssignmentRequestedPayload: %w", err)
	}

	h.logger.Info("Received TagAssignmentRequested event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.UserID)),
		slog.Int("tag_number", *payload.TagNumber),
	)

	// Call the service function to handle the event
	if err := h.leaderboardService.TagAssignmentRequested(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle TagAssignmentRequested event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle TagAssignmentRequested event: %w", err)
	}

	h.logger.Info("TagAssignmentRequested event processed", slog.String("correlation_id", correlationID))
	return nil
}

// HandleTagAssigned handles the TagAssigned event.
func (h *LeaderboardHandlers) HandleTagAssigned(msg *message.Message) error {
	h.logger.Info("HandleTagAssigned triggered",
		slog.String("correlation_id", msg.Metadata.Get(middleware.CorrelationIDMetadataKey)),
		slog.String("message_id", msg.UUID),
	)

	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAssignedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAssignedPayload: %w", err)
	}

	h.logger.Info("Received TagAssigned event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.UserID)),
		slog.Int("tag_number", *payload.TagNumber),
		slog.String("assignment_id", payload.AssignmentID),
	)

	// Call the service function to publish TagAvailable to User module
	if err := h.leaderboardService.PublishTagAvailable(msg.Context(), msg, &payload); err != nil {
		h.logger.Error("Failed to publish TagAvailable event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish TagAvailable event: %w", err)
	}

	h.logger.Info("TagAssigned event processed", slog.String("correlation_id", correlationID))
	return nil
}
