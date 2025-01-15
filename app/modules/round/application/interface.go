package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
)

// RoundService defines the interface for the round service.
type Service interface {
	CreateRound(ctx context.Context, input roundevents.RoundCreateRequestPayload) error
	UpdateRound(ctx context.Context, event *roundevents.RoundUpdatedPayload) error
	DeleteRound(ctx context.Context, event *roundevents.RoundDeletedPayload) error
	JoinRound(ctx context.Context, event *roundevents.ParticipantResponsePayload) error
	UpdateScore(ctx context.Context, event *roundevents.ScoreUpdatedPayload) error
	UpdateScoreAdmin(ctx context.Context, event *roundevents.ScoreUpdatedPayload) error
	FinalizeRound(ctx context.Context, event *roundevents.RoundFinalizedPayload) error
	StartRound(ctx context.Context, event *roundevents.RoundStartedPayload) error
}
