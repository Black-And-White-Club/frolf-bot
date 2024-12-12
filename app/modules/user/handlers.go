package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandlers defines the handlers for user-related events.
type UserHandlers struct {
	commandService usercommands.CommandService
	queryService   userqueries.QueryService
	pubsub         watermillutil.PubSuber // Use the PubSuber interface
}

func NewUserHandlers(commandService usercommands.CommandService, queryService userqueries.QueryService, pubsub watermillutil.PubSuber) *UserHandlers { // Use PubSuber interface
	return &UserHandlers{
		commandService: commandService,
		queryService:   queryService,
		pubsub:         pubsub,
	}
}

// HandleCreateUser creates a new user.
func (h *UserHandlers) HandleCreateUser(msg *message.Message) ([]*message.Message, error) { // Updated return signature
	var req userhandlers.CreateUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return []*message.Message{}, fmt.Errorf("invalid request: %w", err) // Return empty slice and error
	}

	// Extract tagNumber from metadata
	tagNumberStr := msg.Metadata.Get("tagNumber") // Get returns only the value
	if tagNumberStr == "" {
		return []*message.Message{}, fmt.Errorf("tagNumber missing in metadata")
	}

	tagNumber, err := strconv.Atoi(tagNumberStr)
	if err != nil {
		return []*message.Message{}, fmt.Errorf("invalid tagNumber: %w", err)
	}

	if err := h.commandService.CreateUser(context.Background(), req.DiscordID, req.Name, req.Role, tagNumber); err != nil {
		return []*message.Message{}, fmt.Errorf("failed to create user: %w", err)
	}

	return []*message.Message{}, nil // Return an empty slice of messages and nil error
}

// HandleGetUser retrieves a user.
func (h *UserHandlers) HandleGetUser(msg *message.Message) ([]*message.Message, error) { // Updated return signature
	var req userhandlers.GetUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return []*message.Message{}, fmt.Errorf("invalid request: %w", err)
	}

	user, err := h.queryService.GetUserByDiscordID(context.Background(), req.DiscordID)
	if err != nil {
		return []*message.Message{}, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return []*message.Message{}, fmt.Errorf("user not found: %s", req.DiscordID)
	}

	// Publish the user data to a response topic (you'll need to define this topic)
	response := userhandlers.GetUserResponse{User: *user}
	payload, err := json.Marshal(response)
	if err != nil {
		return []*message.Message{}, fmt.Errorf("failed to marshal user response: %w", err)
	}
	if err := h.pubsub.Publish("get-user-response", message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return []*message.Message{}, fmt.Errorf("failed to publish user response: %w", err)
	}

	return []*message.Message{}, nil // Return an empty slice of messages and nil error
}

// HandleUpdateUser updates an existing user.
func (h *UserHandlers) HandleUpdateUser(msg *message.Message) ([]*message.Message, error) { // Updated return signature
	var req userhandlers.UpdateUserRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return []*message.Message{}, fmt.Errorf("invalid request: %w", err)
	}

	// ... (authorization logic remains similar, but use req.DiscordID) ...

	cmd := userhandlers.UpdateUserRequest{
		DiscordID: req.DiscordID,
		Updates:   req.Updates,
	}

	if err := h.commandService.UpdateUser(context.Background(), cmd.DiscordID, cmd.Updates); err != nil {
		return []*message.Message{}, fmt.Errorf("failed to update user: %w", err)
	}

	return []*message.Message{}, nil // Return an empty slice of messages and nil error
}

// HandleGetUserRole retrieves the role of a user.
func (h *UserHandlers) HandleGetUserRole(msg *message.Message) ([]*message.Message, error) { // Updated return signature
	var req userhandlers.GetUserRoleRequestEvent
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return []*message.Message{}, fmt.Errorf("invalid request: %w", err)
	}

	role, err := h.queryService.GetUserRole(context.Background(), req.DiscordID)
	if err != nil {
		return []*message.Message{}, fmt.Errorf("failed to get user role: %w", err)
	}

	// Publish the role to a response topic (you'll need to define this topic)
	response := userhandlers.UserRoleResponseEvent{Role: role}
	payload, err := json.Marshal(response)
	if err != nil {
		return []*message.Message{}, fmt.Errorf("failed to marshal role response: %w", err)
	}
	if err := h.pubsub.Publish("get-user-role-response", message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return []*message.Message{}, fmt.Errorf("failed to publish role response: %w", err)
	}

	return []*message.Message{}, nil // Return an empty slice of messages and nil error
}
