// app/router.go
package app

import (
	"github.com/go-chi/chi/v5"

	"github.com/Black-And-White-Club/tcr-bot/round"
	roundcommands "github.com/Black-And-White-Club/tcr-bot/round/commands"
	converter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/round/queries"
	"github.com/Black-And-White-Club/tcr-bot/user"
	usercommands "github.com/Black-And-White-Club/tcr-bot/user/commands"
	userqueries "github.com/Black-And-White-Club/tcr-bot/user/queries"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
)

// Router sets up the main router for the application.
func (app *App) Router() chi.Router {
	r := chi.NewRouter()

	// Apply middleware
	r = app.applyMiddleware(r).(*chi.Mux)

	// Initialize RoundHandlers
	commandService := roundcommands.NewRoundCommandService(
		app.roundDB,
		&converter.DefaultRoundConverter{},
		app.messagePublisher,
		app.roundEventHandler,
	)

	queryService := roundqueries.NewRoundQueryService(
		app.roundDB,
		&converter.DefaultRoundConverter{},
	)

	roundConverter := &converter.DefaultRoundConverter{}

	roundHandler := round.NewRoundHandlers(app.roundDB, roundConverter, commandService, queryService)

	// Initialize the command bus
	commandBus, err := watermillcmd.NewCommandBus(app.Cfg) // Initialize commandBus
	if err != nil {
		// Handle the error appropriately
		panic(err)
	}

	// Initialize UserHandlers
	userService := usercommands.NewUserCommandService(
		app.userDB, // Use app.db.User
		app.NatsConnectionPool,
		app.messagePublisher,
		commandBus,
	)
	userQueryService := userqueries.NewUserQueryService(
		app.userDB,                   // Assuming app.userDB is of type db.UserDB
		watermillcmd.NewNatsPubSub(), // Assuming this function exists
	)
	userHandler := user.NewUserHandlers(userService, userQueryService)

	// Create RouteMounter and mount routes
	routeMounter := NewRouteMounter(userHandler, roundHandler)
	routeMounter.MountRoutes(r)

	return r
}
