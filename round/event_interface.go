// round/event_interface.go

package round

import (
	"context"
	"time"

	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundEventHandler defines the interface for handling various round-related events.
type RoundEventHandler interface {
	HandleRoundCreate(ctx context.Context, event RoundCreateEvent) error
	HandlePlayerAddedToRound(ctx context.Context, msg *message.Message) error
	HandleTagNumberRetrieved(ctx context.Context, msg *message.Message) error
	HandleScoreSubmitted(ctx context.Context, event ScoreSubmissionEvent) error
	HandleRoundStarted(ctx context.Context, event RoundStartedEvent) error
	HandleRoundStartingOneHour(ctx context.Context, event RoundStartingOneHourEvent) error
	HandleRoundStartingThirtyMinutes(ctx context.Context, event RoundStartingThirtyMinutesEvent) error
	HandleRoundUpdated(ctx context.Context, event RoundUpdatedEvent) error
	HandleRoundDeleted(ctx context.Context, event RoundDeletedEvent) error
	HandleRoundFinalized(ctx context.Context, event RoundFinalizedEvent) error
}

// ScoreSubmissionEvent defines the interface for score submission events.
type ScoreSubmissionEvent interface {
	GetRoundID() int64
	GetDiscordID() string
	GetScore() int
}

// RoundCreateEvent defines the interface for round creation events.
type RoundCreateEvent interface {
	GetDiscordID() string
	GetDate() time.Time
	GetCourse() string
	GetInitialPlayers() []string
	GetTime() string
}

// RoundStartedEvent defines the interface for round started events.
type RoundStartedEvent interface {
	GetRoundID() int64
}

// RoundStartingOneHourEvent defines the interface for round starting one hour events.
type RoundStartingOneHourEvent interface {
	GetRoundID() int64
}

// RoundStartingThirtyMinutesEvent defines the interface for round starting thirty minutes events.
type RoundStartingThirtyMinutesEvent interface {
	GetRoundID() int64
}

// RoundUpdatedEvent defines the interface for round updated events.
type RoundUpdatedEvent interface {
	GetRoundID() int64
	GetTitle() string
	GetLocation() string
	GetDate() time.Time
	GetTime() string
}

// RoundDeletedEvent defines the interface for round deleted events.
type RoundDeletedEvent interface {
	GetRoundID() int64
}

// RoundFinalizedEvent defines the interface for round finalized events.
type RoundFinalizedEvent interface {
	GetRoundID() int64
	GetParticipants() []apimodels.ParticipantScore
}
