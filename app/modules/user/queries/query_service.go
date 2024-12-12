package userqueries

import (
	"context"
	"errors"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
)

// UserQueryService implements the QueryService interface.
type UserQueryService struct {
	UserDB   userdb.UserDB
	eventBus *watermillutil.PubSub // Use your PubSub struct
}

func (s *UserQueryService) EventBus() *watermillutil.PubSub {
	return s.eventBus
}

// NewUserQueryService creates a new UserQueryService.
func NewUserQueryService(userDB userdb.UserDB, eventBus *watermillutil.PubSub) QueryService {
	return &UserQueryService{
		UserDB:   userDB,
		eventBus: eventBus,
	}
}

// GetUserByID retrieves a user by their ID.
func (s *UserQueryService) GetUserByDiscordID(ctx context.Context, discordID string) (*userdb.User, error) {
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
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
	user, err := s.UserDB.GetUserByDiscordID(ctx, discordID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	return string(user.Role), nil // Just return the role, no need to publish an event here
}
