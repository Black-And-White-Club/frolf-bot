package user

import (
	"fmt"
	"log"

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
func (m *UserModule) GetHandlers() map[string]struct {
	topic         string
	handler       message.HandlerFunc
	responseTopic string
} {
	return map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}{
		"user_get_role_handler": {
			topic:         userhandlers.TopicGetUserRole,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicGetUserRoleResponse,
		},
		"user_get_user_handler": {
			topic:         userhandlers.TopicGetUser,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicGetUserResponse,
		},
		"user_create_user_handler": {
			topic:         userhandlers.TopicCreateUser,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicCreateUserResponse,
		},
		"user_update_user_handler": {
			topic:         userhandlers.TopicUpdateUser,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicUpdateUserResponse,
		},
	}
}

// RegisterHandlers registers the user module's handlers.
func (m *UserModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	handlers := map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}{
		"user_get_role_handler": {
			topic:         userhandlers.TopicGetUserRole,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicGetUserRoleResponse,
		},
		"user_get_user_handler": {
			topic:         userhandlers.TopicGetUser,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicGetUserResponse,
		},
		"user_create_user_handler": {
			topic:         userhandlers.TopicCreateUser,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicCreateUserResponse,
		},
		"user_update_user_handler": {
			topic:         userhandlers.TopicUpdateUser,
			handler:       m.messageHandler.Handle,
			responseTopic: userhandlers.TopicUpdateUserResponse,
		},
	}

	for handlerName, h := range handlers {
		log.Printf("Trying to register handler: %s with topic %s", handlerName, h.topic)
		if err := router.AddHandler(
			handlerName,
			h.topic,
			pubsub,
			h.responseTopic,
			pubsub,
			h.handler,
		); err != nil {
			log.Printf("Failed to register handler %s: %v", handlerName, err)
			return fmt.Errorf("failed to register %s handler: %v", handlerName, err)
		}
		log.Printf("Registered handler: %s", handlerName)
	}

	return nil
}
