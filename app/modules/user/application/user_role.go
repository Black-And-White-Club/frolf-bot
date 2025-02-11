package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UpdateUserRole starts the user role update process by publishing a UserPermissionsCheckRequest event.
func (s *UserServiceImpl) UpdateUserRole(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Starting user role update process",
		slog.String("user_id", string(discordID)),
		slog.String("role", string(role)),
		slog.String("requester_id", requesterID),
		slog.String("correlation_id", correlationID),
	)

	// Publish a UserPermissionsCheckRequest event
	eventPayload := userevents.UserPermissionsCheckRequestPayload{
		DiscordID:   discordID,
		Role:        role,
		RequesterID: requesterID,
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create a new message for the outgoing event
	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Copy the correlation ID to the new message's metadata
	s.eventUtil.PropagateMetadata(msg, newMsg)
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.UserPermissionsCheckRequest, newMsg); err != nil {
		s.logger.Error("Failed to publish UserPermissionsCheckRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish UserPermissionsCheckRequest event: %w", err)
	}

	s.logger.Info("Published UserPermissionsCheckRequest event", slog.String("correlation_id", correlationID))
	return nil
}

// UpdateUserRoleInDatabase updates the user's role in the database.
func (s *UserServiceImpl) UpdateUserRoleInDatabase(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	err := s.UserDB.UpdateUserRole(ctx, usertypes.DiscordID(discordID), usertypes.UserRoleEnum(role))
	if err != nil {
		s.logger.Error("Failed to update user role in database",
			slog.String("user_id", string(discordID)),
			slog.String("role", string(role)),
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to update user role in database: %w", err)
	}
	return nil
}

// PublishUserRoleUpdated publishes a UserRoleUpdated event.
func (s *UserServiceImpl) PublishUserRoleUpdated(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Publishing UserRoleUpdated event",
		slog.String("user_id", string(discordID)),
		slog.String("role", string(role)),
		slog.String("correlation_id", correlationID),
	)

	payloadBytes, err := json.Marshal(userevents.UserRoleUpdatedPayload{
		DiscordID: discordID,
		Role:      role,
	})
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Copy the correlation ID to the new message's metadata
	s.eventUtil.PropagateMetadata(msg, newMsg)
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.UserRoleUpdated, newMsg); err != nil {
		s.logger.Error("Failed to publish UserRoleUpdated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	s.logger.Info("Published UserRoleUpdated event", slog.String("correlation_id", correlationID))
	return nil
}

// PublishUserRoleUpdateFailed publishes a UserRoleUpdateFailed event.
func (s *UserServiceImpl) PublishUserRoleUpdateFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, reason string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Publishing UserRoleUpdateFailed event",
		slog.String("user_id", string(discordID)),
		slog.String("role", string(role)),
		slog.String("correlation_id", correlationID),
		slog.String("reason", reason),
	)

	payloadBytes, err := json.Marshal(userevents.UserRoleUpdateFailedPayload{
		DiscordID: discordID,
		Role:      role,
		Reason:    reason,
	})
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Copy the correlation ID to the new message's metadata
	s.eventUtil.PropagateMetadata(msg, newMsg)
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.UserRoleUpdateFailed, newMsg); err != nil {
		s.logger.Error("Failed to publish UserRoleUpdateFailed event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish UserRoleUpdateFailed event: %w", err)
	}

	s.logger.Info("Published UserRoleUpdateFailed event", slog.String("correlation_id", correlationID))
	return nil
}
