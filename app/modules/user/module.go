package user

import (
	"fmt"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserModule represents the user module.
type UserModule struct {
	CommandService usercommands.CommandService // Exported field
	QueryService   userqueries.QueryService    // Exported field
	pubsub         watermillutil.PubSuber
}

// NewUserModule creates a new UserModule with the provided dependencies.
func NewUserModule(
	CommandService usercommands.CommandService,
	QueryService userqueries.QueryService,
	pubsub watermillutil.PubSuber, // Use the PubSuber interface
) *UserModule {
	return &UserModule{
		CommandService: CommandService,
		QueryService:   QueryService,
		pubsub:         pubsub,
	}
}

// RegisterHandlers registers the user module's handlers.
func (m *UserModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	userHandlers := NewUserHandlers(m.CommandService, m.QueryService, m.pubsub)

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
