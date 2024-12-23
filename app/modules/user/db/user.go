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
func (db *UserDBImpl) CreateUser(ctx context.Context, user *User) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
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

// GetUserRole retrieves the role of a user by their Discord ID.
func (db *UserDBImpl) GetUserRole(ctx context.Context, discordID string) (UserRole, error) {
	var role UserRole
	err := db.DB.NewSelect().
		Model((*User)(nil)). // Select from the User model without instantiating a User object
		Column("role").      // Select only the "role" column
		Where("discord_id = ?", discordID).
		Scan(ctx, &role) // Scan the result directly into the role variable
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}
	return role, nil
}

// UpdateUserRole updates the role of a user by their Discord ID.
func (db *UserDBImpl) UpdateUserRole(ctx context.Context, discordID string, newRole UserRole) error {
	if !newRole.IsValid() {
		return fmt.Errorf("invalid user role: %s", newRole)
	}

	_, err := db.DB.NewUpdate().
		Model(&User{}).
		Set("role = ?", newRole).
		Where("discord_id = ?", discordID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}
