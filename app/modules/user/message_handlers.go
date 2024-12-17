package user

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	userrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/user/router"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetUserRequest represents the incoming request to get a user.
type GetUserRequest struct {
	DiscordID string `json:"discord_id"`
}

// GetUserResponse is the response for GetUser
type GetUserResponse struct {
	User userdb.User `json:"user"`
}

// GetUserRoleRequest represents the incoming request to get a user's role.
type GetUserRoleRequest struct {
	DiscordID string `json:"discord_id"`
}

// UserRoleResponseEvent is the response for GetUserRole queries.
type UserRoleResponseEvent struct {
	Role userdb.UserRole `json:"role"`
}

// UserHandlers defines the handlers for user-related events.
type UserHandlers struct {
	commandRouter userrouter.CommandRouter // Handles command messages
	queryService  userqueries.QueryService // Handles query logic
	pubsub        watermillutil.PubSuber   // Used for publishing responses
}

// NewUserHandlers creates a new instance of UserHandlers.
func NewUserHandlers(commandRouter userrouter.CommandRouter, queryService userqueries.QueryService, pubsub watermillutil.PubSuber) *UserHandlers {
	return &UserHandlers{
		commandRouter: commandRouter,
		queryService:  queryService,
		pubsub:        pubsub,
	}
}

// Handle implements the MessageHandler interface.
func (h *UserHandlers) Handle(msg *message.Message) ([]*message.Message, error) {
	log.Printf("UserHandlers.Handle called with topic: %s", msg.Metadata.Get("topic")) // Log entry with topic
	log.Printf("Received message: %v", msg)                                            // Log the incoming message
	switch msg.Metadata.Get("topic") {
	case userhandlers.TopicCreateUser:
		log.Println("Calling HandleCreateUser")
		return h.HandleCreateUser(msg)
	case userhandlers.TopicGetUser:
		log.Println("Calling HandleGetUser")
		return h.HandleGetUser(msg)
	case userhandlers.TopicUpdateUser:
		log.Println("Calling HandleUpdateUser")
		return h.HandleUpdateUser(msg)
	case userhandlers.TopicGetUserRole:
		log.Println("Calling HandleGetUserRole")
		return h.HandleGetUserRole(msg)
	default:
		return nil, fmt.Errorf("unknown message topic: %s", msg.Metadata.Get("topic"))
	}
}

// HandleCreateUser creates a new user (Command).
func (h *UserHandlers) HandleCreateUser(msg *message.Message) ([]*message.Message, error) {
	var req usercommands.CreateUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	tagNumberStr := msg.Metadata.Get("tagNumber")
	if tagNumberStr == "" {
		return nil, fmt.Errorf("tagNumber missing in metadata")
	}

	tagNumber, err := strconv.Atoi(tagNumberStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tagNumber: %w", err)
	}

	if err := h.commandRouter.CreateUser(context.Background(), req.DiscordID, req.Name, req.Role, tagNumber); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return nil, nil
}

// HandleUpdateUser updates an existing user (Command).
func (h *UserHandlers) HandleUpdateUser(msg *message.Message) ([]*message.Message, error) {
	var req usercommands.UpdateUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := h.commandRouter.UpdateUser(context.Background(), req.DiscordID, req.Updates); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return nil, nil
}

// HandleGetUser retrieves a user (Query) by delegating to QueryService.
func (h *UserHandlers) HandleGetUser(msg *message.Message) ([]*message.Message, error) {
	var req GetUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Delegate query logic to QueryService
	user, err := h.queryService.GetUserByDiscordID(context.Background(), req.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found: %s", req.DiscordID)
	}

	response := GetUserResponse{User: *user}
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user response: %w", err)
	}

	respMsg := message.NewMessage(watermill.NewUUID(), payload)
	respMsg.Metadata.Set("topic", userhandlers.TopicGetUserResponse)

	return []*message.Message{respMsg}, nil
}

// HandleGetUser Role retrieves a user's role (Query) by delegating to QueryService.
func (h *UserHandlers) HandleGetUserRole(msg *message.Message) ([]*message.Message, error) {
	var req GetUserRoleRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		log.Printf("Error unmarshalling GetUser RoleRequest: %v", err)
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	log.Printf("Request to get user role for Discord ID: %s", req.DiscordID)

	// Delegate query logic to QueryService
	role, err := h.queryService.GetUserRole(context.Background(), req.DiscordID)
	if err != nil {
		log.Printf("Error getting user role for Discord ID %s: %v", req.DiscordID, err)
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}

	response := UserRoleResponseEvent{Role: role}
	payload, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling role response: %v", err)
		return nil, fmt.Errorf("failed to marshal role response: %w", err)
	}

	respMsg := message.NewMessage(watermill.NewUUID(), payload)
	respMsg.Metadata.Set("topic", userhandlers.TopicGetUserRoleResponse)

	return []*message.Message{respMsg}, nil
}

// HandleGetUserWrapper adapts HandleGetUser to NoPublishHandlerFunc signature.
func (h *UserHandlers) HandleGetUserWrapper(msg *message.Message) error {
	_, err := h.HandleGetUser(msg)
	return err
}

// HandleGetUser RoleWrapper adapts HandleGetUser Role to NoPublishHandlerFunc signature.
func (h *UserHandlers) HandleGetUserRoleWrapper(msg *message.Message) error {
	log.Println("HandleGetUser RoleWrapper called")
	_, err := h.HandleGetUserRole(msg)
	if err != nil {
		log.Printf("Error in HandleGetUser Role: %v", err)
	}
	return err
}
