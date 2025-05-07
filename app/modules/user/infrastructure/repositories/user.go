package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

var ErrUserNotFound = errors.New("user not found")

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
	defer tx.Rollback()

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
func (db *UserDBImpl) UpdateUserRole(ctx context.Context, userID sharedtypes.DiscordID, role sharedtypes.UserRoleEnum) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback is safe to call even if tx is committed

	if !role.IsValid() {
		return fmt.Errorf("invalid user role: %s", role)
	}

	// Execute the update query and get the result
	result, err := tx.NewUpdate().
		Model((*User)(nil)).
		Set("role = ?", role).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		// Handle errors during query execution
		return fmt.Errorf("failed to execute update user role query: %w", err)
	}

	// Check the number of rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Handle error getting rows affected
		return fmt.Errorf("failed to get rows affected after update: %w", err)
	}

	// If no rows were affected, the user was not found
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserByUserID retrieves a user by their Discord ID.
func (db *UserDBImpl) GetUserByUserID(ctx context.Context, userID sharedtypes.DiscordID) (*User, error) {
	user := &User{}
	err := db.DB.NewSelect().Model(user).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound // Now returning a specific error
		}
		return nil, err
	}
	return user, nil
}

// GetUserRole retrieves the role of a user by their Discord ID.
func (db *UserDBImpl) GetUserRole(ctx context.Context, userID sharedtypes.DiscordID) (sharedtypes.UserRoleEnum, error) {
	user := &User{}
	err := db.DB.NewSelect().
		Model(user).
		Column("role").
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrUserNotFound // Now returning a specific error
		}
		return "", err
	}
	return user.Role, nil
}
