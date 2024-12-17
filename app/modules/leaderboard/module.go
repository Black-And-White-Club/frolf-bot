package leaderboard

import (
	"fmt"

	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/handlers"
	leaderboardqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/queries"
	leaderboardrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/router"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardModule represents the leaderboard module.
type LeaderboardModule struct {
	CommandRouter  leaderboardrouter.CommandRouter
	QueryService   leaderboardqueries.QueryService
	PubSub         watermillutil.PubSuber
	messageHandler *LeaderboardHandlers
}

// NewLeaderboardModule creates a new LeaderboardModule with the provided dependencies.
func NewLeaderboardModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*LeaderboardModule, error) {
	marshaler := watermillutil.Marshaler
	leaderboardCommandBus := leaderboardrouter.NewLeaderboardCommandBus(pubsub, marshaler)
	leaderboardCommandRouter := leaderboardrouter.NewLeaderboardCommandRouter(leaderboardCommandBus)

	leaderboardQueryService := leaderboardqueries.NewLeaderboardQueryService(dbService.LeaderboardDB)

	messageHandler := NewLeaderboardHandlers(leaderboardCommandRouter, leaderboardQueryService, pubsub)

	return &LeaderboardModule{
		CommandRouter:  leaderboardCommandRouter,
		QueryService:   leaderboardQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// GetHandlers returns the handlers for the leaderboard module.
func (m *LeaderboardModule) GetHandlers() map[string]types.Handler {
	return map[string]types.Handler{
		"leaderboard_get_handler": {
			Topic:         leaderboardhandlers.TopicGetLeaderboard,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: leaderboardhandlers.TopicGetLeaderboard + "_response",
		},
		"leaderboard_update_handler": {
			Topic:         leaderboardhandlers.TopicUpdateLeaderboard,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: leaderboardhandlers.TopicUpdateLeaderboard + "_response",
		},
		"leaderboard_receive_scores_handler": {
			Topic:         leaderboardhandlers.TopicReceiveScores,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: leaderboardhandlers.TopicReceiveScores + "_response",
		},
		"leaderboard_assign_tags_handler": {
			Topic:         leaderboardhandlers.TopicAssignTags,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: leaderboardhandlers.TopicAssignTags + "_response",
		},
		"leaderboard_initiate_tag_swap_handler": {
			Topic:         leaderboardhandlers.TopicInitiateTagSwap,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: leaderboardhandlers.TopicInitiateTagSwap + "_response",
		},
		"leaderboard_swap_groups_handler": {
			Topic:         leaderboardhandlers.TopicSwapGroups,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: leaderboardhandlers.TopicSwapGroups + "_response",
		},
	}
}

// RegisterHandlers registers the leaderboard module's handlers.
func (m *LeaderboardModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := m.GetHandlers()

	for handlerName, h := range handlers {
		if err := router.AddHandler(
			handlerName,
			string(h.Topic),
			pubsub, // Use the pubsub argument here
			h.ResponseTopic,
			pubsub, // Use the pubsub argument here
			h.Handler,
		); err != nil {
			return fmt.Errorf("failed to register %s handler: %v", handlerName, err)
		}
	}

	return nil
}
