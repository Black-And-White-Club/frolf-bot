// user/db/user.go
package userdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/uptrace/bun"
)

// userDBImpl is an implementation of the UserDB interface using bun.
type userDBImpl struct {
	db *bun.DB
}

// NewUserDB creates a new userDBImpl.
func NewUserDB(db *bun.DB) UserDB {
	return &userDBImpl{db: db}
}

// CreateUser creates a new user.
func (db *userDBImpl) CreateUser(ctx context.Context, user *User) error {
	if db.db == nil {
		return errors.New("database connection is not initialized")
	}

	if user == nil {
		return errors.New("user cannot be nil")
	}

	_, err := db.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by Discord ID.
func (db *userDBImpl) GetUser(ctx context.Context, discordID string) (*User, error) {
	var user User
	err := db.db.NewSelect().
		Model(&user).
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// UpdateUser updates an existing user.
func (db *userDBImpl) UpdateUser(ctx context.Context, discordID string, updates *User) error {
	_, err := db.db.NewUpdate().
		Model(updates).
		Column("name", "role").
		Where("discord_id = ?", discordID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}
