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

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("55555555555555555")
	testNewRole := sharedtypes.UserRoleEnum("admin")

	testPayload := &userevents.UserRoleUpdateRequestedPayloadV1{
		GuildID: testGuildID,
		UserID:  testUserID,
		Role:    testNewRole,
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
			name: "Successfully handle UserRoleUpdateRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserRoleUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				updateResultPayload := &userevents.UserRoleUpdateResultPayloadV1{
					GuildID: testGuildID,
					Success: true,
					UserID:  testUserID,
					Role:    testNewRole,
					Reason:  "",
				}

				mockUserService.EXPECT().UpdateUserRoleInDatabase(gomock.Any(), testGuildID, testUserID, testNewRole).Return(
					userservice.UserOperationResult{
						Success: updateResultPayload,
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				// Handler passes the pointer - expect the pointer
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload, // This is the pointer that the handler passes
					userevents.UserRoleUpdatedV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Failed to update user role",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserRoleUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				failurePayload := &userevents.UserRoleUpdateResultPayloadV1{
					GuildID: testGuildID,
					Success: false,
					UserID:  testUserID,
					Role:    testNewRole,
					Reason:  "user not found",
				}

				mockUserService.EXPECT().UpdateUserRoleInDatabase(gomock.Any(), testGuildID, testUserID, testNewRole).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: failurePayload,
						Error:   nil,
					},
					nil,
				)

				// Handler passes the pointer - expect the pointer
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload, // This is the pointer that the handler passes
					userevents.UserRoleUpdateFailedV1,
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
			name: "Service failure in UpdateUserRoleInDatabase",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserRoleUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().UpdateUserRoleInDatabase(gomock.Any(), testGuildID, testUserID, testNewRole).Return(
					userservice.UserOperationResult{},
					fmt.Errorf("service error"),
				)

				// Handler creates a new payload and passes a value - expect the value
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					userevents.UserRoleUpdateResultPayloadV1{
						GuildID: testGuildID,
						Success: false,
						UserID:  testUserID,
						Role:    testNewRole,
						Reason:  "internal service error: service error",
					},
					userevents.UserRoleUpdateFailedV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
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

			got, err := h.HandleUserRoleUpdateRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserRoleUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleUserRoleUpdateRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleUserRoleUpdateRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
