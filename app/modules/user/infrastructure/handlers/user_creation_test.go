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

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := usertypes.DiscordID("12345678901234567")
	testTagNumber := 1

	testPayload := &userevents.UserSignupRequestPayload{
		UserID:    testUserID,
		TagNumber: &testTagNumber,
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
			name: "Successfully handle UserSignupRequest with tag",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestPayload) = *testPayload
						return nil
					},
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.TagAvailabilityCheckRequestedPayload{
						TagNumber: testTagNumber,
						UserID:    testUserID,
					},
					userevents.TagAvailabilityCheckRequested,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Successfully handle UserSignupRequest without tag",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestPayload) = userevents.UserSignupRequestPayload{
							UserID: testUserID,
						}
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					testUserID,
					gomock.Nil(),
				).Return(
					&userevents.UserCreatedPayload{UserID: testUserID},
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreatedPayload{UserID: testUserID},
					userevents.UserCreated,
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
			name: "Service failure in CreateUser",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestPayload) = userevents.UserSignupRequestPayload{
							UserID: testUserID,
						}
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					testUserID,
					gomock.Nil(),
				).Return(nil, nil, fmt.Errorf("internal service error"))
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to process UserSignupRequest: internal service error",
		},
		{
			name: "Failure to create success message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.UserSignupRequestPayload) = userevents.UserSignupRequestPayload{
							UserID: testUserID,
						}
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					testUserID,
					gomock.Nil(),
				).Return(
					&userevents.UserCreatedPayload{UserID: testUserID},
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreatedPayload{UserID: testUserID},
					userevents.UserCreated,
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

			got, err := h.HandleUserSignupRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleUserSignupRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleUserSignupRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
