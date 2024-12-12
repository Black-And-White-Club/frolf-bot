// user/message_handlers.go
package user

import (
	"context"
	"encoding/json"
	"fmt"
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
	commandRouter userrouter.CommandRouter
	queryService  userqueries.QueryService
	pubsub        watermillutil.PubSuber
}

func NewUserHandlers(commandRouter userrouter.CommandRouter, queryService userqueries.QueryService, pubsub watermillutil.PubSuber) *UserHandlers {
	return &UserHandlers{
		commandRouter: commandRouter,
		queryService:  queryService,
		pubsub:        pubsub,
	}
}

// Handle implements the MessageHandler interface.
func (h *UserHandlers) Handle(msg *message.Message) ([]*message.Message, error) {
	switch msg.Metadata.Get("topic") {
	case userhandlers.TopicCreateUser:
		return h.HandleCreateUser(msg)
	case userhandlers.TopicGetUser:
		return h.HandleGetUser(msg)
	case userhandlers.TopicUpdateUser:
		return h.HandleUpdateUser(msg)
	case userhandlers.TopicGetUserRoleRequest:
		return h.HandleGetUserRole(msg)
	default:
		return nil, fmt.Errorf("unknown message topic: %s", msg.Metadata.Get("topic"))
	}
}

// HandleCreateUser creates a new user.
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

// HandleGetUser retrieves a user.
func (h *UserHandlers) HandleGetUser(msg *message.Message) ([]*message.Message, error) {
	var req GetUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

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

// HandleUpdateUser updates an existing user.
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

// HandleGetUserRole retrieves the role of a user.
func (h *UserHandlers) HandleGetUserRole(msg *message.Message) ([]*message.Message, error) {
	var req GetUserRoleRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	role, err := h.queryService.GetUserRole(context.Background(), req.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role: %w", err)
	}

	response := UserRoleResponseEvent{Role: role}
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role response: %w", err)
	}

	respMsg := message.NewMessage(watermill.NewUUID(), payload)
	respMsg.Metadata.Set("topic", userhandlers.TopicGetUserRoleResponse)

	return []*message.Message{respMsg}, nil
}
