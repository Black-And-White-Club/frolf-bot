package leaderboard

import (
	"fmt"

	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/handlers"
	leaderboardqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/queries"
	leaderboardrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/router"
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

func (m *LeaderboardModule) GetHandlers() map[string]struct {
	topic         string
	handler       message.HandlerFunc
	responseTopic string
} {
	return map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}{
		"leaderboard_get_handler": {
			topic:         leaderboardhandlers.TopicGetLeaderboard,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicGetLeaderboard + "_response",
		},
		"leaderboard_update_handler": {
			topic:         leaderboardhandlers.TopicUpdateLeaderboard,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicUpdateLeaderboard + "_response",
		},
		"leaderboard_receive_scores_handler": {
			topic:         leaderboardhandlers.TopicReceiveScores,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicReceiveScores + "_response",
		},
		"leaderboard_assign_tags_handler": {
			topic:         leaderboardhandlers.TopicAssignTags,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicAssignTags + "_response",
		},
		"leaderboard_initiate_tag_swap_handler": {
			topic:         leaderboardhandlers.TopicInitiateTagSwap,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicInitiateTagSwap + "_response",
		},
		"leaderboard_swap_groups_handler": {
			topic:         leaderboardhandlers.TopicSwapGroups,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicSwapGroups + "_response",
		},
	}
}

// RegisterHandlers registers the leaderboard module's handlers.
func (m *LeaderboardModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}{
		"leaderboard_get_handler": {
			topic:         leaderboardhandlers.TopicGetLeaderboard,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicGetLeaderboard + "_response",
		},
		"leaderboard_update_handler": {
			topic:         leaderboardhandlers.TopicUpdateLeaderboard,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicUpdateLeaderboard + "_response",
		},
		"leaderboard_receive_scores_handler": {
			topic:         leaderboardhandlers.TopicReceiveScores,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicReceiveScores + "_response",
		},
		"leaderboard_assign_tags_handler": {
			topic:         leaderboardhandlers.TopicAssignTags,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicAssignTags + "_response",
		},
		"leaderboard_initiate_tag_swap_handler": {
			topic:         leaderboardhandlers.TopicInitiateTagSwap,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicInitiateTagSwap + "_response",
		},
		"leaderboard_swap_groups_handler": {
			topic:         leaderboardhandlers.TopicSwapGroups,
			handler:       m.messageHandler.Handle,
			responseTopic: leaderboardhandlers.TopicSwapGroups + "_response",
		},
	}

	for handlerName, h := range handlers {
		if err := router.AddHandler(
			handlerName,
			h.topic,
			m.PubSub,
			h.responseTopic,
			m.PubSub,
			h.handler,
		); err != nil {
			return fmt.Errorf("failed to register %s handler: %v", handlerName, err)
		}
	}

	return nil
}
