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
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := s.eventBus.Publish(ctx, userevents.GetUserRoleResponse, msg); err != nil {
		return fmt.Errorf("failed to publish GetUserRoleResponse event: %w", err)
	}

	s.logger.Info("Published GetUserRoleResponse event", slog.String("correlation_id", correlationID))
	return nil
}

// publishGetUserResponse publishes a GetUserResponse event.
func (s *UserServiceImpl) publishGetUserResponse(ctx context.Context, msg *message.Message, user usertypes.User) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	payloadBytes, err := json.Marshal(userevents.GetUserResponsePayload{
		User: user.(*usertypes.UserData),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := s.eventBus.Publish(ctx, userevents.GetUserResponse, msg); err != nil {
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
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := s.eventBus.Publish(ctx, userevents.GetUserRoleFailed, msg); err != nil {
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
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := s.eventBus.Publish(ctx, userevents.GetUserFailed, msg); err != nil {
		return fmt.Errorf("failed to publish GetUserFailed event: %w", err)
	}

	s.logger.Info("Published GetUserFailed event", slog.String("correlation_id", correlationID))
	return nil
}
