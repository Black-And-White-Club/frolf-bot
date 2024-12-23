package userhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	user_mocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := user_mocks.NewMockService(ctrl)
	mockPublisher := testutils.NewMockPublisher(ctrl)
	mockLogger := testutils.NewMockLogger(ctrl)

	handlers := NewHandlers(mockUserService, mockPublisher, mockLogger)

	if handlers == nil {
		t.Error("NewHandlers() returned nil")
	}
}

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	tests := []struct {
		name          string
		message       *message.Message
		setupMocks    func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger)
		expectedError error
	}{
		{
			name: "Success",
			message: func() *message.Message {
				req := userevents.UserSignupRequest{DiscordID: "123", TagNumber: 1}
				payload, _ := json.Marshal(req)
				return message.NewMessage(watermill.NewUUID(), payload)
			}(),
			setupMocks: func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger) {
				resp := &userevents.UserSignupResponse{Success: true}
				mockUserService.EXPECT().OnUserSignupRequest(gomock.Any(), gomock.Any()).Return(resp, nil)
				mockPublisher.EXPECT().Publish(userevents.UserSignupResponseSubject, gomock.Any()).Return(nil)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: nil,
		},
		{
			name: "Invalid Payload",
			message: func() *message.Message {
				return message.NewMessage(watermill.NewUUID(), []byte("invalid json"))
			}(),
			setupMocks: func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger) {
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: errors.New("failed to unmarshal UserSignupRequest"),
		},
		{
			name: "Service Error",
			message: func() *message.Message {
				req := userevents.UserSignupRequest{DiscordID: "123", TagNumber: 1}
				payload, _ := json.Marshal(req)
				return message.NewMessage(watermill.NewUUID(), payload)
			}(),
			setupMocks: func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger) {
				mockUserService.EXPECT().OnUserSignupRequest(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: errors.New("failed to process user signup request: service error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := user_mocks.NewMockService(ctrl)
			mockPublisher := testutils.NewMockPublisher(ctrl)
			mockLogger := testutils.NewMockLogger(ctrl) // Create mock logger

			handlers := NewHandlers(mockUserService, mockPublisher, mockLogger) // Pass the mock logger

			tc.setupMocks(mockUserService, mockPublisher, mockLogger)

			err := handlers.HandleUserSignupRequest(context.Background(), tc.message)

			if tc.expectedError != nil {
				if err == nil {
					t.Errorf("expected error containing '%v', got nil", tc.expectedError)
				} else if !strings.Contains(err.Error(), tc.expectedError.Error()) { // Use strings.Contains
					t.Errorf("Expected error to contain '%v', got: %v", tc.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
	for _, tc := range []struct {
		name          string
		message       *message.Message
		setupMocks    func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger)
		expectedError bool
		expectedLog   string
	}{
		{
			name: "Success",
			message: func() *message.Message {
				req := userevents.UserRoleUpdateRequest{DiscordID: "123", NewRole: userdb.UserRoleAdmin}
				payload, _ := json.Marshal(req)
				return message.NewMessage(watermill.NewUUID(), payload)
			}(),
			setupMocks: func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger) {
				resp := &userevents.UserRoleUpdateResponse{Success: true}
				mockUserService.EXPECT().OnUserRoleUpdateRequest(gomock.Any(), gomock.Any()).Return(resp, nil)
				mockPublisher.EXPECT().Publish(userevents.UserRoleUpdateResponseSubject, gomock.Any()).Return(nil)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: false,
		},
		{
			name: "Invalid Payload",
			message: func() *message.Message {
				return message.NewMessage(watermill.NewUUID(), []byte("invalid json"))
			}(),
			setupMocks: func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger) {
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: true,
			expectedLog:   "invalid character 'i' looking for beginning of value",
		},
		{
			name: "Service Error",
			message: func() *message.Message {
				req := userevents.UserRoleUpdateRequest{DiscordID: "123", NewRole: userdb.UserRoleAdmin}
				payload, _ := json.Marshal(req)
				return message.NewMessage(watermill.NewUUID(), payload)
			}(),
			setupMocks: func(mockUserService *user_mocks.MockService, mockPublisher *testutils.MockPublisher, mockLogger *testutils.MockLogger) {
				mockUserService.EXPECT().OnUserRoleUpdateRequest(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: true,
			expectedLog:   "service error",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := user_mocks.NewMockService(ctrl)
			mockPublisher := testutils.NewMockPublisher(ctrl)
			mockLogger := testutils.NewMockLogger(ctrl)
			handlers := NewHandlers(mockUserService, mockPublisher, mockLogger)

			tc.setupMocks(mockUserService, mockPublisher, mockLogger)

			err := handlers.HandleUserRoleUpdateRequest(context.Background(), tc.message)

			if tc.expectedError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tc.expectedLog != "" && !strings.Contains(err.Error(), tc.expectedLog) {
					t.Errorf("Expected error containing %q, got: %v", tc.expectedLog, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
