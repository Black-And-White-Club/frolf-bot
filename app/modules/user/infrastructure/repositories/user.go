package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/uptrace/bun"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// UserDB is a repository for user data operations.
type UserDBImpl struct {
	DB *bun.DB
}

// CreateUser creates a new user within a transaction.
func (db *UserDBImpl) CreateUser(ctx context.Context, user *User) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback is safe to call even if tx is committed

	_, err = tx.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateUserRole updates the role of an existing user within a transaction.
func (db *UserDBImpl) UpdateUserRole(ctx context.Context, discordID usertypes.DiscordID, role usertypes.UserRoleEnum) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback is safe to call even if tx is committed

	if !role.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}

	_, err = tx.NewUpdate().
		Model((*User)(nil)).
		Set("role = ?", role).
		Where("discord_id = ?", discordID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserByDiscordID retrieves a user by their Discord ID.
func (db *UserDBImpl) GetUserByDiscordID(ctx context.Context, discordID usertypes.DiscordID) (*User, error) {
	user := &User{}
	err := db.DB.NewSelect().Model(user).Where("discord_id = ?", discordID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound // Now returning a specific error
		}
		return nil, err
	}
	return user, nil
}

// GetUserRole retrieves the role of a user by their Discord ID.
func (db *UserDBImpl) GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error) {
	user := &User{}
	err := db.DB.NewSelect().
		Model(user).
		Column("role").
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotFound // Now returning a specific error
		}
		return "", err
	}
	return user.Role, nil
}
