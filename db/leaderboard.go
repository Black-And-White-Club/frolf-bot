package db

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/models"
)

// LeaderboardStore is an interface for leaderboard-related database operations.
type LeaderboardDB interface {
	GetLeaderboard(ctx context.Context) (*models.Leaderboard, error)
	DeactivateCurrentLeaderboard(ctx context.Context) error
	InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error
	GetLeaderboardTagData(context.Context) (*models.Leaderboard, error)
	UpdateLeaderboardWithTransaction(ctx context.Context, leaderboardData map[int]string) error
}
