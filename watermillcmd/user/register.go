package userhandlers

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/db"
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// registerUserCommandHandlers registers all the command handlers for the user module.
func registerUserCommandHandlers(commandBus *cqrs.CommandBus, userDB db.UserDB, eventBus watermillcmd.EventBus) {
	router, err := cqrs.NewRouter(cqrs.RouterConfig{
		GenerateHandlerFunctionName: cqrs.StructNameFromMessage,
	}, commandBus)
	if err != nil {
		// Handle the error appropriately, e.g., log it or panic
		panic(err)
	}

	err = router.AddHandler(
		"CreateUserCommandHandler",
		cqrs.NewCommandHandler(
			"CreateUser",
			func(ctx context.Context, cmd *userapimodels.CreateUserCommand) error {
				handler := &CreateUserHandler{
					userDB:   userDB,
					eventBus: eventBus,
				}
				msg := message.NewMessage(watermill.NewUUID(), nil)
				return handler.Handle(msg)
			},
		),
	)
	if err != nil {
		// Handle the error appropriately, e.g., log it or panic
		panic(err)
	}

	err = router.AddHandler(
		"UpdateUserCommandHandler",
		cqrs.NewCommandHandler(
			"UpdateUser",
			func(ctx context.Context, cmd *userapimodels.UpdateUserCommand) error {
				handler := &UpdateUserHandler{
					userDB:   userDB,
					eventBus: eventBus,
				}
				msg := message.NewMessage(watermill.NewUUID(), nil)
				return handler.Handle(msg)
			},
		),
	)
	if err != nil {
		// Handle the error appropriately, e.g., log it or panic
		panic(err)
	}

	// Start the router
	if err := router.Run(context.Background()); err != nil {
		// Handle the error appropriately
		panic(err)
	}
}

func init() {
	registerUserCommandHandlers(commandBus, userDB, eventBus)
}
