package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/queries"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetTagHandler handles retrieving tag information.
type GetTagHandler struct {
	queryService leaderboardqueries.LeaderboardQueryService
}

// NewGetTagHandler creates a new GetTagHandler.
func NewGetTagHandler(queryService leaderboardqueries.LeaderboardQueryService) *GetTagHandler {
	return &GetTagHandler{queryService: queryService}
}

// Handle retrieves tag information based on the request type.
func (h *GetTagHandler) Handle(ctx context.Context, msg *message.Message) error {
	switch msg.Metadata.Get("request_type") {
	case "get_user_tag":
		return h.handleGetUserTag(ctx, msg)
	case "check_tag_taken":
		return h.handleCheckTagTaken(ctx, msg)
	case "get_participant_tag":
		return h.handleGetParticipantTag(ctx, msg)
	default:
		return fmt.Errorf("unknown tag request type: %s", msg.Metadata.Get("request_type"))
	}
}

func (h *GetTagHandler) handleGetUserTag(ctx context.Context, msg *message.Message) error {
	var query leaderboardqueries.GetUserTagQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return fmt.Errorf("failed to unmarshal GetUserTagQuery: %w", err)
	}

	tagNumber, err := h.queryService.GetUserTag(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get user tag: %w", err)
	}

	// Publish the response (e.g., UserTagResponseEvent)
	// ...

	return nil
}

func (h *GetTagHandler) handleCheckTagTaken(ctx context.Context, msg *message.Message) error {
	var query leaderboardqueries.CheckTagTakenQuery
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		return fmt.Errorf("failed to unmarshal CheckTagTakenQuery: %w", err)
	}

	isTaken, err := h.queryService.IsTagTaken(ctx, query.TagNumber)
	if err != nil {
		return fmt.Errorf("failed to check if tag is taken: %w", err)
	}

	// Publish the response (e.g., TagTakenResponseEvent)
	// ...

	return nil
}

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
	// ...

	return nil
}
