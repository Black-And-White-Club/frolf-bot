package score

import (
	"fmt"
	"log"

	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/handlers"
	scorequeries "github.com/Black-And-White-Club/tcr-bot/app/modules/score/queries"
	scorerouter "github.com/Black-And-White-Club/tcr-bot/app/modules/score/router"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
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
	scoreCommandRouter := scorerouter.NewScoreCommandRouter(scoreCommandBus)

	scoreQueryService := scorequeries.NewScoreQueryService(dbService.ScoreDB)

	messageHandler := NewScoreHandlers(scoreCommandRouter, scoreQueryService, pubsub)

	return &ScoreModule{
		CommandRouter:  scoreCommandRouter,
		QueryService:   scoreQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// GetHandlers returns the handlers registered for the ScoreModule
func (m *ScoreModule) GetHandlers() map[string]types.Handler {
	return map[string]types.Handler{
		"score_update_handler": {
			Topic:         scorehandlers.TopicUpdateScores,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: scorehandlers.TopicUpdateScores + "_response",
		},
		"score_get_handler": {
			Topic:         scorehandlers.TopicGetScore,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: scorehandlers.TopicGetScore + "_response",
		},
	}
}

// RegisterHandlers registers the score module's handlers.
func (m *ScoreModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := m.GetHandlers()

	for handlerName, h := range handlers {
		log.Printf("Registering handler: %s with topic %s", handlerName, string(h.Topic)) // Log handler registration

		if err := router.AddHandler(
			handlerName,
			string(h.Topic),
			pubsub,
			h.ResponseTopic,
			pubsub,
			h.Handler,
		); err != nil {
			log.Printf("Failed to register handler %s: %v", handlerName, err) // Log registration error
			return fmt.Errorf("failed to register %s handler: %v", handlerName, err)
		}
	}

	return nil
}
