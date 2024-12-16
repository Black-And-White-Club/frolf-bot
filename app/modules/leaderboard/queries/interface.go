// leaderboard/queries/interface.go
package leaderboardqueries

import (
	"context"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
)

// LeaderboardQueryService defines methods for querying leaderboard data.
type QueryService interface {
	GetUserTag(ctx context.Context, query GetUserTagQuery) (int, error)
	IsTagTaken(ctx context.Context, tagNumber int) (bool, error)
	GetParticipantTag(ctx context.Context, participantID string) (int, error)
	GetTagHolder(ctx context.Context, tagNumber int) (string, error)
	GetLeaderboard(ctx context.Context) (*leaderboarddb.Leaderboard, error) // Add this method

}
