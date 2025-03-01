package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func (s *UserServiceImpl) CreateUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error {
	// Get correlationID from message metadata
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	s.logger.Info("Creating user",
		slog.String("user_id", string(discordID)),
		slog.String("correlation_id", correlationID),
	)

	// Create the user in the database using the userdb.User model
	userData := usertypes.UserData{
		DiscordID: usertypes.DiscordID(discordID),
	}

	// Convert user_data to userdb.User type
	user := userdb.User{
		DiscordID: userData.DiscordID,
	}

	// Attempt to create the user and handle potential errors
	if err := s.UserDB.CreateUser(ctx, &user); err != nil {
		s.logger.Error("Failed to create user",
			slog.Any("error", err),
			slog.String("user_id", string(discordID)),
			slog.String("correlation_id", correlationID),
		)

		// Publish a UserCreationFailed event
		if pubErr := s.PublishUserCreationFailed(ctx, msg, discordID, tag, err.Error()); pubErr != nil {
			// Handle error if publishing the failure event fails
			return fmt.Errorf("failed to create user and publish UserCreationFailed event: %w", pubErr)
		}

		// Return nil as we've handled the database error by publishing an event
		return nil
	}

	s.logger.Info("User created successfully",
		slog.String("user_id", string(discordID)),
		slog.String("correlation_id", correlationID),
	)

	// Publish a UserCreated event
	return s.PublishUserCreated(ctx, msg, discordID, tag)
}

// PublishUserCreated publishes a UserCreated event.
func (s *UserServiceImpl) PublishUserCreated(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error {
	// Get correlationID from message metadata
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	// Prepare the event payload
	eventPayload := &userevents.UserCreatedPayload{
		DiscordID: usertypes.DiscordID(discordID),
	}
	if tag != nil {
		eventPayload.TagNumber = tag
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		s.logger.Error("Failed to marshal UserCreatedPayload",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to marshal UserCreatedPayload: %w", err)
	}

	// Create a new message with the payload
	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Set the correlation ID for the new message
	newMessage.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	// Set the Nats-Msg-Id for deduplication using msg.UUID
	newMessage.Metadata.Set("Nats-Msg-Id", newMessage.UUID)

	// Copy the discord ID from the original message metadata
	guildID := msg.Metadata.Get("guild_id")
	newMessage.Metadata.Set("guild_id", guildID)
	interactionID := msg.Metadata.Get("interaction_id")
	newMessage.Metadata.Set("interaction_id", interactionID)
	interactionToken := msg.Metadata.Get("interaction_token")
	newMessage.Metadata.Set("interaction_token", interactionToken)

	// Copy the discord ID from the original message metadata
	if discordID := msg.Metadata.Get("user_id"); discordID != "" {
		newMessage.Metadata.Set("user_id", discordID)
	}

	// Publish the event
	if err := s.eventBus.Publish("discord.user.signup.success", newMessage); err != nil {
		s.logger.Error("Failed to publish UserCreated event",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to publish UserCreated event: %w", err)
	}

	s.logger.Info("Published UserCreated event",
		slog.String("correlation_id", correlationID),
		slog.String("message_id", newMessage.UUID),
	)

	return nil
}

// PublishUserCreationFailed publishes a UserCreationFailed event.
func (s *UserServiceImpl) PublishUserCreationFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int, reason string) error {
	// Get correlationID from message metadata
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	// Prepare the event payload
	eventPayload := &userevents.UserCreationFailedPayload{
		DiscordID: discordID, // This is where you set the DiscordID
		Reason:    reason,
	}
	if tag != nil {
		eventPayload.TagNumber = tag
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		s.logger.Error("Failed to marshal UserCreationFailedPayload",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to marshal UserCreationFailedPayload: %w", err)
	}

	// Create a new message with the payload
	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Set the correlation ID for the new message
	newMessage.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	// Publish the event
	if err := s.eventBus.Publish("discord.user.signup.failed", newMessage); err != nil {
		s.logger.Error("Failed to publish UserCreationFailed event",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to publish UserCreationFailed event: %w", err)
	}

	s.logger.Info("Published UserCreationFailed event",
		slog.String("correlation_id", correlationID),
		slog.String("message_id", newMessage.UUID),
	)

	return nil
}
