package score

import (
	"context"
	"encoding/json"
	"fmt"

	scorecommands "github.com/Black-And-White-Club/tcr-bot/app/modules/score/commands"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/handlers"
	scorequeries "github.com/Black-And-White-Club/tcr-bot/app/modules/score/queries"
	scorerouter "github.com/Black-And-White-Club/tcr-bot/app/modules/score/router"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetScoreRequest represents the incoming request to get a score.
type GetScoreRequest struct {
	DiscordID string `json:"discord_id"`
	RoundID   string `json:"round_id"`
}

// GetScoreResponse is the response for GetScore
type GetScoreResponse struct {
	Score scoredb.Score `json:"score"` // Use scoredb.Score
}

// ScoreHandlers defines the handlers for score-related events.
type ScoreHandlers struct {
	commandRouter scorerouter.CommandRouter
	queryService  scorequeries.QueryService
	pubsub        watermillutil.PubSuber
}

// NewScoreHandlers creates a new ScoreHandlers instance.
func NewScoreHandlers(commandRouter scorerouter.CommandRouter, queryService scorequeries.QueryService, pubsub watermillutil.PubSuber) *ScoreHandlers {
	return &ScoreHandlers{
		commandRouter: commandRouter,
		queryService:  queryService,
		pubsub:        pubsub,
	}
}

// Handle implements the MessageHandler interface.
func (h *ScoreHandlers) Handle(msg *message.Message) ([]*message.Message, error) {
	switch msg.Metadata.Get("topic") {
	case scorehandlers.TopicUpdateScores:
		return h.HandleUpdateScores(msg)
	case scorehandlers.TopicGetScore:
		return h.HandleGetScore(msg)
	default:
		return nil, fmt.Errorf("unknown message topic: %s", msg.Metadata.Get("topic"))
	}
}

// HandleUpdateScores updates scores for a round.
func (h *ScoreHandlers) HandleUpdateScores(msg *message.Message) ([]*message.Message, error) {
	var cmd scorecommands.UpdateScoresCommand
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.UpdateScores(context.Background(), cmd.RoundID, cmd.Scores); err != nil {
		return nil, fmt.Errorf("failed to update scores: %w", err)
	}

	return nil, nil
}

// HandleGetScore retrieves a score.
func (h *ScoreHandlers) HandleGetScore(msg *message.Message) ([]*message.Message, error) {
	var req GetScoreRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	score, err := h.queryService.GetScore(context.Background(), &scorequeries.GetScoreQuery{
		DiscordID: req.DiscordID,
		RoundID:   req.RoundID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get score: %w", err)
	}

	if score == nil {
		return nil, fmt.Errorf("score not found: %s, %s", req.DiscordID, req.RoundID)
	}

	response := GetScoreResponse{Score: *score} // Use *score to get the scoredb.Score value
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal score response: %w", err)
	}

	respMsg := message.NewMessage(watermill.NewUUID(), payload)
	respMsg.Metadata.Set("topic", scorehandlers.TopicGetScoreResponse)

	return []*message.Message{respMsg}, nil
}
