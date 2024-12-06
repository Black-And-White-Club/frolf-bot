// internal/db/round.go
package db

import (
	"context"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/models"
)

// RoundStore is an interface for round-related database operations.
type RoundDB interface {
	GetRounds(ctx context.Context) ([]*models.Round, error)
	GetRound(ctx context.Context, roundID int64) (*models.Round, error)
	CreateRound(ctx context.Context, round *models.Round) (*models.Round, error)
	UpdateRound(ctx context.Context, round *models.Round) error
	DeleteRound(ctx context.Context, roundID int64, userID string) error
	UpdateParticipantResponse(ctx context.Context, roundID int64, discordID string, response models.Response) (*models.Round, error)
	UpdateRoundState(ctx context.Context, roundID int64, state models.RoundState) error
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*models.Round, error)
	FindParticipant(ctx context.Context, roundID int64, discordID string) (*models.Participant, error)
}
