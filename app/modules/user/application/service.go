package userservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

type UserService interface {
	Signup(ctx context.Context, discordID, tagNumber int) error
}

// UserServiceImpl (unexported)
type UserServiceImpl struct {
	UserDB                 userdb.UserDB
	eventBus               shared.EventBus
	logger                 *slog.Logger
	tagAvailabilityChecker TagAvailabilityChecker // Use the interface
}

// NewUserService creates a new UserService.
func NewUserService(userDB userdb.UserDB, eventBus shared.EventBus, logger *slog.Logger) *UserServiceImpl {
	s := &UserServiceImpl{
		UserDB:                 userDB,
		eventBus:               eventBus,
		logger:                 logger,
		tagAvailabilityChecker: nil, // Initialize to nil
	}
	// Set the concrete implementation of the interface
	s.tagAvailabilityChecker = s
	return s
}

// signupOrchestrator handles the detailed steps of the signup process.
func (s *UserServiceImpl) signupOrchestrator(ctx context.Context, req events.UserSignupRequestPayload) error {
	// 1. Check tag availability (if provided)
	if req.TagNumber != 0 {
		isTagAvailable, err := s.checkTagAvailability(ctx, req.TagNumber)
		if err != nil {
			s.logger.Error("Failed to check tag availability", slog.Any("error", err))
			return err
		}
		if !isTagAvailable {
			s.logger.Info("Tag number is not available", slog.Int("tag_number", req.TagNumber))
			return fmt.Errorf("tag number %d is already taken", req.TagNumber)
		}

		// 2. Publish TagAssignedRequest
		err = s.publishTagAssigned(ctx, req.DiscordID, req.TagNumber)
		if err != nil {
			s.logger.Error("Failed to publish TagAssignedRequest event", slog.Any("error", err))
			return err
		}
	}

	// 3. Create the user
	err := s.createUser(ctx, req.DiscordID, usertypes.UserRoleRattler)
	if err != nil {
		s.logger.Error("Failed to create user", slog.Any("error", err))
		return err
	}

	// 4. Publish UserCreated event
	err = s.publishUserCreated(ctx, events.UserCreatedPayload{
		DiscordID: req.DiscordID,
		TagNumber: req.TagNumber, // Include the tag number in the payload
		Role:      usertypes.UserRoleRattler,
	})
	if err != nil {
		s.logger.Error("Failed to publish UserCreated event", slog.Any("error", err))
		return err
	}

	return nil
}

// OnUserRoleUpdateRequest processes a user role update request.
func (s *UserServiceImpl) OnUserRoleUpdateRequest(ctx context.Context, req events.UserRoleUpdateRequestPayload) (*events.UserRoleUpdateResponsePayload, error) {
	if !req.NewRole.IsValid() {
		return nil, fmt.Errorf("invalid user role: %s", req.NewRole)
	}

	if req.DiscordID == "" {
		return nil, fmt.Errorf("missing DiscordID in request")
	}

	if err := s.UserDB.UpdateUserRole(ctx, req.DiscordID, req.NewRole); err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	user, err := s.UserDB.GetUserByDiscordID(ctx, req.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found: %s", req.DiscordID)
	}

	evt := events.UserRoleUpdatedPayload{
		DiscordID: req.DiscordID,
		NewRole:   req.NewRole.String(),
	}

	if err := s.publishUserRoleUpdated(ctx, evt); err != nil {
		return nil, fmt.Errorf("failed to publish UserRoleUpdated event: %w", err)
	}

	return &events.UserRoleUpdateResponsePayload{
		Success: true,
	}, nil
}

// GetUserRole retrieves the role of a user.
func (s *UserServiceImpl) GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error) {
	role, err := s.UserDB.GetUserRole(ctx, discordID)
	if err != nil {
		return usertypes.UserRoleUnknown, fmt.Errorf("failed to get user role: %w", err)
	}
	return role, nil
}

// GetUser retrieves user data by Discord ID.
func (s *UserServiceImpl) GetUser(ctx context.Context, discordID usertypes.DiscordID) (usertypes.User, error) {
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}
