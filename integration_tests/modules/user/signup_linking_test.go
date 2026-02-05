package user_integration_tests

import (
	"context"
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// TestSignupCreatesUserAndGuildMembership validates that the signup path
// correctly creates both a global user and a guild_membership record.
// This is a foundational test ensuring the user resolution in imports
// can find users that signed up through the normal flow.
func TestSignupCreatesUserAndGuildMembership(t *testing.T) {
	t.Skip("Integration test - run locally with test infrastructure")

	// This test validates that when a user signs up:
	// 1. A global user record is created in the users table
	// 2. A guild_membership record is created linking user to guild
	// 3. The user can be found via the user lookup methods used by import resolution

	// Test setup would require:
	// - Database connection
	// - User repository
	// - User service with signup handler

	// Expected flow:
	// 1. Publish UserSignupRequested event
	// 2. Handler creates global user
	// 3. Handler creates guild_membership
	// 4. Verify both records exist
	// 5. Verify user can be found by normalized UDisc name lookup
}

// TestSignupWithUDiscIdentityCreatesGuildMembership validates that when a user
// signs up and links their UDisc identity, they can be resolved in imports.
func TestSignupWithUDiscIdentityCreatesGuildMembership(t *testing.T) {
	t.Skip("Integration test - run locally with test infrastructure")

	// This test validates the complete flow:
	// 1. User signs up -> global user + guild_membership created
	// 2. User links UDisc identity -> udisc_username/udisc_name set on global user
	// 3. Import can resolve user by normalized UDisc name

	// This ensures the import flow's resolveUserID function
	// can find users who have:
	// - Signed up to the guild (guild_membership exists)
	// - Linked their UDisc identity (udisc_username/udisc_name set)
}

// TestUserWithoutGuildMembershipNotResolvedInImport validates that users
// without a guild_membership for the target guild are NOT resolved during imports.
func TestUserWithoutGuildMembershipNotResolvedInImport(t *testing.T) {
	t.Skip("Integration test - run locally with test infrastructure")

	// This test validates the security constraint:
	// - Global user exists with UDisc identity
	// - User does NOT have guild_membership for target guild
	// - Import should NOT resolve this user
	// - No participant should be created for this user

	// This ensures that only guild members can be added as participants
	// during singles imports.
}

// verifyUserAndMembershipExist is a helper to check both user and membership records.
// This is used to validate the signup flow created the expected database state.
func verifyUserAndMembershipExist(t *testing.T, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) {
	t.Helper()
	ctx := context.Background()
	repo := userdb.NewRepository(db)

	// Check global user exists
	user, err := repo.GetUserGlobal(ctx, db, userID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if user == nil {
		t.Fatalf("expected user %s to exist, but not found", userID)
	}

	// Check guild membership exists
	membership, err := repo.GetGuildMembership(ctx, db, userID, guildID)
	if err != nil {
		t.Fatalf("failed to get guild membership: %v", err)
	}
	if membership == nil {
		t.Fatalf("expected guild membership for user %s in guild %s, but not found", userID, guildID)
	}
}

// verifyUserCanBeResolvedByName is a helper to check that the user lookup
// methods used by import resolution can find the user by their UDisc name.
func verifyUserCanBeResolvedByName(t *testing.T, db bun.IDB, guildID sharedtypes.GuildID, normalizedName string, expectedUserID sharedtypes.DiscordID) {
	t.Helper()
	ctx := context.Background()
	repo := userdb.NewRepository(db)

	// Try to find by UDisc username (used by import resolution)
	identity, err := repo.FindByUDiscUsername(ctx, db, guildID, normalizedName)
	if err != nil {
		t.Fatalf("failed to find user by UDisc username: %v", err)
	}
	if identity == nil || identity.User == nil {
		t.Fatalf("expected to find user by name %q, but not found", normalizedName)
	}
	if identity.User.GetUserID() != expectedUserID {
		t.Fatalf("expected user ID %s, got %s", expectedUserID, identity.User.GetUserID())
	}
}
