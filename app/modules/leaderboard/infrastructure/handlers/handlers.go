package leaderboardhandlers

import (
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	leaderboardService leaderboardservice.Service
	logger             observability.Logger
	tracer             observability.Tracer
	helpers            utils.Helpers
}

// NewLeaderboardHandlers creates a new instance of LeaderboardHandlers.
func NewLeaderboardHandlers(leaderboardService leaderboardservice.Service, logger observability.Logger, tracer observability.Tracer, helpers utils.Helpers) Handlers {
	return &LeaderboardHandlers{
		leaderboardService: leaderboardService,
		logger:             logger,
		tracer:             tracer,
		helpers:            helpers,
	}
}
