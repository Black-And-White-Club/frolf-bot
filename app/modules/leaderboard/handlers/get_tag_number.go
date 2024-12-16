package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/queries"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetTagHandler handles retrieving tag information.
type GetTagHandler struct {
	eventBus     watermillutil.PubSuber
	queryService leaderboardqueries.QueryService
}

// NewGetTagHandler creates a new GetTagHandler.
func NewGetTagHandler(eventBus watermillutil.PubSuber, queryService leaderboardqueries.QueryService) *GetTagHandler {
	return &GetTagHandler{
		eventBus:     eventBus,
		queryService: queryService,
	}
}

// Handle retrieves tag information based on the request type.
func (h *GetTagHandler) Handle(ctx context.Context, msg *message.Message) error {
	// Check the message topic to determine the request type
	switch msg.Metadata.Get("topic") {
	case TopicGetLeaderboardTag:
		return h.handleGetTagRequest(ctx, msg)
	case TopicCheckLeaderboardTag:
		return h.handleCheckTagTakenRequest(ctx, msg)
	case TopicGetLeaderboard:
		return h.handleGetParticipantTag(ctx, msg)
	default:
		return fmt.Errorf("unknown tag request type: %s", msg.Metadata.Get("request_type"))
	}
}

// handleGetTagRequest handles requests to get a user's tag.
func (h *GetTagHandler) handleGetTagRequest(ctx context.Context, msg *message.Message) error {
	var query leaderboardqueries.GetUserTagQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return fmt.Errorf("failed to unmarshal GetUserTagQuery: %w", err)
	}

	tagNumber, err := h.queryService.GetUserTag(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get user tag: %w", err)
	}

	// Publish the response (e.g., UserTagResponseEvent)
	responseEvent := UserTagResponseEvent{
		UserID:    query.UserID,
		TagNumber: tagNumber,
	}
	payload, err := json.Marshal(responseEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal UserTagResponseEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicUserTagResponse, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish UserTagResponseEvent: %w", err)
	}

	return nil
}

// handleCheckTagTakenRequest handles requests to check if a tag is taken.
func (h *GetTagHandler) handleCheckTagTakenRequest(ctx context.Context, msg *message.Message) error {
	var query leaderboardqueries.CheckTagTakenQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return fmt.Errorf("failed to unmarshal CheckTagTakenQuery: %w", err)
	}

	isTaken, err := h.queryService.IsTagTaken(ctx, query.TagNumber)
	if err != nil {
		return fmt.Errorf("failed to check if tag is taken: %w", err)
	}

	// Publish the response (e.g., TagAvailabilityResponse)
	response := TagAvailabilityResponse{
		TagNumber:   query.TagNumber,
		IsAvailable: !isTaken,
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal TagAvailabilityResponse: %w", err)
	}

	if err := h.eventBus.Publish(TopicTagAvailabilityResponse, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish TagAvailabilityResponse: %w", err)
	}

	return nil
}

// handleGetParticipantTag handles requests to get a participant's tag.
func (h *GetTagHandler) handleGetParticipantTag(ctx context.Context, msg *message.Message) error {
	var query leaderboardqueries.GetParticipantTagQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return fmt.Errorf("failed to unmarshal GetParticipantTagQuery: %w", err)
	}

	tagNumber, err := h.queryService.GetParticipantTag(ctx, query.ParticipantID)
	if err != nil {
		return fmt.Errorf("failed to get participant tag: %w", err)
	}

	// Publish the response (e.g., ParticipantTagResponseEvent)
	response := ParticipantTagResponseEvent{
		ParticipantID: query.ParticipantID,
		TagNumber:     tagNumber,
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal ParticipantTagResponseEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicParticipantTagResponse, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish ParticipantTagResponseEvent: %w", err)
	}

	return nil
}
