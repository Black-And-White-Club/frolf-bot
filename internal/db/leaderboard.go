package db

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
)

// LeaderboardStore is an interface for leaderboard-related database operations.
type LeaderboardDB interface {
	GetLeaderboard(ctx context.Context) (*models.Leaderboard, error)
	GetUserTag(ctx context.Context, discordID string) (*models.Leaderboard, error)
	IsTagAvailable(ctx context.Context, tagNumber int) (bool, error)
	DeactivateCurrentLeaderboard(ctx context.Context) error
	InsertLeaderboard(ctx context.Context, leaderboard *models.Leaderboard) error
}
