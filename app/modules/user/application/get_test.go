package userservice

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	adaptermock "github.com/Black-And-White-Club/tcr-bot/app/adapters/mocks"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	usertypemocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types/mocks"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)
	mockEventAdapter := new(adaptermock.MockEventAdapterInterface)

	s := &UserServiceImpl{
		eventBus:     mockEventBus,
		UserDB:       mockUserDB,
		logger:       mockLogger,
		eventAdapter: mockEventAdapter,
	}

	tests := []struct {
		name        string
		discordID   usertypes.DiscordID
		mockUserDB  func(ctx context.Context, mockUserDB *userdb.MockUserDB)
		want        usertypes.UserRoleEnum
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "Success",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("12345")).Return(usertypes.UserRoleAdmin, nil)
			},
			want:    usertypes.UserRoleAdmin,
			wantErr: false,
		},
		{
			name:      "GetUser Role Error",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("12345")).Return(usertypes.UserRoleUnknown, errors.New("database error"))
			},
			want:        usertypes.UserRoleUnknown,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user role: %w", errors.New("database error")),
		},
		{
			name:      "User Not Found",
			discordID: "nonexistent",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("nonexistent")).Return(usertypes.UserRoleUnknown, errors.New("user not found"))
			},
			want:        usertypes.UserRoleUnknown,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user role: %w", errors.New("user not found")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, mockUserDB)
			}

			got, err := s.GetUserRole(ctx, tt.discordID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser Role() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUser Role() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestUserServiceImpl_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)
	mockEventAdapter := new(adaptermock.MockEventAdapterInterface)

	s := &UserServiceImpl{
		eventBus:     mockEventBus,
		UserDB:       mockUserDB,
		logger:       mockLogger,
		eventAdapter: mockEventAdapter, // Add the mockEventAdapter
	}

	tests := []struct {
		name        string
		discordID   usertypes.DiscordID
		mockUserDB  func(ctx context.Context, mockUserDB *userdb.MockUserDB)
		want        usertypes.User
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "Success",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				expectedUser := usertypemocks.NewMockUser(ctrl)
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(expectedUser, nil)
			},
			want:    usertypemocks.NewMockUser(ctrl),
			wantErr: false,
		},
		{
			name:      "User  Not Found",
			discordID: "nonexistent",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("nonexistent")).Return(nil, errors.New("user not found"))
			},
			want:        nil,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user: %w", errors.New("user not found")),
		},
		{
			name:      "Database Error",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(nil, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user: %w", errors.New("database error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, mockUserDB)
			}

			got, err := s.GetUser(ctx, tt.discordID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser () error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				t.Errorf("Expected error: %v, Got: %v", tt.expectedErr, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUser () = %v, want %v", got, tt.want)
			}
		})
	}
}
