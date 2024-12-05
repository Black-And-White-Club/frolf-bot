// internal/db/bundb/user.go

package bundb

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/uptrace/bun"
)

// userDB is an implementation of the UserDB interface using bun.
type userDB struct {
	db *bun.DB
}

func (db *userDB) Ping(ctx context.Context) error {
	return db.db.Ping() // Assuming db.db is your *bun.DB instance
}

// GetUser retrieves a user by Discord ID.
func (db *userDB) GetUser(ctx context.Context, discordID string) (*models.User, error) {
	log.Printf("userDB.GetUser - Trying to fetch user with discordID: %s", discordID) // Logging added

	var user models.User
	err := db.db.NewSelect().
		Model(&user).
		Where("discord_id = ?", discordID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (db *userDB) CreateUser(ctx context.Context, user *models.User) error {
	if db.db == nil {
		log.Println("userDB.db is nil")
		return errors.New("database connection is not initialized")
	}

	if user == nil {
		log.Println("CreateUser - user is nil")
		return errors.New("user cannot be nil")
	}

	log.Printf("CreateUser - Received user: %+v", user)
	_, err := db.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		log.Printf("CreateUser - Error creating user: %v", err)
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// UpdateUser updates an existing user.
func (db *userDB) UpdateUser(ctx context.Context, discordID string, updates *models.User) error {
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
