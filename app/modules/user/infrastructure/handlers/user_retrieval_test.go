package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	utilmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleGetUserRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := usertypes.DiscordID("12345678901234567")
	testUserData := &usertypes.UserData{
		ID:     1,
		UserID: testUserID,
		Role:   usertypes.UserRoleEnum("admin"),
	}

	testPayload := &userevents.GetUserRequestPayload{
		UserID: testUserID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json")) // Corrupted payload

	// Mock dependencies
	mockUserService := userservice.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &usermetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle GetUserRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUser(
					gomock.Any(),
					gomock.Any(),
					testUserID,
				).Return(
					&userevents.GetUserResponsePayload{User: testUserData},
					nil,
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
			name: "Service failure in GetUser ",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUser(
					gomock.Any(),
					gomock.Any(),
					testUserID,
				).Return(
					nil, &userevents.GetUserFailedPayload{
						UserID: testUserID,
						Reason: "internal service error",
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserFailedPayload{
						UserID: testUserID,
						Reason: "internal service error",
					},
					userevents.GetUserFailed,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Failure to create success message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUser(
					gomock.Any(),
					gomock.Any(),
					testUserID,
				).Return(
					&userevents.GetUserResponsePayload{User: testUserData},
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserResponsePayload{User: testUserData},
					userevents.GetUserResponse,
				).Return(nil, fmt.Errorf("failed to create success message"))
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create success message",
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

	testUserID := usertypes.DiscordID("12345678901234567")

	testPayload := &userevents.GetUserRoleRequestPayload{
		UserID: testUserID,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json")) // Corrupted payload

	// Mock dependencies
	mockUserService := userservice.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)

	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &usermetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

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

				mockUserService.EXPECT().GetUserRole(
					gomock.Any(),
					gomock.Any(),
					testUserID,
				).Return(
					&userevents.GetUserRoleResponsePayload{UserID: testUserID, Role: "admin"},
					nil,
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

				mockUserService.EXPECT().GetUserRole(
					gomock.Any(),
					gomock.Any(),
					testUserID,
				).Return(nil, nil, fmt.Errorf("internal service error"))

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserRoleFailedPayload{
						UserID: testUserID,
						Reason: "internal service error",
					},
					userevents.GetUserRoleFailed,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Failure to create success message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.GetUserRoleRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().GetUserRole(
					gomock.Any(),
					gomock.Any(),
					testUserID,
				).Return(
					&userevents.GetUserRoleResponsePayload{UserID: testUserID, Role: "admin"},
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.GetUserRoleResponsePayload{UserID: testUserID, Role: "admin"},
					userevents.GetUserRoleResponse,
				).Return(nil, fmt.Errorf("failed to create success message"))
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to create GetUserRoleResponse message: failed to create success message",
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
