package userdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/uptrace/bun"
)

// UserDBImpl is an implementation of the UserDB interface using bun.
type UserDBImpl struct {
	DB *bun.DB
}

func (db *UserDBImpl) CreateUser(ctx context.Context, user *User) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	fmt.Println("Starting user insertion:", user)

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		fmt.Println("Transaction start failed:", err)
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	_, err = tx.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		fmt.Println("Insert query failed:", err)
		_ = tx.Rollback()
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Println("Insert query successful. Committing transaction.")

	if err := tx.Commit(); err != nil {
		fmt.Println("Transaction commit failed:", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Println("User insertion and commit successful!")
	return nil
}

// GetUserByDiscordID retrieves a user by Discord ID.
func (db *UserDBImpl) GetUserByDiscordID(ctx context.Context, discordID usertypes.DiscordID) (*User, error) {
	var dbUser User

	err := db.DB.NewSelect().
		Model(&dbUser).
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User not found, return nil without an error
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &dbUser, nil
}

// GetUserRole retrieves the role of a user by their Discord ID.
func (db *UserDBImpl) GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error) {
	var user User
	err := db.DB.NewSelect().
		Model(&user).
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		// Use UserRoleRattler as the default/fallback role
		return usertypes.UserRoleRattler, fmt.Errorf("failed to get user role: %w", err)
	}
	return user.Role, nil
}

// UpdateUserRole updates the role of a user by their Discord ID.
func (db *UserDBImpl) UpdateUserRole(ctx context.Context, discordID usertypes.DiscordID, newRole usertypes.UserRoleEnum) error {
	// Use newRole.IsValid() from the interface
	if !newRole.IsValid() {
		return fmt.Errorf("invalid user role: %s", newRole)
	}
	_, err := db.DB.NewUpdate().
		Model(&User{}).
		Set("role = ?", newRole). // Use newRole directly
		Where("discord_id = ?", discordID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}
