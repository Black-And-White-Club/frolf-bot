package score

import (
	"fmt"

	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/handlers"
	scorequeries "github.com/Black-And-White-Club/tcr-bot/app/modules/score/queries"
	scorerouter "github.com/Black-And-White-Club/tcr-bot/app/modules/score/router"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreModule represents the score module.
type ScoreModule struct {
	CommandRouter  scorerouter.CommandRouter
	QueryService   scorequeries.ScoreQueryService
	PubSub         watermillutil.PubSuber
	messageHandler *ScoreHandlers
}

// NewScoreModule creates a new ScoreModule with the provided dependencies.
func NewScoreModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*ScoreModule, error) {
	marshaler := watermillutil.Marshaler
	scoreCommandBus := scorerouter.NewScoreCommandBus(pubsub, marshaler)
	scoreCommandRouter := scorerouter.NewScoreCommandRouter(scoreCommandBus) // This line was missing

	scoreQueryService := scorequeries.NewScoreQueryService(dbService.ScoreDB)

	messageHandler := NewScoreHandlers(scoreCommandRouter, scoreQueryService, pubsub)

	return &ScoreModule{
		CommandRouter:  scoreCommandRouter,
		QueryService:   scoreQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// RegisterHandlers registers the score module's handlers.
func (m *ScoreModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	// Define handlers with their topics and response topics
	handlers := map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}{
		"score_update_handler": {
			topic:         scorehandlers.TopicUpdateScores,
			handler:       m.messageHandler.Handle,
			responseTopic: scorehandlers.TopicUpdateScores + "_response",
		},
		"score_get_handler": {
			topic:         scorehandlers.TopicGetScore,
			handler:       m.messageHandler.Handle,
			responseTopic: scorehandlers.TopicGetScore + "_response",
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
