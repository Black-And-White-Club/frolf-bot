// app/route_mounter.go
package app

import (
	"github.com/go-chi/chi/v5"

	"github.com/Black-And-White-Club/tcr-bot/round"
)

// RouteMounter mounts the API routes.
type RouteMounter struct {
	// userHandler        user.UserHandler
	roundHandler round.RoundHandler
	// scoreHandler       score.ScoreHandler
	// leaderboardHandler leaderboard.LeaderboardHandler
}

// NewRouteMounter creates a new RouteMounter instance.
func NewRouteMounter(
	// userHandler user.UserHandler,
	roundHandler round.RoundHandler,
	// scoreHandler score.ScoreHandler,
	// leaderboardHandler leaderboard.LeaderboardHandler,
) *RouteMounter {
	return &RouteMounter{
		// userHandler:        userHandler,
		roundHandler: roundHandler,
		// scoreHandler:       scoreHandler,
		// leaderboardHandler: leaderboardHandler,
	}
}

// MountRoutes mounts the API routes.
func (rm *RouteMounter) MountRoutes(r chi.Router) {
	// r.Mount("/users", user.UserRoutes(rm.userHandler))
	r.Mount("/rounds", round.RoundRoutes(rm.roundHandler))
	// r.Mount("/scores", score.ScoreRoutes(rm.scoreHandler))
	// r.Mount("/leaderboard", leaderboard.LeaderboardRoutes(rm.leaderboardHandler))
}
