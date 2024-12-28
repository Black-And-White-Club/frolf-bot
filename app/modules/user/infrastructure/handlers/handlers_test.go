package userhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	"github.com/Black-And-White-Club/tcr-bot/app/events"
	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/events/mocks"
	usermocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	testutils "github.com/Black-And-White-Club/tcr-bot/internal/testutils"
	"go.uber.org/mock/gomock"
)

func TestNewHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := usermocks.NewMockService(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	mockLogger := testutils.NewMockLoggerAdapter(ctrl)

	handlers := NewHandlers(mockUserService, mockEventBus, mockLogger)

	if handlers == nil {
		t.Error("NewHandlers() returned nil")
	}
}

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		message       *testutils.MockMessage
		setupMocks    func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter)
		expectedError error
	}{
		{
			name: "Success",
			message: func() *testutils.MockMessage {
				msg := testutils.NewMockMessage(ctrl)
				req := userevents.UserSignupRequestPayload{DiscordID: "123", TagNumber: 1}
				payload, _ := json.Marshal(req)
				msg.EXPECT().Payload().Return(payload).AnyTimes()
				msg.EXPECT().UUID().Return("mock-uuid").AnyTimes()
				msg.EXPECT().Context().Return(context.Background()).AnyTimes()
				msg.EXPECT().Ack().Times(1)
				return msg
			}(),
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter) {
				resp := &userevents.UserSignupResponsePayload{Success: true}
				mockUserService.EXPECT().OnUserSignupRequest(gomock.Any(), gomock.Any()).Return(resp, nil)
				mockEventBus.EXPECT().Publish(gomock.Any(), userevents.UserSignupResponse, gomock.Any()).Return(nil)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: nil,
		},
		{
			name: "Invalid Payload",
			message: func() *testutils.MockMessage {
				msg := testutils.NewMockMessage(ctrl)
				msg.EXPECT().Payload().Return([]byte("invalid json")).AnyTimes()
				msg.EXPECT().UUID().Return("mock-uuid").AnyTimes()
				return msg
			}(),
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: errors.New("failed to unmarshal UserSignupRequest"),
		},
		{
			name: "Service Error",
			message: func() *testutils.MockMessage {
				msg := testutils.NewMockMessage(ctrl)
				req := userevents.UserSignupRequestPayload{DiscordID: "123", TagNumber: 1}
				payload, _ := json.Marshal(req)
				msg.EXPECT().Payload().Return(payload).AnyTimes()
				msg.EXPECT().UUID().Return("mock-uuid").AnyTimes()
				msg.EXPECT().Context().Return(context.Background()).AnyTimes()
				return msg
			}(),
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter) {
				mockUserService.EXPECT().OnUserSignupRequest(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: errors.New("failed to process user signup request: service error"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserService := usermocks.NewMockService(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			handlers := &UserHandlers{
				UserService: mockUserService,
				EventBus:    mockEventBus,
				logger:      mockLogger,
			}

			tc.setupMocks(mockUserService, mockEventBus, mockLogger)

			err := handlers.HandleUserSignupRequest(context.Background(), tc.message)

			if tc.expectedError != nil {
				if err == nil {
					t.Errorf("expected error containing '%v', got nil", tc.expectedError)
				} else if !strings.Contains(err.Error(), tc.expectedError.Error()) {
					t.Errorf("Expected error to contain '%v', got: %v", tc.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		message       func() *testutils.MockMessage
		setupMocks    func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter)
		expectedError string
	}{
		{
			name: "Success",
			message: func() *testutils.MockMessage {
				msg := testutils.NewMockMessage(ctrl)
				req := userevents.UserRoleUpdateRequestPayload{
					DiscordID: "123",
					NewRole:   usertypes.UserRoleEditor,
				}
				payload, _ := json.Marshal(req)
				msg.EXPECT().Payload().Return(payload).AnyTimes()
				msg.EXPECT().UUID().Return("mock-uuid").AnyTimes()
				msg.EXPECT().Context().Return(context.Background()).AnyTimes()
				msg.EXPECT().Ack().Times(1)
				return msg
			},
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter) {
				req := userevents.UserRoleUpdateRequestPayload{
					DiscordID: "123",
					NewRole:   usertypes.UserRoleEditor,
				}
				resp := &userevents.UserRoleUpdateResponsePayload{Success: true}
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
				mockUserService.EXPECT().
					OnUserRoleUpdateRequest(gomock.Any(), gomock.Eq(req)).
					Return(resp, nil).Times(1)

				// Expect Publish to be called with a Watermill message
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userevents.UserRoleUpdateResponse, gomock.Any()).
					DoAndReturn(func(ctx context.Context, topic events.EventType, msg *adapters.WatermillMessageAdapter) error {
						var responsePayload userevents.UserRoleUpdateResponsePayload
						if err := json.Unmarshal(msg.Payload(), &responsePayload); err != nil {
							t.Errorf("failed to unmarshal response payload: %v", err)
						}

						if !reflect.DeepEqual(responsePayload, *resp) {
							t.Errorf("unexpected response payload: got %v, want %v", responsePayload, *resp)
						}

						return nil
					}).Times(1)

				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedError: "",
		},
		{
			name: "Invalid Payload",
			message: func() *testutils.MockMessage {
				msg := testutils.NewMockMessage(ctrl)
				msg.EXPECT().Payload().Return([]byte("invalid json")).AnyTimes()
				msg.EXPECT().UUID().Return("mock-uuid").AnyTimes()
				return msg
			},
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

				// Use testutils.isErrorType to create the matcher
				mockLogger.EXPECT().
					Error(gomock.Any(), testutils.IsErrorType(&json.SyntaxError{}), gomock.Any()).
					Times(1)
			},
			expectedError: "failed to unmarshal UserRoleUpdateRequest",
		},
		{
			name: "Service Failure",
			message: func() *testutils.MockMessage {
				msg := testutils.NewMockMessage(ctrl)
				req := userevents.UserRoleUpdateRequestPayload{
					DiscordID: "123",
					NewRole:   usertypes.UserRoleEditor,
				}
				payload, _ := json.Marshal(req)
				msg.EXPECT().Payload().Return(payload).AnyTimes()
				msg.EXPECT().UUID().Return("mock-uuid").AnyTimes()
				msg.EXPECT().Context().Return(context.Background()).AnyTimes()
				return msg
			},
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, mockLogger *testutils.MockLoggerAdapter) {
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

				mockUserService.EXPECT().
					OnUserRoleUpdateRequest(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("service failure")).Times(1)

				mockLogger.EXPECT().
					Error(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, msg interface{}, fields ...types.LogFields) {
						// Assert that msg is an error and convert it to string
						err, ok := msg.(error)
						if !ok {
							t.Fatalf("Error message is not an error type: %v", msg)
						}
						errMsg := err.Error()

						// Assert that the error message contains "service failure"
						if !strings.Contains(errMsg, "service failure") {
							t.Errorf("Expected error message to contain 'service failure', got '%s'", errMsg)
						}

						// Check if any of the fields contain "Failed to process user role update request"
						found := false
						for _, field := range fields {
							if msgVal, ok := field["message_id"]; ok {
								if _, ok := msgVal.(string); ok {
									found = true
									break
								}
							}
						}
						if !found {
							t.Errorf("Expected message to contain 'Failed to process user role update request', but it was not found in fields: %v", fields)
						}
					}).
					Times(1)
			},
			expectedError: "service failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserService := usermocks.NewMockService(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			mockLogger := testutils.NewMockLoggerAdapter(ctrl)

			handlers := &UserHandlers{
				UserService: mockUserService,
				EventBus:    mockEventBus,
				logger:      mockLogger,
			}

			tc.setupMocks(mockUserService, mockEventBus, mockLogger)

			err := handlers.HandleUserRoleUpdateRequest(context.Background(), tc.message())

			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("expected error '%v', got: %v", tc.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
