package round

import (
	"fmt"

	roundqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/round/queries"
	roundrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/round/router"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
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

	messageHandler := NewRoundHandlers(roundCommandRouter, dbService.RoundDB, pubsub) // Use dbService.RoundDB

	return &RoundModule{
		CommandRouter:  roundCommandRouter,
		QueryService:   roundQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// GetHandlers returns the handlers for the round module.
func (m *RoundModule) GetHandlers() map[string]types.Handler {
	return map[string]types.Handler{
		// "round_create_handler": {
		// 	Topic:         roundhandlers.TopicCreateRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicCreateRound + "_response",
		// },
		// "round_get_handler": {
		// 	Topic:         roundhandlers.TopicGetRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicGetRound + "_response",
		// },
		// "round_get_rounds_handler": {
		// 	Topic:         roundhandlers.TopicGetRounds,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicGetRounds + "_response",
		// },
		// "round_edit_handler": {
		// 	Topic:         roundhandlers.TopicEditRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicEditRound + "_response",
		// },
		// "round_delete_handler": {
		// 	Topic:         roundhandlers.TopicDeleteRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicDeleteRound + "_response",
		// },
		// "round_update_participant_handler": {
		// 	Topic:         roundhandlers.TopicUpdateParticipant,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicUpdateParticipant + "_response",
		// },
		// "round_join_handler": {
		// 	Topic:         roundhandlers.TopicJoinRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicJoinRound + "_response",
		// },
		// "round_submit_score_handler": {
		// 	Topic:         roundhandlers.TopicSubmitScore,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicSubmitScore + "_response",
		// },
		// "round_start_handler": {
		// 	Topic:         roundhandlers.TopicStartRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicStartRound + "_response",
		// },
		// "round_record_scores_handler": {
		// 	Topic:         roundhandlers.TopicRecordScores,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicRecordScores + "_response",
		// },
		// "round_process_score_submission_handler": {
		// 	Topic:         roundhandlers.TopicProcessScoreSubmission,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicProcessScoreSubmission + "_response",
		// },
		// "round_finalize_handler": {
		// 	Topic:         roundhandlers.TopicFinalizeRound,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicFinalizeRound + "_response",
		// },
		// "round_reminder_handler": {
		// 	Topic:         roundhandlers.TopicRoundReminder,
		// 	Handler:       m.messageHandler.Handle,
		// 	ResponseTopic: roundhandlers.TopicRoundReminder + "_response",
		// },
	}
}

// RegisterHandlers registers the round module's handlers.
func (m *RoundModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := m.GetHandlers()

	for handlerName, h := range handlers {
		if err := router.AddHandler(
			handlerName,
			string(h.Topic),
			pubsub,
			h.ResponseTopic,
			pubsub,
			h.Handler,
		); err != nil {
			return fmt.Errorf("failed to register %s handler: %v", handlerName, err)
		}
	}

	return nil
}
