package round

import (
	"context"
	"encoding/json"
	"fmt"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	roundrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/round/router"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetRoundRequest represents the incoming request to get a round.
type GetRoundRequest struct {
	RoundID int64 `json:"round_id"`
}

// GetRoundResponse is the response for GetRound
type GetRoundResponse struct {
	Round *rounddb.Round `json:"round"`
}

// GetRoundsRequest represents the incoming request to get all rounds.
type GetRoundsRequest struct{}

// GetRoundsResponse is the response for GetRounds
type GetRoundsResponse struct {
	Rounds []*rounddb.Round `json:"rounds"`
}

// RoundHandlers defines the handlers for round-related events.
type RoundHandlers struct {
	commandRouter roundrouter.CommandRouter
	roundDB       rounddb.RoundDB // Use RoundDB instead of QueryService
	pubsub        watermillutil.PubSuber
}

// NewRoundHandlers creates a new RoundHandlers instance.
func NewRoundHandlers(commandRouter roundrouter.CommandRouter, roundDB rounddb.RoundDB, pubsub watermillutil.PubSuber) *RoundHandlers {
	return &RoundHandlers{
		commandRouter: commandRouter,
		roundDB:       roundDB, // Use RoundDB instead of QueryService
		pubsub:        pubsub,
	}
}

// Handle implements the MessageHandler interface.
func (h *RoundHandlers) Handle(msg *message.Message) ([]*message.Message, error) {
	switch msg.Metadata.Get("topic") {
	case roundhandlers.TopicCreateRound:
		return h.HandleCreateRound(msg)
	case roundhandlers.TopicGetRound:
		return h.HandleGetRound(msg)
	case roundhandlers.TopicGetRounds:
		return h.HandleGetRounds(msg)
	case roundhandlers.TopicEditRound:
		return h.HandleEditRound(msg)
	case roundhandlers.TopicDeleteRound:
		return h.HandleDeleteRound(msg)
	case roundhandlers.TopicUpdateParticipant:
		return h.HandleUpdateParticipant(msg)
	case roundhandlers.TopicJoinRound:
		return h.HandleJoinRound(msg)
	case roundhandlers.TopicSubmitScore:
		return h.HandleSubmitScore(msg)
	case roundhandlers.TopicStartRound:
		return h.HandleStartRound(msg)
	case roundhandlers.TopicRecordScores:
		return h.HandleRecordScores(msg)
	case roundhandlers.TopicProcessScoreSubmission:
		return h.HandleProcessScoreSubmission(msg)
	case roundhandlers.TopicFinalizeRound:
		return h.HandleFinalizeRound(msg)
	default:
		return nil, fmt.Errorf("unknown message topic: %s", msg.Metadata.Get("topic"))
	}
}

// HandleCreateRound creates a new round.
func (h *RoundHandlers) HandleCreateRound(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.CreateRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.CreateRound(context.Background(), req.Input); err != nil { // Call CreateRound on commandRouter
		return nil, fmt.Errorf("failed to create round: %w", err)
	}

	return nil, nil
}

// HandleGetRound retrieves a round.
func (h *RoundHandlers) HandleGetRound(msg *message.Message) ([]*message.Message, error) {
	var req GetRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	round, err := h.roundDB.GetRound(context.Background(), req.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}

	if round == nil {
		return nil, fmt.Errorf("round not found: %d", req.RoundID)
	}

	response := GetRoundResponse{Round: round}
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal round response: %w", err)
	}

	respMsg := message.NewMessage(watermill.NewUUID(), payload)
	respMsg.Metadata.Set("topic", roundhandlers.TopicGetRoundResponse)

	return []*message.Message{respMsg}, nil
}

// HandleGetRounds retrieves all rounds.
func (h *RoundHandlers) HandleGetRounds(msg *message.Message) ([]*message.Message, error) {
	var req GetRoundsRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	rounds, err := h.roundDB.GetRounds(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get rounds: %w", err)
	}

	response := GetRoundsResponse{Rounds: rounds}
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rounds response: %w", err)
	}

	respMsg := message.NewMessage(watermill.NewUUID(), payload)
	respMsg.Metadata.Set("topic", roundhandlers.TopicGetRoundsResponse)

	return []*message.Message{respMsg}, nil
}

// HandleEditRound edits an existing round.
func (h *RoundHandlers) HandleEditRound(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.EditRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.EditRound(context.Background(), req.RoundID, req.DiscordID, req.APIInput); err != nil {
		return nil, fmt.Errorf("failed to edit round: %w", err)
	}

	return nil, nil
}

// HandleDeleteRound deletes a round.
func (h *RoundHandlers) HandleDeleteRound(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.DeleteRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.DeleteRound(context.Background(), req.RoundID); err != nil {
		return nil, fmt.Errorf("failed to delete round: %w", err)
	}

	return nil, nil
}

// HandleUpdateParticipant updates a participant in a round.
func (h *RoundHandlers) HandleUpdateParticipant(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.UpdateParticipantRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.UpdateParticipant(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to update participant: %w", err)
	}

	return nil, nil
}

// HandleJoinRound adds a participant to a round.
func (h *RoundHandlers) HandleJoinRound(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.JoinRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.JoinRound(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to join round: %w", err)
	}

	return nil, nil
}

// HandleSubmitScore submits a score for a participant in a round.
func (h *RoundHandlers) HandleSubmitScore(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.SubmitScoreRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.SubmitScore(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to submit score: %w", err)
	}

	return nil, nil
}

// HandleStartRound starts a round.
func (h *RoundHandlers) HandleStartRound(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.StartRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.StartRound(context.Background(), req.RoundID); err != nil {
		return nil, fmt.Errorf("failed to start round: %w", err)
	}

	return nil, nil
}

// HandleRecordScores records scores for a round.
func (h *RoundHandlers) HandleRecordScores(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.RecordScoresRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.RecordRoundScores(context.Background(), req.RoundID); err != nil {
		return nil, fmt.Errorf("failed to record scores: %w", err)
	}

	return nil, nil
}

// HandleProcessScoreSubmission processes a score submission.
func (h *RoundHandlers) HandleProcessScoreSubmission(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.ProcessScoreSubmissionRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.ProcessScoreSubmission(context.Background(), req.Input); err != nil {
		return nil, fmt.Errorf("failed to process score submission: %w", err)
	}

	return nil, nil
}

// HandleFinalizeRound finalizes a round.
func (h *RoundHandlers) HandleFinalizeRound(msg *message.Message) ([]*message.Message, error) {
	var req roundcommands.FinalizeRoundRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.FinalizeAndProcessScores(context.Background(), req.RoundID); err != nil {
		return nil, fmt.Errorf("failed to finalize round: %w", err)
	}

	return nil, nil
}
