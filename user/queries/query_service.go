// user/queries/query_service.go
package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/user/db"
)

// UserQueryService implements the QueryService interface.
type UserQueryService struct {
	userDB db.UserDB
}

// NewUserQueryService creates a new UserQueryService.
func NewUserQueryService(userDB db.UserDB) QueryService {
	return &UserQueryService{
		userDB: userDB,
	}
}

// GetUserByID retrieves a user by their ID.
func (s *UserQueryService) GetUserByID(ctx context.Context, discordID string) (*db.User, error) {
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
	return string(user.Role), nil
}
