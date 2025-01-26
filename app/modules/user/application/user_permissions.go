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

// CheckUserPermissions starts the permission check process by publishing a UserPermissionsCheckRequest event.
func (s *UserServiceImpl) CheckUserPermissions(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Initiating user permissions check",
		slog.String("discord_id", string(discordID)),
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

	// Set the context on the new message
	newMsg.SetContext(ctx)

	// Publish the event
	if err := s.eventBus.Publish(userevents.UserPermissionsCheckRequest, newMsg); err != nil {
		s.logger.Error("Failed to publish UserPermissionsCheckRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish UserPermissionsCheckRequest event: %w", err)
	}

	s.logger.Info("Published UserPermissionsCheckRequest event",
		slog.String("correlation_id", correlationID),
	)
	return nil
}

// CheckUserPermissionsInDB checks if the requesting user has the required permissions in the database
func (s *UserServiceImpl) CheckUserPermissionsInDB(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	s.logger.Info("Checking user permissions in DB",
		slog.String("user_id", string(discordID)),
		slog.String("role", string(role)),
		slog.String("requester_id", requesterID),
		slog.String("correlation_id", correlationID),
	)

	// Get the requesting user from the database
	requester, err := s.UserDB.GetUserByDiscordID(ctx, usertypes.DiscordID(requesterID))
	if err != nil {
		s.logger.Error("Failed to get requesting user",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		// Publish a UserPermissionsCheckFailed event
		return fmt.Errorf("failed to get requesting user: %w",
			s.PublishUserPermissionsCheckFailed(ctx, msg, discordID, role, requesterID, "Failed to get requesting user"),
		)
	}

	// Check if the requesting user has the required role
	if requester.GetRole() != role {
		s.logger.Info("Requester does not have required role",
			slog.String("correlation_id", correlationID),
			slog.String("requester_id", requesterID),
			slog.String("required_role", string(role)),
		)
		// Publish a UserPermissionsCheckFailed event
		return fmt.Errorf("requester does not have required role: %w",
			s.PublishUserPermissionsCheckFailed(ctx, msg, discordID, role, requesterID, "Requester does not have required role"),
		)
	}

	// Publish a UserPermissionsCheckResponse event indicating permission granted
	return s.PublishUserPermissionsCheckResponse(ctx, msg, discordID, role, requesterID, true, "")
}

// PublishUserPermissionsCheckResponse publishes a UserPermissionsCheckResponse event.
func (s *UserServiceImpl) PublishUserPermissionsCheckResponse(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string, hasPermission bool, reason string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Publishing UserPermissionsCheckResponse event",
		slog.String("correlation_id", correlationID),
		slog.Bool("has_permission", hasPermission),
		slog.String("reason", reason),
	)

	payloadBytes, err := json.Marshal(userevents.UserPermissionsCheckResponsePayload{
		HasPermission: hasPermission,
		DiscordID:     discordID,
		Role:          role,
		RequesterID:   requesterID,
	})
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

	if err := s.eventBus.Publish(userevents.UserPermissionsCheckResponse, newMsg); err != nil {
		s.logger.Error("Failed to publish UserPermissionsCheckResponse event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish UserPermissionsCheckResponse event: %w", err)
	}

	s.logger.Info("Published UserPermissionsCheckResponse event",
		slog.String("correlation_id", correlationID),
		slog.String("message_id", newMsg.UUID),
	)
	return nil
}

func (s *UserServiceImpl) PublishUserPermissionsCheckFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string, reason string) error {
	// Extract correlation ID from the original message
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	payload := userevents.UserPermissionsCheckFailedPayload{
		DiscordID:   discordID,
		Role:        role,
		RequesterID: requesterID,
		Reason:      reason,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create a new message with the correlation ID
	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	newMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	// Publish the event
	if err := s.eventBus.Publish(userevents.UserPermissionsCheckFailed, newMsg); err != nil {
		return fmt.Errorf("failed to publish UserPermissionsCheckFailed event: %w", err)
	}

	s.logger.Info("Published UserPermissionsCheckFailed event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(discordID)),
		slog.String("role", string(role)),
		slog.String("requester_id", requesterID),
		slog.String("reason", reason),
	)
	return nil
}
