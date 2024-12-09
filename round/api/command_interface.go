package roundapi

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/round/common"

	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// RoundService defines the interface for round command operations.
type CommandService interface {
	ScheduleRound(ctx context.Context, input apimodels.ScheduleRoundInput) (*apimodels.Round, error)
	UpdateParticipant(ctx context.Context, input apimodels.UpdateParticipantResponseInput) (*apimodels.Round, error)
	JoinRound(ctx context.Context, input apimodels.JoinRoundInput) (*apimodels.Round, error)
	SubmitScore(ctx context.Context, input apimodels.SubmitScoreInput) error
	ProcessScoreSubmission(ctx context.Context, event common.ScoreSubmissionEvent, input apimodels.SubmitScoreInput) error
	FinalizeAndProcessScores(ctx context.Context, roundID int64) (*apimodels.Round, error)
	EditRound(ctx context.Context, roundID int64, discordID string, input apimodels.EditRoundInput) (*apimodels.Round, error)
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateRoundState(ctx context.Context, roundID int64, state common.RoundState) error
}
