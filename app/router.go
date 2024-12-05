package app

import (
	"github.com/Black-And-White-Club/tcr-bot/app/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *App) Router() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Mount("/users", handlers.UserRoutes())
	r.Mount("/rounds", handlers.RoundRoutes())
	r.Mount("/scores", handlers.ScoreRoutes())
	r.Mount("/leaderboard", handlers.LeaderboardRoutes())

	return r
}
