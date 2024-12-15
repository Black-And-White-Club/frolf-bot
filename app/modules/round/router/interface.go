package roundrouter

import (
	"context"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// RoundRouter defines the interface for the round router.
type CommandRouter interface {
	CreateRound(ctx context.Context, input rounddto.CreateRoundInput) error // Add this method
	UpdateParticipant(ctx context.Context, input rounddto.UpdateParticipantResponseInput) error
	JoinRound(ctx context.Context, input rounddto.JoinRoundInput) error
	SubmitScore(ctx context.Context, input rounddto.SubmitScoreInput) error
	StartRound(ctx context.Context, roundID int64) error
	RecordRoundScores(ctx context.Context, roundID int64) error
	ProcessScoreSubmission(ctx context.Context, input rounddto.SubmitScoreInput) error
	FinalizeAndProcessScores(ctx context.Context, roundID int64) error
	EditRound(ctx context.Context, roundID int64, discordID string, input rounddto.EditRoundInput) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateRoundState(ctx context.Context, roundID int64, state rounddb.RoundState) error
}

// CommandBus is the interface for the command bus.
type CommandBus interface {
	Send(ctx context.Context, cmd commands.Command) error
}
