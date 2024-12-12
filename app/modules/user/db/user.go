package userdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

// UserDBImpl is an implementation of the UserDB interface using bun.
type UserDBImpl struct {
	DB *bun.DB
}

// CreateUser creates a new user.
func (db *UserDBImpl) CreateUser(ctx context.Context, discordID string, name string, role UserRole) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	user := &User{
		DiscordID: discordID,
		Name:      name,
		Role:      role,
	}

	_, err := db.DB.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByDiscordID retrieves a user by Discord ID.
func (db *UserDBImpl) GetUserByDiscordID(ctx context.Context, discordID string) (*User, error) {
	var user User
	err := db.DB.NewSelect().
		Model(&user).
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user.
func (db *UserDBImpl) UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error {
	query := db.DB.NewUpdate().Model(&User{}).Where("discord_id = ?", discordID)

	for column, value := range updates {
		query = query.Set(fmt.Sprintf("%s = ?", column), value)
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// GetUserRole retrieves the role of a user.
func (db *UserDBImpl) GetUserRole(ctx context.Context, discordID string) (UserRole, error) {
	var user User
	err := db.DB.NewSelect().
		Model(&user).
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return user.Role, nil
}
