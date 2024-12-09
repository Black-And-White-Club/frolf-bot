// app/router.go
package app

import (
	"github.com/go-chi/chi/v5"

	"github.com/Black-And-White-Club/tcr-bot/round"
	roundapi "github.com/Black-And-White-Club/tcr-bot/round/api"
	roundcommands "github.com/Black-And-White-Club/tcr-bot/round/commands"
	converter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/round/queries"
)

// Router sets up the main router for the application.
func (app *App) Router() chi.Router {
	var r chi.Router
	r = chi.NewRouter()

	// Apply middleware
	r = app.applyMiddleware(r)

	// Initialize RoundHandlers
	var commandService roundapi.CommandService
	commandService = roundcommands.NewRoundCommandService(
		app.roundDB,
		&converter.DefaultRoundConverter{},
		app.messagePublisher,
		app.roundEventHandler,
	)

	var queryService roundqueries.QueryService
	queryService = roundqueries.NewRoundQueryService(
		app.roundDB,
		&converter.DefaultRoundConverter{},
	)

	var roundConverter converter.RoundConverter
	roundConverter = &converter.DefaultRoundConverter{}

	roundHandler := round.NewRoundHandlers(app.roundDB, roundConverter, commandService, queryService)

	// Create RouteMounter and mount routes
	routeMounter := NewRouteMounter(roundHandler)
	routeMounter.MountRoutes(r)

	return r
}
