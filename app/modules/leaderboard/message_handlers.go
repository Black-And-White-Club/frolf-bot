package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/handlers"
	leaderboardqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/queries"
	leaderboardrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/router"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardHandlers defines the handlers for leaderboard-related events.
type LeaderboardHandlers struct {
	commandRouter leaderboardrouter.CommandRouter
	queryService  leaderboardqueries.QueryService
	pubsub        watermillutil.PubSuber
}

// NewLeaderboardHandlers creates a new LeaderboardHandlers instance.
func NewLeaderboardHandlers(commandRouter leaderboardrouter.CommandRouter, queryService leaderboardqueries.QueryService, pubsub watermillutil.PubSuber) *LeaderboardHandlers {
	return &LeaderboardHandlers{
		commandRouter: commandRouter,
		queryService:  queryService,
		pubsub:        pubsub,
	}
}

// Handle implements the MessageHandler interface.
func (h *LeaderboardHandlers) Handle(msg *message.Message) ([]*message.Message, error) {
	switch msg.Metadata.Get("topic") {
	case leaderboardhandlers.TopicGetLeaderboard:
		return h.HandleGetLeaderboard(msg)
	case leaderboardhandlers.TopicUpdateLeaderboard:
		return h.HandleUpdateLeaderboard(msg)
	case leaderboardhandlers.TopicReceiveScores:
		return h.HandleReceiveScores(msg)
	case leaderboardhandlers.TopicAssignTag:
		return h.HandleAssignTags(msg)
	case leaderboardhandlers.TopicInitiateTagSwap:
		return h.HandleInitiateTagSwap(msg)
	case leaderboardhandlers.TopicSwapGroups:
		return h.HandleSwapGroups(msg)
	default:
		return nil, fmt.Errorf("unknown message topic: %s", msg.Metadata.Get("topic"))
	}
}

// HandleGetLeaderboard handles the GetLeaderboardRequest command.
func (h *LeaderboardHandlers) HandleGetLeaderboard(msg *message.Message) ([]*message.Message, error) {
	var req leaderboardcommands.GetLeaderboardRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.GetLeaderboard(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	return nil, nil
}

// HandleUpdateLeaderboard handles the UpdateLeaderboardRequest command.
func (h *LeaderboardHandlers) HandleUpdateLeaderboard(msg *message.Message) ([]*message.Message, error) {
	var req leaderboardcommands.UpdateLeaderboardRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.UpdateLeaderboard(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to update leaderboard: %w", err)
	}

	return nil, nil
}

// HandleReceiveScores handles the ReceiveScoresRequest command.
func (h *LeaderboardHandlers) HandleReceiveScores(msg *message.Message) ([]*message.Message, error) {
	var req leaderboardcommands.ReceiveScoresRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.ReceiveScores(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to receive scores: %w", err)
	}

	return nil, nil
}

// HandleAssignTags handles the AssignTagsRequest command.
func (h *LeaderboardHandlers) HandleAssignTags(msg *message.Message) ([]*message.Message, error) {
	var req leaderboardcommands.AssignTagsRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.AssignTags(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to assign tags: %w", err)
	}

	return nil, nil
}

// HandleInitiateTagSwap handles the InitiateTagSwapRequest command.
func (h *LeaderboardHandlers) HandleInitiateTagSwap(msg *message.Message) ([]*message.Message, error) {
	var req leaderboardcommands.InitiateTagSwapRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.InitiateTagSwap(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to initiate tag swap: %w", err)
	}

	return nil, nil
}

// HandleSwapGroups handles the SwapGroupsRequest command.
func (h *LeaderboardHandlers) HandleSwapGroups(msg *message.Message) ([]*message.Message, error) {
	var req leaderboardcommands.SwapGroupsRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.SwapGroups(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to swap groups: %w", err)
	}

	return nil, nil
}
