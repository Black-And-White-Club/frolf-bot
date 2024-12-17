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

// NewUser Module creates a new UserModule with the provided dependencies.
func NewUserModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*UserModule, error) {
	log.Println("Initializing UserModule...")

	marshaler := watermillutil.Marshaler
	userCommandBus := userrouter.NewUserCommandBus(pubsub, marshaler)
	userCommandRouter := userrouter.NewUserCommandRouter(userCommandBus)

	getUserByDiscordIDHandler := userqueries.NewGetUserByDiscordIDHandler(dbService.UserDB)
	getUserRoleHandler := userqueries.NewGetUserRoleHandler(dbService.UserDB)

	userQueryService := userqueries.NewUserQueryService(getUserByDiscordIDHandler, getUserRoleHandler)

	messageHandler := NewUserHandlers(userCommandRouter, userQueryService, pubsub)

	log.Println("User Module initialized successfully.")

	return &UserModule{
		CommandRouter:  userCommandRouter,
		QueryService:   userQueryService,
		PubSub:         pubsub,
		messageHandler: messageHandler,
	}, nil
}

// RegisterHandlers registers the user module's handlers.
func (m *UserModule) RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error {
	log.Println("Registering user module handlers...")

	// Log the state of the messageHandler
	if m.messageHandler == nil {
		return fmt.Errorf("messageHandler is nil")
	}
	log.Println("MessageHandler is initialized.")

	// Register user_get_role_handler
	log.Println("Registering user_get_role_handler...")
	log.Printf("HandleGetUser RoleWrapper function: %v", m.messageHandler.HandleGetUserRoleWrapper) // Log the handler function
	log.Printf("TopicGetUser Role: %s", userhandlers.TopicGetUserRole)                              // Log the topic constant

	if err := router.AddNoPublisherHandler(
		"user_get_role_handler",
		string(userhandlers.TopicGetUserRole),
		pubsub,
		m.messageHandler.HandleGetUserRoleWrapper,
	); err != nil {
		log.Printf("Error registering user_get_role_handler: %v", err) // Log the error
		return fmt.Errorf("failed to register user_get_role_handler: %v", err)
	}
	log.Println("user_get_role_handler registered successfully.")

	// Register user_get_user_handler
	log.Println("Registering user_get_user_handler...")
	log.Printf("HandleGetUser Wrapper function: %v", m.messageHandler.HandleGetUserWrapper) // Log the handler function
	log.Printf("TopicGet:User  %s", userhandlers.TopicGetUser)                              // Log the topic constant

	if err := router.AddNoPublisherHandler(
		"user_get_user_handler",
		string(userhandlers.TopicGetUser),
		pubsub,
		m.messageHandler.HandleGetUserWrapper,
	); err != nil {
		log.Printf("Error registering user_get_user_handler: %v", err) // Log the error
		return fmt.Errorf("failed to register user_get_user_handler: %v", err)
	}
	log.Println("user_get_user_handler registered successfully.")

	// Register user_create_user_handler
	log.Println("Registering user_create_user_handler...")
	log.Printf("HandleCreateUser  function: %v", m.messageHandler.HandleCreateUser) // Log the handler function
	log.Printf("TopicCreate:User  %s", userhandlers.TopicCreateUser)                // Log the topic constant

	if err := router.AddHandler(
		"user_create_user_handler",
		string(userhandlers.TopicCreateUser),
		pubsub,
		userhandlers.TopicCreateUserResponse,
		pubsub,
		m.messageHandler.HandleCreateUser,
	); err != nil {
		log.Printf("Error registering user_create_user_handler: %v", err) // Log the error
		return fmt.Errorf("failed to register user_create_user_handler: %v", err)
	}
	log.Println("user_create_user_handler registered successfully.")

	// Register user_update_user_handler
	log.Println("Registering user_update_user_handler...")
	log.Printf("HandleUpdateUser  function: %v", m.messageHandler.HandleUpdateUser) // Log the handler function
	log.Printf("TopicUpdate:User  %s", userhandlers.TopicUpdateUser)                // Log the topic constant

	if err := router.AddHandler(
		"user_update_user_handler",
		string(userhandlers.TopicUpdateUser),
		pubsub,
		userhandlers.TopicUpdateUserResponse,
		pubsub,
		m.messageHandler.HandleUpdateUser,
	); err != nil {
		log.Printf("Error registering user_update_user_handler: %v", err) // Log the error
		return fmt.Errorf("failed to register user_update_user_handler: %v", err)
	}
	log.Println("user_update_user_handler registered successfully.")

	return nil
}
