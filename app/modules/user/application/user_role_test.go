package userservice

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	events "github.com/Black-And-White-Club/tcr-bot/app/events"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	usertypemocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types/mocks"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_OnUserRoleUpdateRequest(t *testing.T) {
	tests := []struct {
		name          string
		req           userevents.UserRoleUpdateRequestPayload
		mockUserDB    func(context.Context, *gomock.Controller, *userdb.MockUserDB)
		mockEventBus  func(context.Context, *gomock.Controller, *eventbusmock.MockEventBus)
		mockLogger    func(*testutils.MockLoggerAdapter)
		want          *userevents.UserRoleUpdateResponsePayload
		wantErr       error
		publishCalled bool // Add this to check if publishUserRoleUpdated is called
	}{
		{
			name: "Success",
			req: userevents.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(gomock.Any(), usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).Return(nil)
				mockUser := usertypemocks.NewMockUser(ctrl)
				mockDB.EXPECT().GetUserByDiscordID(gomock.Any(), usertypes.DiscordID("12345")).Return(mockUser, nil)

				// Add this expectation:
				mockUser.EXPECT().GetDiscordID().Return(usertypes.DiscordID("12345"))
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				mockEB.EXPECT().
					Publish(gomock.Any(), gomock.Eq(userevents.UserRoleUpdated), gomock.Any()).
					DoAndReturn(func(ctx context.Context, eventType events.EventType, msg types.Message) error {
						// Validate the message payload here (optional)
						return nil
					}).
					Times(1)
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: &userevents.UserRoleUpdateResponsePayload{
				Success: true,
			},
			wantErr:       nil,
			publishCalled: true,
		},
		{
			name: "Invalid Role",
			req: userevents.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleEnum("InvalidRole"),
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				// No expectations on mockUserDB as it should not be called
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// No expectations on mockEventBus as it should not be called
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				// Add expectations for error logging if needed
			},
			want:          nil,
			wantErr:       fmt.Errorf("invalid user role: %s", "InvalidRole"),
			publishCalled: false, // Expect publishUserRoleUpdated to NOT be called
		},
		{
			name: "Error Updating Role",
			req: userevents.UserRoleUpdateRequestPayload{
				DiscordID: "12345",
				NewRole:   usertypes.UserRoleAdmin,
			},
			mockUserDB: func(ctx context.Context, ctrl *gomock.Controller, mockDB *userdb.MockUserDB) {
				mockDB.EXPECT().UpdateUserRole(gomock.Any(), usertypes.DiscordID("12345"), usertypes.UserRoleAdmin).
					Return(errors.New("database error"))
			},
			mockEventBus: func(ctx context.Context, ctrl *gomock.Controller, mockEB *eventbusmock.MockEventBus) {
				// No expectations on mockEventBus as it should not be called
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				// Add expectations for error logging if needed
			},
			want:          nil,
			wantErr:       fmt.Errorf("failed to update user role: %w", errors.New("database error")),
			publishCalled: false, // Expect publishUserRoleUpdated to NOT be called
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserDB := userdb.NewMockUserDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			ctx := context.Background()

			if tt.mockUserDB != nil {
				tt.mockUserDB(ctx, ctrl, mockUserDB)
			}
			if tt.mockEventBus != nil {
				tt.mockEventBus(ctx, ctrl, mockEventBus)
			}
			if tt.mockLogger != nil {
				tt.mockLogger(mockLogger)
			}

			service := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   mockLogger,
			}

			got, err := service.OnUserRoleUpdateRequest(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.wantErr == nil {
					t.Errorf("Expected no error, but got: %v", err)
				} else if err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expected result: %v, got: %v", tt.want, got)
			}
		})
	}
}

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockUserDB := userdb.NewMockUserDB(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)

	s := &UserServiceImpl{
		eventBus: mockEventBus,
		UserDB:   mockUserDB,
		logger:   mockLogger,
	}

	tests := []struct {
		name         string
		discordID    usertypes.DiscordID
		mockUserDB   func(ctx context.Context, mockUserDB *userdb.MockUserDB)
		mockEventBus func(ctx context.Context, mockEventBus *eventbusmock.MockEventBus)
		mockLogger   func(mockLogger *testutils.MockLoggerAdapter)
		want         usertypes.UserRoleEnum // Change to interface type
		wantErr      bool
		expectedErr  error
	}{
		{
			name:      "Success",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("12345")).Return(usertypes.UserRoleAdmin, nil)
			},
			mockEventBus: func(ctx context.Context, mockEventBus *eventbusmock.MockEventBus) {
				// No expectations on the event bus
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
				// No expectations on the logger
			},
			want:    usertypes.UserRoleAdmin, // Use the constant directly
			wantErr: false,
		},
		{
			name:      "GetUser Role Error",
			discordID: "12345",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("12345")).Return(usertypes.UserRoleUnknown, errors.New("database error")) // Return UserRoleUnknown
			},
			mockEventBus: func(ctx context.Context, mockEventBus *eventbusmock.MockEventBus) {
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
			},
			want:        usertypes.UserRoleUnknown, // Use UserRoleUnknown here
			wantErr:     true,
			expectedErr: fmt.Errorf("failed to get user role: %w", errors.New("database error")),
		},
		{
			name:      "User Not Found",
			discordID: "nonexistent",
			mockUserDB: func(ctx context.Context, mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().GetUserRole(ctx, usertypes.DiscordID("nonexistent")).Return(usertypes.UserRoleUnknown, errors.New("user not found")) // Return UserRoleUnknown
			},
			mockEventBus: func(ctx context.Context, mockEventBus *eventbusmock.MockEventBus) {
			},
			mockLogger: func(mockLogger *testutils.MockLoggerAdapter) {
			},
			want:        usertypes.UserRoleUnknown, // Use UserRoleUnknown here
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
			if tt.mockEventBus != nil {
				tt.mockEventBus(ctx, mockEventBus)
			}
			if tt.mockLogger != nil {
				tt.mockLogger(mockLogger)
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

	mockUserDB := userdb.NewMockUserDB(ctrl) // Create a mock for the UserDB interface
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)

	s := &UserServiceImpl{
		eventBus: mockEventBus,
		UserDB:   mockUserDB,
		logger:   mockLogger,
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
				expectedUser := usertypemocks.NewMockUser(ctrl) // Create a new mock user
				mockUserDB.EXPECT().GetUserByDiscordID(ctx, usertypes.DiscordID("12345")).Return(expectedUser, nil)
			},
			want:    usertypemocks.NewMockUser(ctrl), // Return a new mock user
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

func TestUserServiceImpl_publishUserRoleUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		discordID    usertypes.DiscordID
		newRole      usertypes.UserRoleEnum
		mockUserDB   func(mockUserDB *userdb.MockUserDB)
		mockEventBus func(mockEventBus *eventbusmock.MockEventBus)
		wantErr      bool
	}{
		{
			name:      "Happy Path",
			discordID: "1234567890",
			newRole:   usertypes.UserRoleAdmin,
			mockUserDB: func(mockUserDB *userdb.MockUserDB) {
				expectedUser := usertypemocks.NewMockUser(ctrl)
				expectedUser.EXPECT().GetDiscordID().Return(usertypes.DiscordID("1234567890"))

				mockUserDB.EXPECT().
					GetUserByDiscordID(gomock.Any(), usertypes.DiscordID("1234567890")).
					Return(expectedUser, nil)
			},
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.UserRoleUpdated, gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "Error Getting User",
			discordID: "nonexistent",
			newRole:   usertypes.UserRoleEditor,
			mockUserDB: func(mockUserDB *userdb.MockUserDB) {
				mockUserDB.EXPECT().
					GetUserByDiscordID(gomock.Any(), usertypes.DiscordID("nonexistent")).
					Return(nil, fmt.Errorf("user not found"))
			},
			wantErr: true,
		},
		{
			name:      "Error Marshaling Event",
			discordID: "1234567890",
			newRole:   usertypes.UserRoleAdmin,
			mockUserDB: func(mockUserDB *userdb.MockUserDB) {
				expectedUser := usertypemocks.NewMockUser(ctrl)
				expectedUser.EXPECT().GetDiscordID().Return(usertypes.DiscordID("1234567890"))

				mockUserDB.EXPECT().
					GetUserByDiscordID(gomock.Any(), usertypes.DiscordID("1234567890")).
					Return(expectedUser, nil)
			},
			mockEventBus: func(mockEventBus *eventbusmock.MockEventBus) {
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.UserRoleUpdated, gomock.Any()).
					Return(fmt.Errorf("failed to publish"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserDB := userdb.NewMockUserDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			if tt.mockUserDB != nil {
				tt.mockUserDB(mockUserDB)
			}
			if tt.mockEventBus != nil {
				tt.mockEventBus(mockEventBus)
			}

			s := &UserServiceImpl{
				UserDB:   mockUserDB,
				eventBus: mockEventBus,
				logger:   mockLogger,
			}

			if err := s.publishUserRoleUpdated(context.Background(), tt.discordID, tt.newRole); (err != nil) != tt.wantErr {
				t.Errorf("User ServiceImpl.publishUser RoleUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
