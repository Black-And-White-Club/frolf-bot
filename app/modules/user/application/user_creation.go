package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

func (s *UserServiceImpl) CreateUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error {
	// Get correlationID from message metadata
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	s.logger.Info("Creating user",
		slog.String("discord_id", string(discordID)),
		slog.String("correlation_id", correlationID),
	)

	// Create the user in the database using the userdb.User model
	user := &userdb.User{
		DiscordID: discordID,
		// Role will be set to the default value by the database
	}

	// Attempt to create the user and handle potential errors
	if err := s.UserDB.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user",
			slog.Any("error", err),
			slog.String("discord_id", string(discordID)),
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

	s.logger.Info("CALLED FROM SERVICE User created successfully",
		slog.String("discord_id", string(discordID)),
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
		DiscordID: discordID,
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

	// Publish the event
	if err := s.eventBus.Publish(userevents.UserCreated, newMessage); err != nil {
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
		DiscordID: discordID,
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

	// Set the Nats-Msg-Id for deduplication using msg.UUID
	newMessage.Metadata.Set("Nats-Msg-Id", newMessage.UUID)

	// Publish the event
	if err := s.eventBus.Publish(userevents.UserCreationFailed, newMessage); err != nil {
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
