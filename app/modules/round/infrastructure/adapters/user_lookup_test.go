package adapters

import (
	"context"
	"errors"
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func TestUserLookupAdapter_FindByNormalizedUDiscUsername(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-123")
	testUsername := "testuser"
	testUserID := sharedtypes.DiscordID("user-123")

	t.Run("User found", func(t *testing.T) {
		fakeRepo := &userdb.FakeRepository{
			FindByUDiscUsernameFn: func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*userdb.UserWithMembership, error) {
				if guildID != testGuildID || username != testUsername {
					return nil, errors.New("unexpected arguments")
				}
				return &userdb.UserWithMembership{
					User: &userdb.User{UserID: &testUserID},
					Role: sharedtypes.UserRoleUser,
				}, nil
			},
		}
		adapter := NewUserLookupAdapter(fakeRepo, nil)

		result, err := adapter.FindByNormalizedUDiscUsername(context.Background(), nil, testGuildID, testUsername)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected result, got nil")
		} else if result.UserID != testUserID {
			t.Errorf("expected user ID %s, got %s", testUserID, result.UserID)
		}
	})

	t.Run("User not found", func(t *testing.T) {
		fakeRepo := &userdb.FakeRepository{
			FindByUDiscUsernameFn: func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*userdb.UserWithMembership, error) {
				return nil, userdb.ErrNotFound
			},
		}
		adapter := NewUserLookupAdapter(fakeRepo, nil)

		result, err := adapter.FindByNormalizedUDiscUsername(context.Background(), nil, testGuildID, testUsername)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result, got non-nil")
		}
	})

	t.Run("DB error", func(t *testing.T) {
		fakeRepo := &userdb.FakeRepository{
			FindByUDiscUsernameFn: func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*userdb.UserWithMembership, error) {
				return nil, errors.New("db error")
			},
		}
		adapter := NewUserLookupAdapter(fakeRepo, nil)

		_, err := adapter.FindByNormalizedUDiscUsername(context.Background(), nil, testGuildID, testUsername)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestUserLookupAdapter_FindByNormalizedUDiscDisplayName(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-123")
	testDisplayName := "Test User"
	testUserID := sharedtypes.DiscordID("user-123")

	t.Run("User found", func(t *testing.T) {
		fakeRepo := &userdb.FakeRepository{
			FindByUDiscNameFn: func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*userdb.UserWithMembership, error) {
				if guildID != testGuildID || name != testDisplayName {
					return nil, errors.New("unexpected arguments")
				}
				return &userdb.UserWithMembership{
					User: &userdb.User{UserID: &testUserID},
					Role: sharedtypes.UserRoleUser,
				}, nil
			},
		}
		adapter := NewUserLookupAdapter(fakeRepo, nil)

		result, err := adapter.FindByNormalizedUDiscDisplayName(context.Background(), nil, testGuildID, testDisplayName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected result, got nil")
		} else if result.UserID != testUserID {
			t.Errorf("expected user ID %s, got %s", testUserID, result.UserID)
		}
	})

	t.Run("User not found", func(t *testing.T) {
		fakeRepo := &userdb.FakeRepository{
			FindByUDiscNameFn: func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*userdb.UserWithMembership, error) {
				return nil, userdb.ErrNotFound
			},
		}
		adapter := NewUserLookupAdapter(fakeRepo, nil)

		result, err := adapter.FindByNormalizedUDiscDisplayName(context.Background(), nil, testGuildID, testDisplayName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result, got non-nil")
		}
	})

	t.Run("DB error", func(t *testing.T) {
		fakeRepo := &userdb.FakeRepository{
			FindByUDiscNameFn: func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*userdb.UserWithMembership, error) {
				return nil, errors.New("db error")
			},
		}
		adapter := NewUserLookupAdapter(fakeRepo, nil)

		_, err := adapter.FindByNormalizedUDiscDisplayName(context.Background(), nil, testGuildID, testDisplayName)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
