package testutils

import (
	"context"
	"fmt"
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// --- New helpers for User module ---

// InsertUser creates and inserts a user directly into the database.
// This bypasses the service layer to set up specific database preconditions for tests.
func InsertUser(t *testing.T, db *bun.DB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	t.Helper()
	user := &userdb.User{ // Use the actual userdb.User model
		UserID:  userID,
		GuildID: guildID,
		Role:    role,
	}
	// If a role is not explicitly provided, use the default from the DB model or a sensible test default
	if role == "" {
		user.Role = sharedtypes.UserRoleUser // Use the constant from sharedtypes
	}
	_, err := db.NewInsert().Model(user).Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to insert user %s with guild %s and role %s: %w", userID, guildID, role, err)
	}
	return nil
}
