package user

import (
	"fmt"

	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	userrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/user/router"
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
	messageHandler *UserHandlers // Pointer to UserHandlers
}

// NewUserModule creates a new UserModule with the provided dependencies.
func NewUserModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*UserModule, error) {
	marshaler := userrouter.Marshaler
	userCommandBus := userrouter.NewUserCommandBus(pubsub, marshaler)
	userCommandRouter := userrouter.NewUserCommandRouter(userCommandBus)

	getUserByDiscordIDHandler := userqueries.NewGetUserByDiscordIDHandler(dbService.User)
	getUserRoleHandler := userqueries.NewGetUserRoleHandler(dbService.User)
	userQueryService := userqueries.NewUserQueryService(getUserByDiscordIDHandler, getUserRoleHandler)

	messageHandler := NewUserHandlers(userCommandRouter, userQueryService, pubsub)

	return &UserModule{
		CommandRouter:  userCommandRouter,
		QueryService:   userQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler, // Store the pointer
	}, nil
}

// RegisterHandlers registers the user module's handlers.
func (m *UserModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := []struct {
		handlerName string
		topic       string
		handler     message.HandlerFunc
	}{
		{
			handlerName: "user_create_handler",
			topic:       userhandlers.TopicCreateUser,
			handler:     m.messageHandler.Handle,
		},
		{
			handlerName: "user_get_handler",
			topic:       userhandlers.TopicGetUser,
			handler:     m.messageHandler.Handle,
		},
		{
			handlerName: "user_update_handler",
			topic:       userhandlers.TopicUpdateUser,
			handler:     m.messageHandler.Handle,
		},
		{
			handlerName: "user_get_role_handler",
			topic:       userhandlers.TopicGetUserRoleRequest, // Correct topic
			handler:     m.messageHandler.Handle,
		},
	}

	for _, h := range handlers {
		if err := router.AddHandler(
			h.handlerName,
			h.topic,
			pubsub,
			h.topic+"_response", // Assuming you have response topics
			pubsub,
			h.handler,
		); err != nil {
			return fmt.Errorf("failed to register %s handler: %v", h.handlerName, err)
		}
	}

	return nil
}
