package userservice

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// createUser creates a new user in the database.
func (s *UserServiceImpl) createUser(ctx context.Context, discordID usertypes.DiscordID, role usertypes.UserRoleEnum) error {
	newUser := &userdb.User{
		DiscordID: discordID,
		Role:      role,
	}
	if err := s.UserDB.CreateUser(ctx, newUser); err != nil {
		// Consider publishing a user.signup.failed event here with the error details
		return fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User successfully created", slog.String("discord_id", string(discordID)))
	return nil
}

// publishUserCreated publishes a UserCreated event after the user is created.
func (s *UserServiceImpl) publishUserCreated(ctx context.Context, payload events.UserCreatedPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal UserCreatedPayload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.SetContext(ctx)
	msg.Metadata.Set("subject", events.UserCreated)

	if err := s.eventBus.Publish(ctx, events.UserStreamName, msg); err != nil {
		return fmt.Errorf("failed to publish UserCreated event: %w", err)
	}

	return nil
}
