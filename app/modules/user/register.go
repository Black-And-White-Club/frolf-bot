package userhandlers

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/user/db"
	userapimodels "github.com/Black-And-White-Club/tcr-bot/user/models"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// registerUserCommandHandlers registers all the command handlers for the user module.
func RegisterUserCommandHandlers(
	commandBus *cqrs.CommandBus,
	userDB userdb.UserDB,
	eventBus watermillcmd.MessageBus, // Use MessageBus
) {
	router, err := message.NewRouter(message.RouterConfig{}, nil)
	if err != nil {
		panic(err)
	}

	commandProcessor, err := cqrs.NewCommandProcessorWithConfig(router, cqrs.CommandProcessorConfig{})
	if err != nil {
		panic(err)
	}

	err = commandProcessor.AddHandlers(
		cqrs.NewCommandHandler(
			"CreateUser",
			func(ctx context.Context, cmd *userapimodels.CreateUserCommand) error {
				handler := &CreateUserHandler{
					userDB:   userDB,
					eventBus: eventBus,
				}
				msg := message.NewMessage(watermill.NewUUID(), nil)
				msg.Metadata.Set(cqrs.ProtobufMarshaler{}.NameFromMessage(msg), "userapimodels.CreateUserCommand")
				return handler.Handle(msg)
			},
		),
		cqrs.NewCommandHandler(
			"UpdateUser",
			func(ctx context.Context, cmd *userapimodels.UpdateUserCommand) error {
				handler := &UpdateUserHandler{
					userDB:   userDB,
					eventBus: eventBus,
				}
				msg := message.NewMessage(watermill.NewUUID(), nil)
				msg.Metadata.Set(cqrs.ProtobufMarshaler{}.NameFromMessage(msg), "userapimodels.UpdateUserCommand")
				return handler.Handle(msg)
			},
		),
	)
	if err != nil {
		panic(err)
	}
}
