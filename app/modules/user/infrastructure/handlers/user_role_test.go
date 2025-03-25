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

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := usertypes.DiscordID("12345678901234567")
	testRequesterID := usertypes.DiscordID("98765432109876543")
	testNewRole := usertypes.UserRoleEnum("admin")

	testPayload := &userevents.UserRoleUpdateRequestPayload{
		UserID:      testUserID,
		Role:        testNewRole,
		RequesterID: testRequesterID,
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
			name: "Successfully handle UserRoleUpdateRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserRoleUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				updateResultPayload := &userevents.UserRoleUpdateResultPayload{
					UserID: testUserID,
					Role:   testNewRole,
				}

				mockUserService.EXPECT().UpdateUserRoleInDatabase(
					gomock.Any(),
					gomock.Any(),
					testUserID,
					testNewRole,
				).Return(
					updateResultPayload,
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					userevents.UserRoleUpdated,
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
						*out.(*userevents.UserRoleUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().UpdateUserRoleInDatabase(
					gomock.Any(),
					gomock.Any(),
					testUserID,
					testNewRole,
				).Return(nil, &userevents.UserRoleUpdateFailedPayload{
					UserID: testUserID,
					Reason: "internal service error",
				}, nil)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserRoleUpdateFailedPayload{
						UserID: testUserID,
						Reason: "internal service error",
					},
					userevents.UserRoleUpdateFailed,
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
						*out.(*userevents.UserRoleUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().UpdateUserRoleInDatabase(
					gomock.Any(),
					gomock.Any(),
					testUserID,
					testNewRole,
				).Return(
					nil,
					&userevents.UserRoleUpdateFailedPayload{
						UserID: testUserID,
						Reason: "internal service error",
					},
					fmt.Errorf("internal service error"),
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserRoleUpdatedPayload{UserID: testUserID, Role: testNewRole},
					userevents.UserRoleUpdated,
				).Return(nil, fmt.Errorf("failed to create success message"))
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to create UserRoleUpdateResult message: failed to create success message",
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
