package userhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	usermocks "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestNewHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := usermocks.NewMockService(ctrl)
	mockEventBus := eventbusmock.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handlers := NewHandlers(mockUserService, mockEventBus, logger)

	if handlers == nil {
		t.Error("NewHandlers() returned nil")
	}
}

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		message       *message.Message
		setupMocks    func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger)
		expectedError error
	}{
		{
			name: "Success",
			message: func() *message.Message {
				req := userevents.UserSignupRequestPayload{DiscordID: "123", TagNumber: 1}
				payload, _ := json.Marshal(req)
				msg := message.NewMessage("mock-uuid", payload)
				msg.Metadata.Set("correlation_id", "mock-correlation-id")
				return msg
			}(),
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger) {
				resp := &userevents.UserSignupResponsePayload{Success: true}
				mockUserService.EXPECT().OnUserSignupRequest(gomock.Any(), gomock.Any()).Return(resp, nil)
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userstream.UserSignupResponseStreamName, gomock.Any()). // Use UserSignupResponseStreamName
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						// Verify the stream name
						if streamName != userstream.UserSignupResponseStreamName {
							t.Errorf("Expected stream name: %s, got: %s", userstream.UserSignupResponseStreamName, streamName)
						}
						// Verify the subject from message metadata
						subject := msg.Metadata.Get("subject")
						if subject != userevents.UserSignupResponse {
							t.Errorf("Expected subject: %s, got: %s", userevents.UserSignupResponse, subject)
						}
						return nil
					}).
					Times(1)
				logger.Info("HandleUserSignupRequest started", slog.String("contextErr", ""))
				logger.Info("Processing UserSignupRequest", slog.String("message_id", "mock-uuid"))
				logger.Info("HandleUserSignupRequest completed", slog.String("message_id", "mock-uuid"))
			},
			expectedError: nil,
		},
		{
			name: "Invalid Payload",
			message: func() *message.Message {
				msg := message.NewMessage("mock-uuid", []byte("invalid json"))
				msg.Metadata.Set("correlation_id", "mock-correlation-id")
				return msg
			}(),
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger) {
				logger.Info("HandleUserSignupRequest started", slog.String("contextErr", ""))
				logger.Info("Processing UserSignupRequest", slog.String("message_id", "mock-uuid"))
				logger.Error("Failed to unmarshal UserSignupRequest", slog.Any("error", errors.New("invalid character 'i' looking for beginning of value")), slog.String("message_id", "mock-uuid"))
			},
			expectedError: errors.New("failed to unmarshal UserSignupRequest"),
		},
		{
			name: "Service Error",
			message: func() *message.Message {
				req := userevents.UserSignupRequestPayload{DiscordID: "123", TagNumber: 1}
				payload, _ := json.Marshal(req)
				msg := message.NewMessage("mock-uuid", payload)
				msg.Metadata.Set("correlation_id", "mock-correlation-id")
				return msg
			}(),
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger) {
				mockUserService.EXPECT().OnUserSignupRequest(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				logger.Info("HandleUserSignupRequest started", slog.String("contextErr", ""))
				logger.Info("Processing UserSignupRequest", slog.String("message_id", "mock-uuid"))

				// Expect Publish to be called with an error payload
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userstream.UserSignupResponseStreamName, gomock.Any()). // Use UserSignupResponseStreamName
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						// Verify the stream name
						if streamName != userstream.UserSignupResponseStreamName {
							t.Errorf("Expected stream name: %s, got: %s", userstream.UserSignupResponseStreamName, streamName)
						}
						// Verify the subject from message metadata
						subject := msg.Metadata.Get("subject")
						if subject != userevents.UserSignupResponse {
							t.Errorf("Expected subject: %s, got: %s", userevents.UserSignupResponse, subject)
						}
						return nil
					}).
					Times(1)
				logger.Error("Failed to process user signup request", slog.Any("error", errors.New("service error")), slog.String("message_id", "mock-uuid"), slog.String("discord_id", "123"), slog.Int("tag_number", 1))
				logger.Info("HandleUserSignupRequest completed", slog.String("message_id", "mock-uuid"))
			},
			expectedError: nil, // We expect no error from the handler as it handles it internally
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserService := usermocks.NewMockService(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handlers := &UserHandlers{
				UserService: mockUserService,
				EventBus:    mockEventBus,
				logger:      logger,
			}

			tc.setupMocks(mockUserService, mockEventBus, logger)

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
		message       *message.Message // Corrected type
		setupMocks    func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger)
		expectedError string
	}{
		{
			name: "Success",
			message: func() *message.Message { // Helper function to create the message
				req := userevents.UserRoleUpdateRequestPayload{
					DiscordID: "123",
					NewRole:   usertypes.UserRoleEditor,
				}
				payload, _ := json.Marshal(req)
				msg := message.NewMessage("mock-uuid", payload)
				msg.Metadata.Set("correlation_id", "mock-correlation-id")
				return msg
			}(), // Immediately call the function
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger) {
				req := userevents.UserRoleUpdateRequestPayload{
					DiscordID: "123",
					NewRole:   usertypes.UserRoleEditor,
				}
				resp := &userevents.UserRoleUpdateResponsePayload{Success: true}
				logger.Info("HandleUserRoleUpdateRequest started", slog.String("message_id", "mock-uuid"), slog.String("contextErr", ""))
				mockUserService.EXPECT().
					OnUserRoleUpdateRequest(gomock.Any(), gomock.Eq(req)).
					Return(resp, nil).Times(1)

				// Expect Publish to be called with a Watermill message
				mockEventBus.EXPECT().
					Publish(gomock.Any(), userstream.UserRoleUpdateResponseStreamName, gomock.Any()).
					DoAndReturn(func(ctx context.Context, streamName string, msg *message.Message) error {
						// Verify the stream name
						if streamName != userstream.UserRoleUpdateResponseStreamName {
							t.Errorf("Expected stream name: %s, got: %s", userstream.UserRoleUpdateResponseStreamName, streamName)
						}
						// Verify the subject from message metadata
						subject := msg.Metadata.Get("subject")
						if subject != userevents.UserRoleUpdateResponse {
							t.Errorf("Expected subject: %s, got: %s", userevents.UserRoleUpdateResponse, subject)
						}
						return nil
					}).
					Times(1)

				logger.Info("HandleUserRoleUpdateRequest completed successfully", slog.String("message_id", "mock-uuid"))
			},
			expectedError: "",
		},
		{
			name: "Invalid Payload",
			message: func() *message.Message { // Helper function
				msg := message.NewMessage("mock-uuid", []byte("invalid json"))
				msg.Metadata.Set("correlation_id", "mock-correlation-id")
				return msg
			}(), // Immediately call the function
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger) {
				logger.Info("HandleUserRoleUpdateRequest started", slog.String("message_id", "mock-uuid"), slog.String("contextErr", ""))
				logger.Error("Failed to unmarshal UserRoleUpdateRequest", slog.Any("error", errors.New("invalid character 'i' looking for beginning of value")), slog.String("message_id", "mock-uuid"))
			},
			expectedError: "failed to unmarshal UserRoleUpdateRequest",
		},
		{
			name: "Service Failure",
			message: func() *message.Message { // Helper function
				req := userevents.UserRoleUpdateRequestPayload{
					DiscordID: "123",
					NewRole:   usertypes.UserRoleEditor,
				}
				payload, _ := json.Marshal(req)
				msg := message.NewMessage("mock-uuid", payload)
				msg.Metadata.Set("correlation_id", "mock-correlation-id")
				return msg
			}(), // Immediately call the function
			setupMocks: func(mockUserService *usermocks.MockService, mockEventBus *eventbusmock.MockEventBus, logger *slog.Logger) {
				logger.Info("HandleUserRoleUpdateRequest started", slog.String("message_id", "mock-uuid"), slog.String("contextErr", ""))

				mockUserService.EXPECT().
					OnUserRoleUpdateRequest(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("service failure")).Times(1)

				logger.Error("Failed to process user role update request", slog.Any("error", errors.New("service failure")), slog.String("message_id", "mock-uuid"), slog.String("error_msg", "service failure"))

			},
			expectedError: "service failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserService := usermocks.NewMockService(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handlers := &UserHandlers{
				UserService: mockUserService,
				EventBus:    mockEventBus,
				logger:      logger,
			}

			tc.setupMocks(mockUserService, mockEventBus, logger)

			err := handlers.HandleUserRoleUpdateRequest(context.Background(), tc.message)

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
