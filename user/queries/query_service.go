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
	eventBus watermillcmd.MessageBus // Add an eventBus field
}

func (s *UserQueryService) EventBus() watermillcmd.MessageBus {
	return s.eventBus
}

// NewUserQueryService creates a new UserQueryService.
func NewUserQueryService(userDB userdb.UserDB, eventBus watermillcmd.MessageBus) QueryService { // Add eventBus to the constructor
	return &UserQueryService{
		userDB:   userDB,
		eventBus: eventBus,
	}
}

// GetUserByID retrieves a user by their ID.
func (s *UserQueryService) GetUserByDiscordID(ctx context.Context, discordID string) (*userdb.User, error) {
	user, err := s.userDB.GetUserByDiscordID(ctx, discordID)
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
	user, err := s.userDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	role := string(user.Role)

	// Publish UserRoleResponseEvent
	if err := s.eventBus.PublishEvent(ctx, "get-user-role-topic", userhandlers.UserRoleResponseEvent{
		Role: role,
	}); err != nil {
		// Handle the error (e.g., log it)
	}

	return role, nil
}
