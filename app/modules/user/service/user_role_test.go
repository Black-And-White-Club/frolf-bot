package userservice

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"testing"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	user_mocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/mock/gomock"
)

// setupMocks is a helper function to set up mocks for testing.
func setupMocks(_ *testing.T, mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher, mockSubscriber *testutils.MockSubscriber) (Service, error) { // Replace YourServiceType
	// Create the service with the mocks
	service := NewUserService(mockPublisher, mockSubscriber, mockDB, watermill.NopLogger{})

	return service, nil // Or return an error if necessary
}

func TestUserServiceImpl_OnUserRoleUpdateRequest(t *testing.T) {
	funcName := runtime.FuncForPC(reflect.ValueOf(TestUserServiceImpl_OnUserRoleUpdateRequest).Pointer()).Name()
	fmt.Printf("Currently running: %s\n", funcName)

	tests := []struct {
		name          string
		req           userevents.UserRoleUpdateRequest
		setupMocks    func(*user_mocks.MockUserDB, *testutils.MockPublisher)
		expectedResp  *userevents.UserRoleUpdateResponse
		expectedError bool
	}{
		{
			name: "Success",
			req:  userevents.UserRoleUpdateRequest{DiscordID: "12345", NewRole: userdb.UserRoleAdmin},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher) {
				ctx := context.Background()
				mockDB.EXPECT().UpdateUserRole(ctx, "12345", userdb.UserRoleAdmin).Return(nil)

				// Set up the mock Publisher to expect a Publish call with any arguments
				mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedResp: &userevents.UserRoleUpdateResponse{Success: true},
		},
		{
			name:          "Invalid Role",
			req:           userevents.UserRoleUpdateRequest{DiscordID: "12345", NewRole: "InvalidRole"},
			expectedError: true,
		},
		{
			name: "UpdateUserRole Error",
			req:  userevents.UserRoleUpdateRequest{DiscordID: "12345", NewRole: userdb.UserRoleAdmin},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher) {
				ctx := context.Background()
				mockDB.EXPECT().UpdateUserRole(ctx, "12345", userdb.UserRoleAdmin).Return(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "Publish Event Error",
			req:  userevents.UserRoleUpdateRequest{DiscordID: "12345", NewRole: userdb.UserRoleAdmin},
			setupMocks: func(mockDB *user_mocks.MockUserDB, mockPublisher *testutils.MockPublisher) {
				ctx := context.Background()
				mockDB.EXPECT().UpdateUserRole(ctx, "12345", userdb.UserRoleAdmin).Return(nil)
				mockPublisher.EXPECT().Publish(userevents.UserRoleUpdatedSubject, gomock.Any()).Return(errors.New("publish error"))
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := user_mocks.NewMockUserDB(ctrl)
			mockPublisher := testutils.NewMockPublisher(ctrl)
			mockSubscriber := testutils.NewMockSubscriber(ctrl)

			service, err := setupMocks(t, mockDB, mockPublisher, mockSubscriber)
			if err != nil {
				t.Fatalf("Failed to setup mocks: %v", err)
			}

			if tc.setupMocks != nil {
				tc.setupMocks(mockDB, mockPublisher)
			}

			ctx := context.Background()
			resp, err := service.OnUserRoleUpdateRequest(ctx, tc.req)

			if tc.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(resp, tc.expectedResp) {
					t.Errorf("Expected response: %+v, got: %+v", tc.expectedResp, resp)
				}
			}
		})
	}
}

func TestUserServiceImpl_GetUserRole(t *testing.T) {
	tests := []struct {
		name          string
		discordID     string
		expectedRole  userdb.UserRole
		expectedError bool
	}{
		{
			name:         "Success",
			discordID:    "12345",
			expectedRole: userdb.UserRoleAdmin,
		},
		{
			name:          "GetUserRole Error",
			discordID:     "12345",
			expectedError: true,
		},
		{
			name:          "User Not Found",
			discordID:     "nonexistent",
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := user_mocks.NewMockUserDB(ctrl)
			service := NewUserService(nil, nil, mockDB, nil)
			ctx := context.Background()

			// Set expectations on mock based on test case
			switch tc.name {
			case "Success":
				mockDB.EXPECT().GetUserRole(ctx, tc.discordID).Return(tc.expectedRole, nil)
			case "GetUserRole Error":
				mockDB.EXPECT().GetUserRole(ctx, tc.discordID).Return(userdb.UserRole(""), errors.New("database error"))
			case "User Not Found":
				mockDB.EXPECT().GetUserRole(ctx, tc.discordID).Return(userdb.UserRole(""), errors.New("user not found"))
			}

			role, err := service.GetUserRole(ctx, tc.discordID)

			if tc.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if role != nil { // Check for nil before dereferencing
					t.Errorf("Expected nil role for error case, got: %v", *role)
				}

			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if role == nil {
					t.Error("Expected role, got nil")
				} else if *role != tc.expectedRole { // Dereference role here
					t.Errorf("Expected role: %v, got: %v", tc.expectedRole, *role)
				}
			}
		})
	}
}

func TestUserServiceImpl_GetUser(t *testing.T) {
	tests := []struct {
		name          string
		discordID     string
		expectedUser  *userdb.User
		expectedError bool
	}{
		{
			name:         "Success",
			discordID:    "12345",
			expectedUser: &userdb.User{DiscordID: "12345", Role: userdb.UserRoleAdmin},
		},
		{
			name:          "User Not Found",
			discordID:     "nonexistent",
			expectedError: true,
		},
		{
			name:          "Database Error",
			discordID:     "12345",
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := user_mocks.NewMockUserDB(ctrl)
			service := NewUserService(nil, nil, mockDB, nil)
			ctx := context.Background()

			// Set expectations on mock based on test case
			switch tc.name {
			case "Success":
				expectedUser := &userdb.User{DiscordID: "12345", Role: userdb.UserRoleAdmin}
				mockDB.EXPECT().GetUserByDiscordID(ctx, tc.discordID).Return(expectedUser, nil)
			case "User Not Found":
				mockDB.EXPECT().GetUserByDiscordID(ctx, tc.discordID).Return((*userdb.User)(nil), errors.New("user not found"))
			case "Database Error":
				mockDB.EXPECT().GetUserByDiscordID(ctx, tc.discordID).Return((*userdb.User)(nil), errors.New("database error"))
			}

			user, err := service.GetUser(ctx, tc.discordID)

			if tc.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if user != nil {
					t.Error("Expected nil user for error case, got a user")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(user, tc.expectedUser) {
					t.Errorf("Expected user: %+v, got: %+v", tc.expectedUser, user)
				}
			}
		})
	}
}

func TestUserServiceImpl_publishUserRoleUpdated(t *testing.T) {
	tests := []struct {
		name          string
		discordID     string
		newRole       userdb.UserRole
		setupMocks    func(*testutils.MockPublisher)
		expectedError bool
	}{
		{
			name:      "Success",
			discordID: "12345",
			newRole:   userdb.UserRoleAdmin,
			setupMocks: func(mockPublisher *testutils.MockPublisher) {
				// Use gomock.Any() to match any message
				mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:      "Publish Error",
			discordID: "12345",
			newRole:   userdb.UserRoleAdmin,
			setupMocks: func(mockPublisher *testutils.MockPublisher) {
				// Use gomock.Any() to match any message
				mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(errors.New("publish error"))
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockPublisher := testutils.NewMockPublisher(ctrl)
			serviceImpl := &UserServiceImpl{
				Publisher: mockPublisher,
			}

			// Call setupMocks to set the expectations
			tc.setupMocks(mockPublisher)

			err := serviceImpl.publishUserRoleUpdated(tc.discordID, tc.newRole)

			if tc.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
