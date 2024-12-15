package round

import (
	"fmt"

	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/round/queries"
	roundrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/round/router"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundModule represents the round module.
type RoundModule struct {
	CommandRouter  roundrouter.CommandRouter
	QueryService   roundqueries.QueryService
	PubSub         watermillutil.PubSuber
	messageHandler *RoundHandlers
}

// NewRoundModule creates a new RoundModule with the provided dependencies.
func NewRoundModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*RoundModule, error) {
	marshaler := watermillutil.Marshaler
	roundCommandBus := roundrouter.NewRoundCommandBus(pubsub, marshaler)
	roundCommandRouter := roundrouter.NewRoundCommandRouter(roundCommandBus)

	roundQueryService := roundqueries.NewRoundQueryService(dbService.RoundDB)

	messageHandler := NewRoundHandlers(roundCommandRouter, roundQueryService, pubsub)

	return &RoundModule{
		CommandRouter:  roundCommandRouter,
		QueryService:   roundQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// RegisterHandlers registers the round module's handlers.
func (m *RoundModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}{
		"round_create_handler": {
			topic:         roundhandlers.TopicCreateRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicCreateRound + "_response",
		},
		"round_get_handler": {
			topic:         roundhandlers.TopicGetRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicGetRound + "_response",
		},
		"round_get_rounds_handler": {
			topic:         roundhandlers.TopicGetRounds,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicGetRounds + "_response",
		},
		"round_edit_handler": {
			topic:         roundhandlers.TopicEditRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicEditRound + "_response",
		},
		"round_delete_handler": {
			topic:         roundhandlers.TopicDeleteRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicDeleteRound + "_response",
		},
		"round_update_participant_handler": {
			topic:         roundhandlers.TopicUpdateParticipant,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicUpdateParticipant + "_response",
		},
		"round_join_handler": {
			topic:         roundhandlers.TopicJoinRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicJoinRound + "_response",
		},
		"round_submit_score_handler": {
			topic:         roundhandlers.TopicSubmitScore,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicSubmitScore + "_response",
		},
		"round_start_handler": {
			topic:         roundhandlers.TopicStartRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicStartRound + "_response",
		},
		"round_record_scores_handler": {
			topic:         roundhandlers.TopicRecordScores,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicRecordScores + "_response",
		},
		"round_process_score_submission_handler": {
			topic:         roundhandlers.TopicProcessScoreSubmission,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicProcessScoreSubmission + "_response",
		},
		"round_finalize_handler": {
			topic:         roundhandlers.TopicFinalizeRound,
			handler:       m.messageHandler.Handle,
			responseTopic: roundhandlers.TopicFinalizeRound + "_response",
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
