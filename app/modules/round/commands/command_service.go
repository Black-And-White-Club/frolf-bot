package roundcommands

import (
	"context"
	"errors"

	// Assuming handlers package is in the same directory
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/common"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	apimodels "github.com/Black-And-White-Club/tcr-bot/app/modules/round/models"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// RoundCommandService handles command-related logic for rounds.
type RoundCommandService struct {
	roundDB    rounddb.RoundDB
	commandBus cqrs.CommandBus
}

// NewRoundCommandService creates a new RoundCommandService.
func NewRoundCommandService(roundDB rounddb.RoundDB, commandBus cqrs.CommandBus) CommandService {
	return &RoundCommandService{
		roundDB:    roundDB,
		commandBus: commandBus,
	}
}

func (s *RoundCommandService) CommandBus() cqrs.CommandBus {
	return s.commandBus
}

// ScheduleRound implements the CommandService interface.
func (s *RoundCommandService) ScheduleRound(ctx context.Context, input apimodels.ScheduleRoundInput) error {
	if input.Title == "" {
		return errors.New("title is required")
	}

	scheduleRoundCmd := roundhandlers.ScheduleRoundRequest{
		ScheduleRoundInput: input,
	}
	return s.commandBus.Send(ctx, scheduleRoundCmd)
}

// UpdateParticipant implements the CommandService interface.
func (s *RoundCommandService) UpdateParticipant(ctx context.Context, input apimodels.UpdateParticipantResponseInput) error {
	updateParticipantCmd := roundhandlers.UpdateParticipantRequest{
		UpdateParticipantResponseInput: input,
	}
	return s.commandBus.Send(ctx, updateParticipantCmd)
}

// JoinRound implements the CommandService interface.
func (s *RoundCommandService) JoinRound(ctx context.Context, input apimodels.JoinRoundInput) error {
	joinRoundCmd := roundhandlers.JoinRoundRequest{
		JoinRoundInput: input,
	}
	return s.commandBus.Send(ctx, joinRoundCmd)
}

// SubmitScore implements the CommandService interface.
func (s *RoundCommandService) SubmitScore(ctx context.Context, input apimodels.SubmitScoreInput) error {
	submitScoreCmd := roundhandlers.SubmitScoreRequest{
		SubmitScoreInput: input,
	}
	return s.commandBus.Send(ctx, submitScoreCmd)
}

// StartRound implements the CommandService interface.
func (s *RoundCommandService) StartRound(ctx context.Context, roundID int64) error {
	startRoundCmd := roundhandlers.StartRoundRequest{
		RoundID: roundID,
	}
	return s.commandBus.Send(ctx, startRoundCmd)
}

// RecordRoundScores implements the CommandService interface.
func (s *RoundCommandService) RecordRoundScores(ctx context.Context, roundID int64, scores ...any) error {
	// You'll need to define the RecordRoundScoresRequest struct in the handlers package
	recordRoundScoresCmd := roundhandlers.RecordRoundScoresRequest{
		RoundID: roundID,
		Scores:  scores, // Make sure the types match
	}
	return s.commandBus.Send(ctx, recordRoundScoresCmd)
}

// ProcessScoreSubmission implements the CommandService interface.
func (s *RoundCommandService) ProcessScoreSubmission(ctx context.Context, input apimodels.SubmitScoreInput) error {
	processScoreSubmissionCmd := roundhandlers.ProcessScoreSubmissionRequest{
		SubmitScoreInput: input,
	}
	return s.commandBus.Send(ctx, processScoreSubmissionCmd)
}

// FinalizeAndProcessScores implements the CommandService interface.
func (s *RoundCommandService) FinalizeAndProcessScores(ctx context.Context, roundID int64) error {
	finalizeAndProcessScoresCmd := roundhandlers.FinalizeAndProcessScoresRequest{
		RoundID: roundID,
	}
	return s.commandBus.Send(ctx, finalizeAndProcessScoresCmd)
}

// EditRound implements the CommandService interface.
func (s *RoundCommandService) EditRound(ctx context.Context, roundID int64, discordID string, input apimodels.EditRoundInput) error {
	editRoundCmd := roundhandlers.EditRoundRequest{
		RoundID:        roundID,
		DiscordID:      discordID,
		EditRoundInput: input,
	}
	return s.commandBus.Send(ctx, editRoundCmd)
}

// DeleteRound implements the CommandService interface.
func (s *RoundCommandService) DeleteRound(ctx context.Context, roundID int64) error {
	deleteRoundCmd := roundhandlers.DeleteRoundRequest{
		RoundID: roundID,
	}
	return s.commandBus.Send(ctx, deleteRoundCmd)
}

// UpdateRoundState implements the CommandService interface.
func (s *RoundCommandService) UpdateRoundState(ctx context.Context, roundID int64, state common.RoundState) error {
	updateRoundStateCmd := roundhandlers.UpdateRoundStateRequest{
		RoundID: roundID,
		State:   state,
	}
	return s.commandBus.Send(ctx, updateRoundStateCmd)
}
