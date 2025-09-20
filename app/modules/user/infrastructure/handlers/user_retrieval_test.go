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
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	usermocks "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleGetUserRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("55555555555555555")
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testUserData := &usertypes.UserData{
		UserID: testUserID,
		Role:   "member",
	}

	testPayload := &userevents.GetUserRequestPayload{
		GuildID: testGuildID,
		UserID:  testUserID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockUserService := usermocks.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle GetUser Request",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUser(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: &userevents.GetUserResponsePayload{User: testUserData},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserResponsePayload{User: testUserData},
					userevents.GetUserResponse,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "User not found",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUser(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.GetUserFailedPayload{
							UserID: testUserID,
							Reason: "user not found",
						},
						Error: nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserFailedPayload{UserID: testUserID, Reason: "user not found"},
					userevents.GetUserFailed,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in GetUser",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUser(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "technical error during GetUser service call: service error",
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

			got, err := h.HandleGetUserRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetUserRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetUserRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetUserRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserHandlers_HandleGetUserRoleRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("55555555555555555")
	testUserID := sharedtypes.DiscordID("12345678901234567")

	testPayload := &userevents.GetUserRoleRequestPayload{
		GuildID: testGuildID,
		UserID:  testUserID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	// Mock dependencies
	mockUserService := usermocks.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &usermetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle GetUserRoleRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRoleRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUserRole(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: &userevents.GetUserRoleResponsePayload{UserID: testUserID, Role: "admin"},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserRoleResponsePayload{UserID: testUserID, Role: "admin"},
					userevents.GetUserRoleResponse,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "User role not found",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRoleRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUserRole(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.GetUserRoleFailedPayload{
							UserID: testUserID,
							Reason: "user role not found",
						},
						Error: nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserRoleFailedPayload{UserID: testUserID, Reason: "user role not found"},
					userevents.GetUserRoleFailed,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in GetUserRole",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRoleRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUserRole(gomock.Any(), testGuildID, testUserID).Return(
					userservice.UserOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "technical error during GetUserRole service call: service error",
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

			got, err := h.HandleGetUserRoleRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetUserRoleRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleGetUserRoleRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleGetUserRoleRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
