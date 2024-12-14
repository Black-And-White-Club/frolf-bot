package roundrouter

import (
	"context"
	"errors"
	"log"

	// Assuming handlers package is in the same directory

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	apimodels "github.com/Black-And-White-Club/tcr-bot/app/modules/round/models"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserCommandBus is the command bus for the user module.
type RoundCommandBus struct {
	publisher message.Publisher
	marshaler cqrs.CommandEventMarshaler
}

// NewUserCommandBus creates a new UserCommandBus.
func NewRoundCommandBus(publisher message.Publisher, marshaler cqrs.CommandEventMarshaler) *RoundCommandBus {
	return &RoundCommandBus{publisher: publisher, marshaler: marshaler}
}

func (r RoundCommandBus) Send(ctx context.Context, cmd commands.Command) error {
	return watermillutil.SendCommand(ctx, r.publisher, r.marshaler, cmd, cmd.CommandName())
}

// UserCommandRouter implements the CommandRouter interface.
type RoundCommandRouter struct {
	commandBus CommandBus
}

// NewUserCommandRouter creates a new UserCommandRouter.
func NewRoundCommandRouter(commandBus CommandBus) CommandRouter {
	return &RoundCommandRouter{commandBus: commandBus}
}

// CreateRound implements the CommandService interface.
func (r RoundCommandRouter) CreateRound(ctx context.Context, input rounddto.CreateRoundInput) error {
	if input.Title == "" {
		return errors.New("title is required")
	}

	createRoundCmd := roundcommands.CreateRoundRequest{
		Input: input,
	}
	return r.commandBus.Send(ctx, createRoundCmd)
}

// UpdateParticipant implements the CommandService interface.
func (r *RoundCommandRouter) UpdateParticipant(ctx context.Context, input apimodels.UpdateParticipantResponseInput) error {
	updateParticipantCmd := roundhandlers.UpdateParticipantRequest{
		UpdateParticipantResponseInput: input,
	}
	return r.commandBus.Send(ctx, updateParticipantCmd)
}

// JoinRound implements the CommandService interface.
func (r *RoundCommandRouter) JoinRound(ctx context.Context, input apimodels.JoinRoundInput) error {
	joinRoundCmd := roundhandlers.JoinRoundRequest{
		JoinRoundInput: input,
	}
	return r.commandBus.Send(ctx, joinRoundCmd)
}

// SubmitScore implements the CommandService interface.
func (r *RoundCommandRouter) SubmitScore(ctx context.Context, input apimodels.SubmitScoreInput) error {
	submitScoreCmd := roundhandlers.SubmitScoreRequest{
		SubmitScoreInput: input,
	}
	return r.commandBus.Send(ctx, submitScoreCmd)
}

// StartRound implements the CommandService interface.
func (r *RoundCommandRouter) StartRound(ctx context.Context, roundID int64) error {
	startRoundCmd := roundhandlers.StartRoundRequest{
		RoundID: roundID,
	}
	return r.commandBus.Send(ctx, startRoundCmd)
}

// RecordRoundScores implements the CommandService interface.
func (r *RoundCommandRouter) RecordRoundScores(ctx context.Context, roundID int64, scores ...any) error {
	// You'll need to define the RecordRoundScoresRequest struct in the handlers package
	recordRoundScoresCmd := roundhandlers.RecordRoundScoresRequest{
		RoundID: roundID,
		Scores:  scores, // Make sure the types match
	}
	return r.commandBus.Send(ctx, recordRoundScoresCmd)
}

// ProcessScoreSubmission implements the CommandService interface.
func (r *RoundCommandRouter) ProcessScoreSubmission(ctx context.Context, input apimodels.SubmitScoreInput) error {
	processScoreSubmissionCmd := roundhandlers.ProcessScoreSubmissionRequest{
		SubmitScoreInput: input,
	}
	return r.commandBus.Send(ctx, processScoreSubmissionCmd)
}

// FinalizeAndProcessScores implements the CommandService interface.
func (r *RoundCommandRouter) FinalizeAndProcessScores(ctx context.Context, roundID int64) error {
	finalizeAndProcessScoresCmd := roundhandlers.FinalizeAndProcessScoresRequest{
		RoundID: roundID,
	}
	return r.commandBus.Send(ctx, finalizeAndProcessScoresCmd)
}

// EditRound implements the RoundRouter interface.
func (r *RoundCommandRouter) EditRound(ctx context.Context, roundID int64, discordID string, input apimodels.EditRoundInput) error {
	editRoundCmd := roundcommands.EditRoundRequest{
		RoundID:   roundID,
		DiscordID: discordID,
		APIInput:  input,
	}
	return r.commandBus.Send(ctx, editRoundCmd)
}

// DeleteRound implements the CommandService interface.
func (r *RoundCommandRouter) DeleteRound(ctx context.Context, roundID int64) error {
	deleteRoundCmd := roundcommands.DeleteRoundRequest{ // Use handlers.DeleteRoundRequest
		RoundID: roundID,
	}
	log.Printf("Sending DeleteRoundCommand: %+v\n", deleteRoundCmd)
	return r.commandBus.Send(ctx, deleteRoundCmd)
}

func (r *RoundCommandRouter) UpdateRoundState(ctx context.Context, roundID int64, state rounddb.RoundState) error {
	cmd := roundcommands.UpdateRoundStateRequest{
		RoundID: roundID,
		State:   state,
	}
	return r.commandBus.Send(ctx, cmd)
}
