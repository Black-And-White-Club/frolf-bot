package user

import (
	"fmt"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserModule represents the user module.
type UserModule struct {
	CommandService usercommands.CommandService
	QueryService   userqueries.QueryService
	PubSub         watermillutil.PubSuber
}

// NewUserModule creates a new UserModule with the provided dependencies.
func NewUserModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*UserModule, error) {
	// Initialize UserHandlers
	userCommandService := usercommands.NewUserCommandService(
		dbService.User,
		pubsub,
		*commandBus,
	)
	userQueryService := userqueries.NewUserQueryService(
		dbService.User,
		pubsub.(*watermillutil.PubSub),
	)

	// Register the user command handlers
	if err := userhandlers.RegisterUserCommandHandlers(commandBus, dbService.User, pubsub.(*watermillutil.PubSub)); err != nil {
		return nil, fmt.Errorf("failed to register user command handlers: %w", err)
	}

	return &UserModule{
		CommandService: userCommandService,
		QueryService:   userQueryService,
		PubSub:         pubsub,
	}, nil
}

// RegisterHandlers registers the user module's handlers.
func (m *UserModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	userHandlers := NewUserHandlers(m.CommandService, m.QueryService, pubsub)

	handlers := []struct {
		handlerName string
		topic       string
		handler     message.HandlerFunc
	}{
		{
			handlerName: "user_create_handler",
			topic:       "create-user",
			handler:     userHandlers.HandleCreateUser,
		},
		{
			handlerName: "user_get_handler",
			topic:       "get-user",
			handler:     userHandlers.HandleGetUser,
		},
		{
			handlerName: "user_update_handler",
			topic:       "update-user",
			handler:     userHandlers.HandleUpdateUser,
		},
		{
			handlerName: "user_get_role_handler",
			topic:       "get-user-role",
			handler:     userHandlers.HandleGetUserRole,
		},
		// ... add more user handlers ...
	}

	for _, h := range handlers {
		if err := router.AddHandler(
			h.handlerName,
			h.topic,
			pubsub,
			h.topic+"_response",
			pubsub,
			h.handler,
		); err != nil {
			return fmt.Errorf("failed to register %s handler: %v", h.handlerName, err)
		}
	}

	return nil
}
