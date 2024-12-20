package roundservice

import (
	"context"

	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
)

// RoundService defines the interface for the round service.
type Service interface {
	CreateRound(ctx context.Context, event *roundevents.RoundCreatedEvent, dto rounddto.CreateRoundParams) error
	UpdateRound(ctx context.Context, event *roundevents.RoundUpdatedEvent) error
	DeleteRound(ctx context.Context, event *roundevents.RoundDeletedEvent) error
	JoinRound(ctx context.Context, event *roundevents.ParticipantResponseEvent) error
	UpdateScore(ctx context.Context, event *roundevents.ScoreUpdatedEvent) error
	UpdateScoreAdmin(ctx context.Context, event *roundevents.ScoreUpdatedEvent) error
	FinalizeRound(ctx context.Context, event *roundevents.RoundFinalizedEvent) error
	StartRound(ctx context.Context, event *roundevents.RoundStartedEvent) error
}
