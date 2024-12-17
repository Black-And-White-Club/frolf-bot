package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// CreateUserHandler handles the CreateUserCommand.
type CreateUserHandler struct {
	userDB   userdb.UserDB
	eventBus watermillutil.PubSuber
}

// NewCreateUserHandler creates a new CreateUserHandler.
func NewCreateUserHandler(userDB userdb.UserDB, eventBus watermillutil.PubSuber) *CreateUserHandler {
	return &CreateUserHandler{
		userDB:   userDB,
		eventBus: eventBus,
	}
}

// Handle processes the CreateUserCommand.
func (h *CreateUserHandler) Handle(ctx context.Context, msg *message.Message) error {
	// 1. Unmarshal the CreateUserCommand from the message payload.
	var cmd usercommands.CreateUserRequest
	marshaler := watermillutil.Marshaler // Use the marshaler from userhandlers
	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal CreateUserCommand")
	}

	// 2. Set default role if not provided
	if cmd.Role == "" {
		cmd.Role = userdb.UserRoleRattler
	} else if !cmd.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", cmd.Role)
	}

	// 3. Check if a user with this Discord ID already exists
	existingUser, err := h.userDB.GetUserByDiscordID(ctx, cmd.DiscordID)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing user")
	}

	if existingUser != nil {
		return fmt.Errorf("user with Discord ID %s already exists", cmd.DiscordID)
	}

	// 4. Check tag availability using PubSub
	tagAvailable, err := h.checkTagAvailability(ctx, cmd.TagNumber)
	if err != nil {
		return errors.Wrap(err, "failed to check tag availability")
	}

	if tagAvailable {
		// 5. Create the user in the database
		err := h.userDB.CreateUser(ctx, cmd.DiscordID, cmd.Name, cmd.Role) // Remove tagNumber
		if err != nil {
			return errors.Wrap(err, "failed to create user in database")
		}

		// 6. Publish a UserCreatedEvent using the publisher from PubSub
		userCreatedEvent := UserCreatedEvent{ // Assuming you have this event struct defined
			DiscordID: cmd.DiscordID,
			TagNumber: cmd.TagNumber, // Still include tag number in the event
			// ... add other relevant data from the command ...
		}

		payload, err := json.Marshal(userCreatedEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal UserCreatedEvent: %w", err)
		}

		if err := h.eventBus.Publish(TopicCreateUser, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
			return errors.Wrap(err, "failed to publish UserCreatedEvent")
		}
	} else {
		return fmt.Errorf("tag number %d is already in use", cmd.TagNumber)
	}

	return nil
}

// checkTagAvailability checks if the tag number is available.
func (h *CreateUserHandler) checkTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	// Create a new context with a timeout (e.g., 5 seconds)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel() // Ensure the timeout is canceled

	// 1. Prepare the request data for the leaderboard module
	checkTagEvent := &CheckTagAvailabilityEvent{
		TagNumber: tagNumber,
	}

	// 2. Publish the request using PubSub
	payload, err := json.Marshal(checkTagEvent) // Marshal using encoding/json
	if err != nil {
		return false, fmt.Errorf("failed to marshal CheckTagAvailabilityEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicCheckTagRequest, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return false, errors.Wrap(err, "failed to publish check tag availability event")
	}

	// 3. Subscribe to the response topic using PubSub
	responseChan, err := h.eventBus.Subscribe(ctx, "tag-availability-result")
	if err != nil {
		return false, errors.Wrap(err, "failed to subscribe to tag availability result")
	}

	// 4. Wait for the response
	select {
	case msg := <-responseChan:
		// Unmarshal the response from msg.Payload
		var tagAvailable bool
		if err := json.Unmarshal(msg.Payload, &tagAvailable); err != nil {
			return false, fmt.Errorf("failed to unmarshal tag availability response: %w", err)
		}
		return tagAvailable, nil
	case <-ctx.Done(): // Handle timeout or cancellation
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf("timeout waiting for tag availability response")
		}
		return false, ctx.Err() // Return the context error (e.g., cancellation)
	}
}
