package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	utilmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("33333333333333333")
	testTagNumber := sharedtypes.TagNumber(1)

	// Helper functions to create messages and payloads.
	createSignupRequestMessage := func(userID sharedtypes.DiscordID, tagNumber *sharedtypes.TagNumber) *message.Message {
		payload := &userevents.UserSignupRequestedPayloadV1{
			GuildID:   testGuildID,
			UserID:    userID,
			TagNumber: tagNumber,
		}
		payloadBytes, _ := json.Marshal(payload)
		return message.NewMessage("test-id", payloadBytes)
	}

	// Mock dependencies
	mockUserService := usermocks.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		msg            *message.Message
		mockSetup      func()
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successful signup with tag",
			msg:  createSignupRequestMessage(testUserID, &testTagNumber),
			mockSetup: func() {
				// Unmarshal Payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestedPayloadV1) = userevents.UserSignupRequestedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: &testTagNumber,
						}
						return nil
					},
				)
				// CreateResultMessage for TagAvailabilityCheckRequested
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.TagAvailabilityCheckRequestedV1).
					Return(message.NewMessage("tag-check-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("tag-check-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Successful signup without tag",
			msg:  createSignupRequestMessage(testUserID, nil),
			mockSetup: func() {
				// Unmarshal Payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestedPayloadV1) = userevents.UserSignupRequestedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: nil,
						}
						return nil
					},
				)
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayloadV1{UserID: testUserID},
						Failure: nil,
						Error:   nil,
					}, nil)

				// CreateResultMessage for UserCreated (first call)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserCreatedV1).
					Return(message.NewMessage("user-created-id", []byte{}), nil)

				// CreateResultMessage for UserSignupSuccess (second call)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserSignupSucceededV1).
					Return(message.NewMessage("user-signup-success-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("user-created-id", []byte{}), message.NewMessage("user-signup-success-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Failed signup",
			msg:  createSignupRequestMessage(testUserID, nil),
			mockSetup: func() {
				// Unmarshal Payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestedPayloadV1) = userevents.UserSignupRequestedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: nil,
						}
						return nil
					},
				)
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.UserCreationFailedPayloadV1{UserID: testUserID, Reason: "failed"},
						Error:   nil,
					}, nil)

				// CreateResultMessage for UserCreationFailed (first call)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserCreationFailedV1).
					Return(message.NewMessage("user-failed-id", []byte{}), nil)

				// CreateResultMessage for UserSignupFailed (second call)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserSignupFailedV1).
					Return(message.NewMessage("user-signup-failed-id", []byte{}), nil)
			},
			want:    []*message.Message{message.NewMessage("user-failed-id", []byte{}), message.NewMessage("user-signup-failed-id", []byte{})},
			wantErr: false,
		},
		{
			name: "Error from CreateUser",
			msg:  createSignupRequestMessage(testUserID, nil),
			mockSetup: func() {
				// Unmarshal Payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestedPayloadV1) = userevents.UserSignupRequestedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: nil,
						}
						return nil
					},
				)
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{}, fmt.Errorf("service error"))
			},
			wantErr:        true,
			expectedErrMsg: "failed to process UserSignupRequest service call: service error",
		},
		{
			name: "Error creating result message",
			msg:  createSignupRequestMessage(testUserID, nil),
			mockSetup: func() {
				// Unmarshal Payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestedPayloadV1) = userevents.UserSignupRequestedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: nil,
						}
						return nil
					},
				)
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayloadV1{UserID: testUserID},
						Failure: nil,
						Error:   nil,
					}, nil)

				// CreateResultMessage error - the first call (UserCreated) fails
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserCreatedV1).
					Return(nil, fmt.Errorf("message error"))
			},
			wantErr:        true,
			expectedErrMsg: "failed to create success message: message error",
		},
		{
			name: "Error creating discord result message",
			msg:  createSignupRequestMessage(testUserID, nil),
			mockSetup: func() {
				// Unmarshal Payload
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestedPayloadV1) = userevents.UserSignupRequestedPayloadV1{
							GuildID:   testGuildID,
							UserID:    testUserID,
							TagNumber: nil,
						}
						return nil
					},
				)
				mockUserService.EXPECT().CreateUser(gomock.Any(), testGuildID, testUserID, nil, gomock.Any(), gomock.Any()).
					Return(userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayloadV1{UserID: testUserID},
						Failure: nil,
						Error:   nil,
					}, nil)

				// First CreateResultMessage succeeds (UserCreated)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserCreatedV1).
					Return(message.NewMessage("user-created-id", []byte{}), nil)

				// Second CreateResultMessage fails (UserSignupSuccess)
				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), gomock.Any(), userevents.UserSignupSucceededV1).
					Return(nil, fmt.Errorf("discord message error"))
			},
			wantErr:        true,
			expectedErrMsg: "failed to create discord success message: discord message error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			h := &UserHandlers{
				userService: mockUserService,
				logger:      logger,
				tracer:      tracer,
				metrics:     metrics,
				helpers:     mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleUserSignupRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleUserSignupRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			// Compare the contents of the messages
			if len(got) != len(tt.want) {
				t.Errorf("HandleUserSignupRequest() = %v, want %v", got, tt.want)
				return
			}

			for i := range got {
				if !reflect.DeepEqual(got[i].Payload, tt.want[i].Payload) {
					t.Errorf("HandleUserSignupRequest() got message %v, want %v", got[i], tt.want[i])
				}
			}
		})
	}
}
