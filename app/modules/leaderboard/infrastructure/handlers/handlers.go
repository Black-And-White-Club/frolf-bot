package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/application"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardHandlers handles leaderboard-related leaderboardevents.
type LeaderboardHandlers struct {
	LeaderboardService leaderboardservice.Service
	EventBus           shared.EventBus
	logger             *slog.Logger
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers.
func NewLeaderboardHandlers(leaderboardService leaderboardservice.Service, eventBus shared.EventBus, logger *slog.Logger) *LeaderboardHandlers {
	return &LeaderboardHandlers{
		LeaderboardService: leaderboardService,
		EventBus:           eventBus,
		logger:             logger,
	}
}

// HandleLeaderboardUpdate handles LeaderboardUpdateEvent leaderboardevents.
func (h *LeaderboardHandlers) HandleLeaderboardUpdate(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event leaderboardevents.LeaderboardUpdateEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal LeaderboardUpdateEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal LeaderboardUpdateEvent: %w", err)
	}

	h.logger.Info("HandleLeaderboardUpdate called", "message_id", msg.UUID, "event", event)

	if err := h.LeaderboardService.UpdateLeaderboard(ctx, event); err != nil {
		h.logger.Error("Failed to process leaderboard update", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to process leaderboard update: %w", err)
	}

	h.logger.Info("HandleLeaderboardUpdate completed", "message_id", msg.UUID)
	return nil
}

// HandleTagAssigned handles TagAssignedEvent events.
func (h *LeaderboardHandlers) HandleTagAssigned(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	// --- Add logging ---
	h.logger.Info("HandleTagAssigned called", "message_id", msg.UUID)

	var event leaderboardevents.TagAssignedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal TagAssignedEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal TagAssignedEvent: %w", err)
	}

	h.logger.Info("TagAssignedEvent received", "event", event)

	if err := h.LeaderboardService.AssignTag(ctx, event); err != nil {
		h.logger.Error("Failed to process tag assignment", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to process tag assignment: %w", err)
	}

	h.logger.Info("HandleTagAssigned completed", "message_id", msg.UUID)
	return nil
}

// HandleTagSwapRequest handles TagSwapRequestEvent leaderboardevents.
func (h *LeaderboardHandlers) HandleTagSwapRequest(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event leaderboardevents.TagSwapRequestEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal TagSwapRequestEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal TagSwapRequestEvent: %w", err)
	}

	h.logger.Info("HandleTagSwapRequest called", "message_id", msg.UUID, "event", event)

	if err := h.LeaderboardService.SwapTags(ctx, event.RequestorID, event.TargetID); err != nil {
		h.logger.Error("Failed to process tag swap request", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to process tag swap request: %w", err)
	}

	h.logger.Info("HandleTagSwapRequest completed", "message_id", msg.UUID)
	return nil
}

// HandleGetLeaderboardRequest handles GetLeaderboardRequestEvent leaderboardevents.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	h.logger.Info("HandleGetLeaderboardRequest called", "message_id", msg.UUID)

	leaderboard, err := h.LeaderboardService.GetLeaderboard(ctx)
	if err != nil {
		h.logger.Error("Failed to get leaderboard", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	resp := leaderboardevents.GetLeaderboardResponseEvent{Leaderboard: leaderboard}
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal GetLeaderboardResponseEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to marshal GetLeaderboardResponseEvent: %w", err)
	}

	if err := h.publishEvent(ctx, leaderboardevents.GetLeaderboardResponseSubject, leaderboardevents.LeaderboardStreamName, respData); err != nil {
		h.logger.Error("Failed to publish GetLeaderboardResponseEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to publish GetLeaderboardResponseEvent: %w", err)
	}

	h.logger.Info("HandleGetLeaderboardRequest completed", "message_id", msg.UUID)
	return nil
}

// HandleGetTagByDiscordIDRequest handles GetTagByDiscordIDRequestEvent leaderboardevents.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDRequest(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var req leaderboardevents.GetTagByDiscordIDRequestEvent
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal GetTagByDiscordIDRequestEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to unmarshal GetTagByDiscordIDRequestEvent: %w", err)
	}

	h.logger.Info("HandleGetTagByDiscordIDRequest called", "message_id", msg.UUID, "request", req)

	tag, err := h.LeaderboardService.GetTagByDiscordID(ctx, req.DiscordID)
	if err != nil {
		h.logger.Error("Failed to get tag by Discord ID", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to get tag by Discord ID: %w", err)
	}

	resp := leaderboardevents.GetTagByDiscordIDResponseEvent{TagNumber: strconv.Itoa(tag)}
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal GetTagByDiscordIDResponseEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to marshal GetTagByDiscordIDResponseEvent: %w", err)
	}

	if err := h.publishEvent(ctx, leaderboardevents.GetTagByDiscordIDResponseSubject, leaderboardevents.LeaderboardStreamName, respData); err != nil {
		h.logger.Error("Failed to publish GetTagByDiscordIDResponseEvent", "error", err, "message_id", msg.UUID)
		return fmt.Errorf("failed to publish GetTagByDiscordIDResponseEvent: %w", err)
	}

	h.logger.Info("HandleGetTagByDiscordIDRequest completed", "message_id", msg.UUID)
	return nil
}

// HandleCheckTagAvailabilityRequest handles CheckTagAvailabilityRequestEvent leaderboardevents.
func (h *LeaderboardHandlers) HandleCheckTagAvailabilityRequest(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	h.logger.Info("HandleCheckTagAvailabilityRequest called",
		slog.String("message_id", msg.UUID),
		slog.String("correlation_id", msg.Metadata.Get("correlation_id")),
	)

	// Log received metadata
	h.logger.Debug("Received CheckTagAvailabilityRequest metadata",
		slog.String("message_id", msg.UUID),
		slog.Any("metadata", msg.Metadata),
	)

	// Unmarshal the request
	var req leaderboardevents.CheckTagAvailabilityRequestEvent
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal CheckTagAvailabilityRequestEvent",
			slog.Any("error", err),
			slog.String("message_id", msg.UUID),
		)
		return fmt.Errorf("failed to unmarshal CheckTagAvailabilityRequestEvent: %w", err)
	}

	// Check tag availability
	isAvailable, err := h.LeaderboardService.CheckTagAvailability(ctx, req.TagNumber)
	if err != nil {
		h.logger.Error("Failed to check tag availability",
			slog.Any("error", err),
			slog.String("message_id", msg.UUID),
		)
		return fmt.Errorf("failed to check tag availability: %w", err)
	}

	// Prepare the response
	resp := leaderboardevents.CheckTagAvailabilityResponseEvent{IsAvailable: isAvailable}
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal CheckTagAvailabilityResponseEvent",
			slog.Any("error", err),
			slog.String("message_id", msg.UUID),
		)
		return fmt.Errorf("failed to marshal CheckTagAvailabilityResponseEvent: %w", err)
	}

	// Publish the response
	if err := h.publishEvent(ctx, leaderboardevents.CheckTagAvailabilityResponseSubject, leaderboardevents.UserStreamName, respData); err != nil {
		h.logger.Error("Failed to publish CheckTagAvailabilityResponseEvent",
			slog.Any("error", err),
			slog.String("message_id", msg.UUID),
		)
		return fmt.Errorf("failed to publish CheckTagAvailabilityResponseEvent: %w", err)
	}

	h.logger.Info("HandleCheckTagAvailabilityRequest completed", slog.String("message_id", msg.UUID))
	return nil
}

func (h *LeaderboardHandlers) publishEvent(ctx context.Context, subject string, streamName string, payload []byte) error {
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("subject", subject)

	if err := h.EventBus.Publish(ctx, streamName, msg); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
