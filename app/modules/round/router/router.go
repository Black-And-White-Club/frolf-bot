package roundrouter

import (
	"context"
	"errors"
	"log"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundCommandBus is the command bus for the round module.
type RoundCommandBus struct {
	publisher message.Publisher
	marshaler cqrs.CommandEventMarshaler
}

// NewRoundCommandBus creates a new RoundCommandBus.
func NewRoundCommandBus(publisher message.Publisher, marshaler cqrs.CommandEventMarshaler) *RoundCommandBus {
	return &RoundCommandBus{publisher: publisher, marshaler: marshaler}
}

func (r RoundCommandBus) Send(ctx context.Context, cmd commands.Command) error {
	return watermillutil.SendCommand(ctx, r.publisher, r.marshaler, cmd, cmd.CommandName())
}

// RoundCommandRouter implements the CommandRouter interface.
type RoundCommandRouter struct {
	commandBus CommandBus
}

// NewRoundCommandRouter creates a new RoundCommandRouter.
func NewRoundCommandRouter(commandBus CommandBus) CommandRouter {
	return &RoundCommandRouter{commandBus: commandBus}
}

// CreateRound implements the CommandService interface.
func (r *RoundCommandRouter) CreateRound(ctx context.Context, input rounddto.CreateRoundInput) error {
	if input.Title == "" {
		return errors.New("title is required")
	}

	createRoundCmd := roundcommands.CreateRoundRequest{
		Input: input,
	}
	return r.commandBus.Send(ctx, createRoundCmd)
}

// UpdateParticipant implements the CommandService interface.
func (r *RoundCommandRouter) UpdateParticipant(ctx context.Context, input rounddto.UpdateParticipantResponseInput) error {
	updateParticipantCmd := roundcommands.UpdateParticipantRequest{
		Input: input,
	}
	return r.commandBus.Send(ctx, updateParticipantCmd)
}

// JoinRound implements the CommandService interface.
func (r *RoundCommandRouter) JoinRound(ctx context.Context, input rounddto.JoinRoundInput) error {
	joinRoundCmd := roundcommands.JoinRoundRequest{
		Input: input,
	}
	return r.commandBus.Send(ctx, joinRoundCmd)
}

// SubmitScore implements the CommandService interface.
func (r *RoundCommandRouter) SubmitScore(ctx context.Context, input rounddto.SubmitScoreInput) error {
	submitScoreCmd := roundcommands.SubmitScoreRequest{
		Input: input,
	}
	return r.commandBus.Send(ctx, submitScoreCmd)
}

// StartRound implements the CommandService interface.
func (r *RoundCommandRouter) StartRound(ctx context.Context, roundID int64) error {
	startRoundCmd := roundcommands.StartRoundRequest{
		RoundID: roundID,
	}
	return r.commandBus.Send(ctx, startRoundCmd)
}

// RecordRoundScores implements the CommandService interface.
func (r *RoundCommandRouter) RecordRoundScores(ctx context.Context, roundID int64) error {
	recordRoundScoresCmd := roundcommands.RecordScoresRequest{
		RoundID: roundID,
	}
	return r.commandBus.Send(ctx, recordRoundScoresCmd)
}

// ProcessScoreSubmission implements the CommandService interface.
func (r *RoundCommandRouter) ProcessScoreSubmission(ctx context.Context, input rounddto.SubmitScoreInput) error {
	processScoreSubmissionCmd := roundcommands.ProcessScoreSubmissionRequest{
		Input: input,
	}
	return r.commandBus.Send(ctx, processScoreSubmissionCmd)
}

// FinalizeAndProcessScores implements the CommandService interface.
func (r *RoundCommandRouter) FinalizeAndProcessScores(ctx context.Context, roundID int64) error {
	finalizeAndProcessScoresCmd := roundcommands.FinalizeRoundRequest{
		RoundID: roundID,
	}
	return r.commandBus.Send(ctx, finalizeAndProcessScoresCmd)
}

// EditRound implements the RoundRouter interface.
func (r *RoundCommandRouter) EditRound(ctx context.Context, roundID int64, discordID string, input rounddto.EditRoundInput) error {
	editRoundCmd := roundcommands.EditRoundRequest{
		RoundID:   roundID,
		DiscordID: discordID,
		APIInput:  input,
	}
	return r.commandBus.Send(ctx, editRoundCmd)
}

// DeleteRound implements the CommandService interface.
func (r *RoundCommandRouter) DeleteRound(ctx context.Context, roundID int64) error {
	deleteRoundCmd := roundcommands.DeleteRoundRequest{
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
