package roundrouter

import (
	"context"

	apimodels "github.com/Black-And-White-Club/tcr-bot/app/modules/round/models"
)

// RoundRouter defines the interface for the round router.
type CommandRouter interface {
	ScheduleRound(ctx context.Context, input apimodels.ScheduleRoundInput) error
	UpdateParticipant(ctx context.Context, input apimodels.UpdateParticipantResponseInput) error
	JoinRound(ctx context.Context, input apimodels.JoinRoundInput) error
	SubmitScore(ctx context.Context, input apimodels.SubmitScoreInput) error
	StartRound(ctx context.Context, roundID int64) error
	RecordRoundScores(ctx context.Context, roundID int64, scores ...any) error
	ProcessScoreSubmission(ctx context.Context, input apimodels.SubmitScoreInput) error
	FinalizeAndProcessScores(ctx context.Context, roundID int64) error
	EditRound(ctx context.Context, roundID int64, discordID string, input apimodels.EditRoundInput) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateRoundState(ctx context.Context, roundID int64, state apimodels.RoundState) error
}

// CommandBus is the interface for the command bus.
type CommandBus interface {
	Send(ctx context.Context, cmd roundcommands.Command) error
}
