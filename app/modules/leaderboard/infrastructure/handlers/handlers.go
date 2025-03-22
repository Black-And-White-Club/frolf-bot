package leaderboardhandlers

import (
	"log/slog"

	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	leaderboardService leaderboardservice.Service
	logger             *slog.Logger
}

// NewLeaderboardHandlers creates a new instance of LeaderboardHandlers.
func NewLeaderboardHandlers(leaderboardService leaderboardservice.Service, logger *slog.Logger) Handlers {
	return &LeaderboardHandlers{
		leaderboardService: leaderboardService,
		logger:             logger,
	}
}
