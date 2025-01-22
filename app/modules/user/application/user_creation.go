package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
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

	// Create the user in the database
	user := &usertypes.UserData{
		DiscordID: discordID,
		Role:      usertypes.UserRoleRattler,
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
			return fmt.Errorf("failed to publish UserCreationFailed event: %w", pubErr)
		}

		// Return nil as we've handled the database error by publishing an event
		return nil
	}

	s.logger.Info("User created successfully",
		slog.String("discord_id", string(discordID)),
		slog.String("correlation_id", correlationID),
	)

	// Publish a UserCreated event
	return s.PublishUserCreated(ctx, msg, discordID, tag)
}

// PublishUserCreated publishes a UserCreated event.
func (s *UserServiceImpl) PublishUserCreated(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	payload := userevents.UserCreatedPayload{
		DiscordID: discordID,
	}

	if tag != nil {
		payload.TagNumber = tag
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg = message.NewMessage(correlationID, payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	if err := s.eventBus.Publish(ctx, userevents.UserCreated, msg); err != nil {
		s.logger.Error("Failed to publish UserCreated event",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		// Consider implementing a retry mechanism or a dead-letter queue here
		return fmt.Errorf("failed to publish UserCreated event: %w", err)
	}

	s.logger.Info("Published UserCreated event", slog.String("correlation_id", correlationID))

	return nil
}

// PublishUserCreationFailed publishes a UserCreationFailed event.
func (s *UserServiceImpl) PublishUserCreationFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int, reason string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	payload := userevents.UserCreationFailedPayload{
		Reason: reason,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg = message.NewMessage(correlationID, payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	if err := s.eventBus.Publish(ctx, userevents.UserCreationFailed, msg); err != nil {
		return fmt.Errorf("failed to publish UserCreationFailed event: %w", err)
	}

	s.logger.Info("UserCreationFailed event published", slog.String("correlationID", correlationID))
	return nil
}
