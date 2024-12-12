package roundcommands

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/common"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	apimodels "github.com/Black-And-White-Club/tcr-bot/app/modules/round/models"
)

// CommandService defines the interface for round commands.
type CommandService interface {
	ScheduleRound(ctx context.Context, input apimodels.ScheduleRoundInput) error
	UpdateParticipant(ctx context.Context, input apimodels.UpdateParticipantResponseInput) error
	JoinRound(ctx context.Context, input apimodels.JoinRoundInput) error
	SubmitScore(ctx context.Context, input apimodels.SubmitScoreInput) error
	StartRound(ctx context.Context, roundID int64) error
	RecordRoundScores(ctx context.Context, roundID int64, scores ...any) error // New command
	ProcessScoreSubmission(ctx context.Context, input apimodels.SubmitScoreInput) error
	FinalizeAndProcessScores(ctx context.Context, roundID int64) error
	EditRound(ctx context.Context, roundID int64, discordID string, input apimodels.EditRoundInput) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateRoundState(ctx context.Context, roundID int64, state common.RoundState) error
	CommandBus() cqrs.CommandBus
}
