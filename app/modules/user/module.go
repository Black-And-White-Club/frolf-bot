package user

import (
	"fmt"
	"log"

	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	userrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/user/router"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserModule represents the user module.
type UserModule struct {
	CommandRouter  userrouter.CommandRouter
	QueryService   userqueries.QueryService
	PubSub         watermillutil.PubSuber
	messageHandler *UserHandlers
}

// NewUserModule creates a new UserModule with the provided dependencies.
func NewUserModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*UserModule, error) {
	marshaler := watermillutil.Marshaler
	userCommandBus := userrouter.NewUserCommandBus(pubsub, marshaler)
	userCommandRouter := userrouter.NewUserCommandRouter(userCommandBus)

	getUserByDiscordIDHandler := userqueries.NewGetUserByDiscordIDHandler(dbService.UserDB)
	getUserRoleHandler := userqueries.NewGetUserRoleHandler(dbService.UserDB)

	userQueryService := userqueries.NewUserQueryService(getUserByDiscordIDHandler, getUserRoleHandler)

	messageHandler := NewUserHandlers(userCommandRouter, userQueryService, pubsub)

	return &UserModule{
		CommandRouter:  userCommandRouter,
		QueryService:   userQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// GetHandlers returns the handlers registered for the UserModule
func (m *UserModule) GetHandlers() map[string]types.Handler {
	return map[string]types.Handler{
		"user_get_role_handler": {
			Topic:         userhandlers.TopicGetUserRole,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: userhandlers.TopicGetUserRoleResponse,
		},
		"user_get_user_handler": {
			Topic:         userhandlers.TopicGetUser,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: userhandlers.TopicGetUserResponse,
		},
		"user_create_user_handler": {
			Topic:         userhandlers.TopicCreateUser,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: userhandlers.TopicCreateUserResponse,
		},
		"user_update_user_handler": {
			Topic:         userhandlers.TopicUpdateUser,
			Handler:       m.messageHandler.Handle,
			ResponseTopic: userhandlers.TopicUpdateUserResponse,
		},
	}
}

// RegisterHandlers registers the user module's handlers.
func (m *UserModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := m.GetHandlers()

	for handlerName, h := range handlers {
		log.Printf("Registering handler: %s with topic %s", handlerName, string(h.Topic))
		log.Printf("pubsub: %v, m.PubSub: %v", pubsub, m.PubSub) // Log pubsub values

		if err := router.AddHandler(
			handlerName,
			string(h.Topic),
			pubsub, // Use the pubsub argument here
			h.ResponseTopic,
			pubsub, // Use the pubsub argument here
			h.Handler,
		); err != nil {
			log.Printf("Failed to register handler %s: %v", handlerName, err)
			return fmt.Errorf("failed to register %s handler: %v", handlerName, err)
		}
	}

	return nil
}
