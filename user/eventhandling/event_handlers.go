package usereventhandling

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/user/queries"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserEventHandlerImpl implements the UserEventHandler interface.
type UserEventHandlerImpl struct {
	commandService commandsinterface.CommandService
	queryService   queries.QueryService
	publisher      message.Publisher
}

// NewUserEventHandler creates a new UserEventHandlerImpl.
func NewUserEventHandler(commandService commandsinterface.CommandService, queryService queries.QueryService, publisher message.Publisher) *UserEventHandlerImpl {
	return &UserEventHandlerImpl{
		commandService: commandService,
		queryService:   queryService,
		publisher:      publisher,
	}
}

// HandleUserCreated handles the UserCreatedEvent.
func (h *UserEventHandlerImpl) HandleUserCreated(ctx context.Context, event UserCreatedEvent) error {
	fmt.Printf("User created: %+v\n", event)

	// Delegate user creation logic to the command service
	user := db.User{
		DiscordID: event.DiscordID,
		// Set other user properties as needed
	}
	if err := h.commandService.CreateUser(ctx, &user, event.TagNumber); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// HandleUserUpdated handles the UserUpdatedEvent.
func (h *UserEventHandlerImpl) HandleUserUpdated(ctx context.Context, event UserUpdatedEvent) error {
	fmt.Printf("User updated: %+v\n", event)

	// Delegate user update logic to the command service
	updates := db.User{
		Name: event.Name,
		Role: event.Role,
	}
	if err := h.commandService.UpdateUser(ctx, event.DiscordID, &updates); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// HandleCheckTagAvailability handles the CheckTagAvailabilityEvent.
func (h *UserEventHandlerImpl) HandleCheckTagAvailability(ctx context.Context, event CheckTagAvailabilityEvent) error {
	fmt.Printf("Check tag availability: %+v\n", event)

	// Publish the event to the leaderboard module
	if err := h.publisher.Publish(
		"check-tag-availability-stream",
		message.NewMessage(watermill.NewUUID(), // Generate a unique message ID
			[]byte(`{"tag_number": event.TagNumber}`)), // Message payload
	); err != nil {
		return fmt.Errorf("failed to publish CheckTagAvailabilityEvent: %w", err)
	}

	return nil
}

// HandleGetUserRole handles the GetUserRoleEvent.
func (h *UserEventHandlerImpl) HandleGetUserRole(ctx context.Context, event GetUserRoleEvent) error {
	fmt.Printf("Get user role: %+v\n", event)

	// Delegate user role retrieval to the query service
	role, err := h.queryService.GetUserRole(ctx, event.DiscordID)
	if err != nil {
		return fmt.Errorf("failed to get user role: %w", err)
	}

	// Publish UserRoleResponseEvent
	responseEvent := UserRoleResponseEvent{
		Role: role,
	}
	if err := h.publisher.Publish("user-role-result-stream", message.NewMessage(watermill.NewUUID(), []byte(`{"role": responseEvent.Role}`))); err != nil {
		return fmt.Errorf("failed to publish user-role-result event: %w", err)
	}

	return nil
}
