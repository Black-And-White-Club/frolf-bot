package adapters

import (
	"context"
	"errors"
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestUserLookupAdapter_FindByNormalizedUDiscUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := usermocks.NewMockRepository(ctrl)
	adapter := NewUserLookupAdapter(mockUserDB)

	testGuildID := sharedtypes.GuildID("guild-123")
	testUsername := "testuser"
	testUserID := sharedtypes.DiscordID("user-123")

	t.Run("User found", func(t *testing.T) {
		mockUserDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(&userdb.UserWithMembership{
				User: &userdb.User{UserID: testUserID},
				Role: sharedtypes.UserRoleUser,
			}, nil)

		result, err := adapter.FindByNormalizedUDiscUsername(context.Background(), testGuildID, testUsername)
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
		mockUserDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(nil, userdb.ErrNotFound)

		result, err := adapter.FindByNormalizedUDiscUsername(context.Background(), testGuildID, testUsername)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result, got non-nil")
		}
	})

	t.Run("DB error", func(t *testing.T) {
		mockUserDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(nil, errors.New("db error"))

		_, err := adapter.FindByNormalizedUDiscUsername(context.Background(), testGuildID, testUsername)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestUserLookupAdapter_FindByNormalizedUDiscDisplayName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := usermocks.NewMockRepository(ctrl)
	adapter := NewUserLookupAdapter(mockUserDB)

	testGuildID := sharedtypes.GuildID("guild-123")
	testDisplayName := "Test User"
	testUserID := sharedtypes.DiscordID("user-123")

	t.Run("User found", func(t *testing.T) {
		mockUserDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testDisplayName).
			Return(&userdb.UserWithMembership{
				User: &userdb.User{UserID: testUserID},
				Role: sharedtypes.UserRoleUser,
			}, nil)

		result, err := adapter.FindByNormalizedUDiscDisplayName(context.Background(), testGuildID, testDisplayName)
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
		mockUserDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testDisplayName).
			Return(nil, userdb.ErrNotFound)

		result, err := adapter.FindByNormalizedUDiscDisplayName(context.Background(), testGuildID, testDisplayName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil result, got non-nil")
		}
	})

	t.Run("DB error", func(t *testing.T) {
		mockUserDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testDisplayName).
			Return(nil, errors.New("db error"))

		_, err := adapter.FindByNormalizedUDiscDisplayName(context.Background(), testGuildID, testDisplayName)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
