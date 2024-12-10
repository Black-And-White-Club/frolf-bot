package userqueries

import (
	"context"
	"errors"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/user/db"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/watermillcmd/user"
)

// UserQueryService implements the QueryService interface.
type UserQueryService struct {
	userDB   userdb.UserDB
	eventBus watermillcmd.EventBus // Add an eventBus field
}

func (s *UserQueryService) EventBus() watermillcmd.EventBus {
	return s.eventBus
}

// NewUserQueryService creates a new UserQueryService.
func NewUserQueryService(userDB userdb.UserDB, eventBus watermillcmd.EventBus) QueryService { // Add eventBus to the constructor
	return &UserQueryService{
		userDB:   userDB,
		eventBus: eventBus,
	}
}

// GetUserByID retrieves a user by their ID.
func (s *UserQueryService) GetUserByID(ctx context.Context, discordID string) (*userdb.User, error) {
	user, err := s.userDB.GetUser(ctx, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// GetUserRole retrieves the role of a user.
func (s *UserQueryService) GetUserRole(ctx context.Context, discordID string) (string, error) {
	user, err := s.userDB.GetUser(ctx, discordID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	role := string(user.Role)

	// Publish UserRoleResponseEvent
	if err := s.eventBus.Publish(ctx, userhandlers.UserRoleResponseEvent{
		Role: role,
	}); err != nil {
		// Handle the error (e.g., log it)
	}

	return role, nil
}
