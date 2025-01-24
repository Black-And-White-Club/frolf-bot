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

func (s *UserServiceImpl) GetUserRole(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	role, err := s.UserDB.GetUserRole(ctx, discordID)
	if err != nil {
		s.logger.Error("Failed to get user role",
			slog.Any("error", err),
			slog.String("discordID", string(discordID)),
			slog.String("correlation_id", correlationID),
		)
		// Publish a GetUserRoleFailed event and return the error
		if pubErr := s.publishGetUserRoleFailed(ctx, msg, discordID, err.Error()); pubErr != nil {
			return fmt.Errorf("failed to get user role and publish GetUserRoleFailed event: %w", pubErr)
		}
		return fmt.Errorf("failed to get user role: %w", err) // Return the error
	}

	// Publish a GetUserRoleResponse event
	return s.publishGetUserRoleResponse(ctx, msg, discordID, role)
}

// GetUser retrieves user data and publishes a GetUserResponse event.
func (s *UserServiceImpl) GetUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		s.logger.Error("Failed to get user",
			slog.Any("error", err),
			slog.String("discordID", string(discordID)),
			slog.String("correlation_id", correlationID),
		)
		// Publish a GetUserFailed event and then return the error
		if pubErr := s.publishGetUserFailed(ctx, msg, discordID, err.Error()); pubErr != nil {
			return fmt.Errorf("failed to get user and publish GetUserFailed event: %w", pubErr)
		}
		return fmt.Errorf("failed to get user: %w", err) // Return the error
	}

	// Publish a GetUserResponse event
	return s.publishGetUserResponse(ctx, msg, user)
}

// publishGetUserRoleResponse publishes a GetUserRoleResponse event.
func (s *UserServiceImpl) publishGetUserRoleResponse(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	payloadBytes, err := json.Marshal(userevents.GetUserRoleResponsePayload{
		DiscordID: string(discordID),
		Role:      string(role),
	})
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	s.eventUtil.PropagateMetadata(msg, newMsg) // Use eventutil to copy metadata

	// Set the context on the new message
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.GetUserRoleResponse, newMsg); err != nil {
		s.logger.Error("Failed to publish GetUserRoleResponse event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish GetUserRoleResponse event: %w", err)
	}

	s.logger.Info("Published GetUserRoleResponse event", slog.String("correlation_id", correlationID))
	return nil
}

// publishGetUserResponse publishes a GetUserResponse event.
// publishGetUserResponse publishes a GetUserResponse event.
func (s *UserServiceImpl) publishGetUserResponse(ctx context.Context, msg *message.Message, user *userdb.User) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	userData := &usertypes.UserData{
		ID:        user.ID,
		DiscordID: user.DiscordID,
		Role:      user.Role,
	}
	payloadBytes, err := json.Marshal(userevents.GetUserResponsePayload{
		User: userData,
	})
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	s.eventUtil.PropagateMetadata(msg, newMsg) // Use eventutil to copy metadata

	// Set the context on the new message
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.GetUserResponse, newMsg); err != nil {
		s.logger.Error("Failed to publish GetUserResponse event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish GetUserResponse event: %w", err)
	}

	s.logger.Info("Published GetUserResponse event", slog.String("correlation_id", correlationID))
	return nil
}

// publishGetUserRoleFailed publishes a GetUserRoleFailed event.
func (s *UserServiceImpl) publishGetUserRoleFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, reason string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	payloadBytes, err := json.Marshal(userevents.GetUserRoleFailedPayload{
		DiscordID: string(discordID),
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
	s.eventUtil.PropagateMetadata(msg, newMsg) // Use eventutil to copy metadata

	// Set the context on the new message
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.GetUserRoleFailed, newMsg); err != nil {
		s.logger.Error("Failed to publish GetUserRoleFailed event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish GetUserRoleFailed event: %w", err)
	}

	s.logger.Info("Published GetUserRoleFailed event", slog.String("correlation_id", correlationID))
	return nil
}

// publishGetUserFailed publishes a GetUserFailed event.
func (s *UserServiceImpl) publishGetUserFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, reason string) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	payloadBytes, err := json.Marshal(userevents.GetUserFailedPayload{
		DiscordID: string(discordID),
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
	s.eventUtil.PropagateMetadata(msg, newMsg) // Use eventutil to copy metadata

	// Set the context on the new message
	newMsg.SetContext(ctx)

	if err := s.eventBus.Publish(userevents.GetUserFailed, newMsg); err != nil {
		s.logger.Error("Failed to publish GetUserFailed event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish GetUserFailed event: %w", err)
	}

	s.logger.Info("Published GetUserFailed event", slog.String("correlation_id", correlationID))
	return nil
}
