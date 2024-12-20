package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/service"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	LeaderboardService leaderboardservice.Service
	Publisher          message.Publisher
	logger             watermill.LoggerAdapter
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers.
func NewLeaderboardHandlers(leaderboardService leaderboardservice.Service, publisher message.Publisher, logger watermill.LoggerAdapter) *LeaderboardHandlers {
	return &LeaderboardHandlers{
		LeaderboardService: leaderboardService,
		Publisher:          publisher,
		logger:             logger,
	}
}

// HandleLeaderboardUpdate handles LeaderboardUpdateEvent events.
func (h *LeaderboardHandlers) HandleLeaderboardUpdate(ctx context.Context, msg *message.Message) error {
	msg.Ack()
	var event leaderboardevents.LeaderboardUpdateEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal LeaderboardUpdateEvent", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal LeaderboardUpdateEvent: %w", err)
	}

	h.logger.Info("HandleLeaderboardUpdate called", watermill.LogFields{
		"message_id":   msg.UUID,
		"event_detail": fmt.Sprintf("%+v", event),
	})

	// Use the context from the message
	ctx = msg.Context()

	err := h.LeaderboardService.UpdateLeaderboard(ctx, event)
	if err != nil {
		h.logger.Error("Failed to process leaderboard update", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to process leaderboard update: %w", err)
	}

	h.logger.Info("HandleLeaderboardUpdate completed", watermill.LogFields{
		"message_id": msg.UUID,
	})
	return nil
}

// HandleTagAssigned handles TagAssigned events.
func (h *LeaderboardHandlers) HandleTagAssigned(ctx context.Context, msg *message.Message) error {
	msg.Ack()
	var event leaderboardevents.TagAssigned
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal TagAssigned", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal TagAssigned: %w", err)
	}

	h.logger.Info("HandleTagAssigned called", watermill.LogFields{
		"message_id":   msg.UUID,
		"event_detail": fmt.Sprintf("%+v", event),
	})

	// Use the context from the message
	ctx = msg.Context()

	err := h.LeaderboardService.AssignTag(ctx, event)
	if err != nil {
		h.logger.Error("Failed to process tag assignment", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to process tag assignment: %w", err)
	}

	h.logger.Info("HandleTagAssigned completed", watermill.LogFields{
		"message_id": msg.UUID,
	})
	return nil
}

// HandleTagSwapRequest handles TagSwapRequest events.
func (h *LeaderboardHandlers) HandleTagSwapRequest(ctx context.Context, msg *message.Message) error {
	msg.Ack()
	var event leaderboardevents.TagSwapRequest
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		h.logger.Error("Failed to unmarshal TagSwapRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal TagSwapRequest: %w", err)
	}

	h.logger.Info("HandleTagSwapRequest called", watermill.LogFields{
		"message_id":   msg.UUID,
		"event_detail": fmt.Sprintf("%+v", event),
	})

	// Use the context from the message
	ctx = msg.Context()

	err := h.LeaderboardService.SwapTags(ctx, event.RequestorID, event.TargetID)
	if err != nil {
		h.logger.Error("Failed to process tag swap request", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to process tag swap request: %w", err)
	}

	h.logger.Info("HandleTagSwapRequest completed", watermill.LogFields{
		"message_id": msg.UUID,
	})
	return nil
}

// HandleGetLeaderboardRequest handles GetLeaderboardRequest events.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(ctx context.Context, msg *message.Message) error {
	msg.Ack()
	h.logger.Info("HandleGetLeaderboardRequest called", watermill.LogFields{
		"message_id": msg.UUID,
	})

	// Use the context from the message
	ctx = msg.Context()

	leaderboard, err := h.LeaderboardService.GetLeaderboard(ctx)
	if err != nil {
		h.logger.Error("Failed to get leaderboard", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Publish the response event
	resp := leaderboardevents.GetLeaderboardResponse{Leaderboard: leaderboard}
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal GetLeaderboardResponse", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to marshal GetLeaderboardResponse: %w", err)
	}

	if err := h.Publisher.Publish(leaderboardevents.GetLeaderboardResponseSubject, message.NewMessage(watermill.NewUUID(), respData)); err != nil {
		h.logger.Error("Failed to publish GetLeaderboardResponse", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to publish GetLeaderboardResponse: %w", err)
	}

	h.logger.Info("HandleGetLeaderboardRequest completed", watermill.LogFields{
		"message_id": msg.UUID,
	})
	return nil
}

// HandleGetTagByDiscordIDRequest handles GetTagByDiscordIDRequest events.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDRequest(ctx context.Context, msg *message.Message) error {
	msg.Ack()
	var req leaderboardevents.GetTagByDiscordIDRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal GetTagByDiscordIDRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal GetTagByDiscordIDRequest: %w", err)
	}

	h.logger.Info("HandleGetTagByDiscordIDRequest called", watermill.LogFields{
		"message_id":   msg.UUID,
		"request_data": fmt.Sprintf("%+v", req),
	})

	// Use the context from the message
	ctx = msg.Context()

	tag, err := h.LeaderboardService.GetTagByDiscordID(ctx, req.DiscordID)
	if err != nil {
		h.logger.Error("Failed to get tag by Discord ID", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to get tag by Discord ID: %w", err)
	}

	// Publish the response event
	resp := leaderboardevents.GetTagByDiscordIDResponse{TagNumber: strconv.Itoa(tag)} // Convert tag to string
	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal GetTagByDiscordIDResponse", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to marshal GetTagByDiscordIDResponse: %w", err)
	}

	if err := h.Publisher.Publish(leaderboardevents.GetTagByDiscordIDResponseSubject, message.NewMessage(watermill.NewUUID(), respData)); err != nil {
		h.logger.Error("Failed to publish GetTagByDiscordIDResponse", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to publish GetTagByDiscordIDResponse: %w", err)
	}

	h.logger.Info("HandleGetTagByDiscordIDRequest completed", watermill.LogFields{
		"message_id": msg.UUID,
	})
	return nil
}

// HandleCheckTagAvailabilityRequest handles CheckTagAvailabilityRequest events.
func (h *LeaderboardHandlers) HandleCheckTagAvailabilityRequest(ctx context.Context, msg *message.Message) error {
	msg.Ack()
	h.logger.Info("HandleCheckTagAvailabilityRequest called", watermill.LogFields{
		"message_id":     msg.UUID,
		"correlation_id": msg.Metadata.Get("correlation_id"),
	})

	var req leaderboardevents.CheckTagAvailabilityRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		h.logger.Error("Failed to unmarshal CheckTagAvailabilityRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal CheckTagAvailabilityRequest: %w", err)
	}

	fmt.Printf("[DEBUG] HandleCheckTagAvailabilityRequest: Request details: %+v\n", req)

	// Use the context from the message
	ctx = msg.Context()

	isAvailable, err := h.LeaderboardService.CheckTagAvailability(ctx, req.TagNumber)
	if err != nil {
		h.logger.Error("Failed to check tag availability", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to check tag availability: %w", err)
	}

	fmt.Printf("[DEBUG] HandleCheckTagAvailabilityRequest: isAvailable: %+v\n", isAvailable)

	// Publish the response event
	resp := leaderboardevents.CheckTagAvailabilityResponse{
		IsAvailable: isAvailable,
	}

	respData, err := json.Marshal(resp)
	if err != nil {
		h.logger.Error("Failed to marshal CheckTagAvailabilityResponse", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to marshal CheckTagAvailabilityResponse: %w", err)
	}

	// Create a new message for the response, using the correlation ID from the request
	respMsg := message.NewMessage(watermill.NewUUID(), respData)
	respMsg.Metadata.Set("correlation_id", msg.Metadata.Get("correlation_id"))

	h.logger.Info("Publishing CheckTagAvailabilityResponse", watermill.LogFields{
		"message_id":   respMsg.UUID, // Use the response message ID
		"is_available": resp.IsAvailable,
	})

	if err := h.Publisher.Publish(leaderboardevents.CheckTagAvailabilityResponseSubject, respMsg); err != nil {
		h.logger.Error("Failed to publish CheckTagAvailabilityResponse", err, watermill.LogFields{
			"message_id": respMsg.UUID, // Use the response message ID
		})
		return fmt.Errorf("failed to publish CheckTagAvailabilityResponse: %w", err)
	}

	fmt.Printf("[DEBUG] HandleCheckTagAvailabilityRequest: Response published successfully\n")

	h.logger.Info("HandleCheckTagAvailabilityRequest completed", watermill.LogFields{
		"message_id": msg.UUID,
	})
	return nil
}
