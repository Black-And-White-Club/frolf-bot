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
// It creates both the global User and the GuildMembership.
func InsertUser(t *testing.T, db *bun.DB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error {
	t.Helper()

	// Create global user if not exists
	user := &userdb.User{
		UserID: &userID,
	}
	// Insert or ignore if user already exists
	_, err := db.NewInsert().Model(user).On("CONFLICT (user_id) DO NOTHING").Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to insert global user %s: %w", userID, err)
	}

	// Create guild membership
	if role == "" {
		role = sharedtypes.UserRoleUser
	}
	membership := &userdb.GuildMembership{
		UserID:  userID,
		GuildID: guildID,
		Role:    role,
	}
	_, err = db.NewInsert().Model(membership).Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to insert guild membership for user %s in guild %s: %w", userID, guildID, err)
	}
	return nil
}
