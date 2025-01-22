package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UpdateUserRole starts the user role update process by publishing a UserPermissionsCheckRequest event.
func (s *UserServiceImpl) UpdateUserRole(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, role, requesterID string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Starting user role update process",
		slog.String("user_id", string(userID)),
		slog.String("role", role),
		slog.String("requester_id", requesterID),
		slog.String("correlation_id", correlationID),
	)

	// Publish a UserPermissionsCheckRequest event
	payloadBytes, err := json.Marshal(userevents.UserPermissionsCheckRequestPayload{
		DiscordID:   userID,
		Role:        role,
		RequesterID: requesterID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Use correlationID for the message
	msg = message.NewMessage(correlationID, payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	if err := s.eventBus.Publish(ctx, userevents.UserPermissionsCheckRequest, msg); err != nil {
		return fmt.Errorf("failed to publish UserPermissionsCheckRequest event: %w", err)
	}

	s.logger.Info("Published UserPermissionsCheckRequest event", slog.String("correlation_id", correlationID))
	return nil
}

// UpdateUserRoleInDatabase updates the user's role in the database.
func (s *UserServiceImpl) UpdateUserRoleInDatabase(ctx context.Context, userID string, role, correlationID string) error {
	err := s.UserDB.UpdateUserRole(ctx, usertypes.DiscordID(userID), usertypes.UserRoleEnum(role))
	if err != nil {
		s.logger.Error("Failed to update user role in database",
			slog.String("user_id", userID),
			slog.String("role", role),
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to update user role in database: %w", err)
	}
	return nil
}

// PublishUserRoleUpdated publishes a UserRoleUpdated event.
func (s *UserServiceImpl) PublishUserRoleUpdated(ctx context.Context, msg *message.Message, userID, role string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Publishing UserRoleUpdated event",
		slog.String("user_id", userID),
		slog.String("role", role),
		slog.String("correlation_id", correlationID),
	)

	payloadBytes, err := json.Marshal(userevents.UserRoleUpdatedPayload{
		DiscordID: userID,
		Role:      role,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	if err := s.eventBus.Publish(ctx, userevents.UserRoleUpdated, msg); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	s.logger.Info("Published UserRoleUpdated event", slog.String("correlation_id", correlationID))
	return nil
}

// PublishUserRoleUpdateFailed publishes a UserRoleUpdateFailed event.
func (s *UserServiceImpl) PublishUserRoleUpdateFailed(ctx context.Context, msg *message.Message, userID, role, reason string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Publishing UserRoleUpdateFailed event",
		slog.String("user_id", userID),
		slog.String("role", role),
		slog.String("correlation_id", correlationID),
		slog.String("reason", reason),
	)

	payloadBytes, err := json.Marshal(userevents.UserRoleUpdateFailedPayload{
		DiscordID: userID,
		Role:      role,
		Reason:    reason,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(correlationID, payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	if err := s.eventBus.Publish(ctx, userevents.UserRoleUpdateFailed, msg); err != nil {
		return fmt.Errorf("failed to publish UserRoleUpdateFailed event: %w", err)
	}

	s.logger.Info("Published UserRoleUpdateFailed event", slog.String("correlation_id", correlationID))
	return nil
}
