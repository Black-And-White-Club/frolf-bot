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

func TestUserHandlers_HandleTagAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("12345678901234567")
	testTagNumber := sharedtypes.TagNumber(1)

	testPayload := &userevents.TagAvailablePayload{
		UserID:    testUserID,
		TagNumber: testTagNumber,
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
			name: "Successfully handle TagAvailable event",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.TagAvailablePayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(gomock.Any(), testUserID, gomock.Eq(&testTagNumber)).Return(
					userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber},
					userevents.UserCreated,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to create user",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.TagAvailablePayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(gomock.Any(), testUserID, gomock.Eq(&testTagNumber)).Return(
					userservice.UserOperationResult{
						Success: nil,
						Failure: &userevents.UserCreationFailedPayload{
							UserID:    testUserID,
							TagNumber: &testTagNumber,
							Reason:    "failed",
						},
						Error: nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreationFailedPayload{UserID: testUserID, TagNumber: &testTagNumber, Reason: "failed"},
					userevents.UserCreationFailed,
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
						*out.(*userevents.TagAvailablePayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(gomock.Any(), testUserID, gomock.Eq(&testTagNumber)).Return(
					userservice.UserOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to create user with tag: internal service error",
		},
		{
			name: "Failure to create success message",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.TagAvailablePayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(gomock.Any(), testUserID, gomock.Eq(&testTagNumber)).Return(
					userservice.UserOperationResult{
						Success: &userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber},
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), &userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber}, userevents.UserCreated).Return(nil, fmt.Errorf("failed to create success message"))
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create success message",
		},
		{
			name: "Unknown result from CreateUser",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.TagAvailablePayload) = *testPayload
						return nil
					},
				)

				mockUserService.EXPECT().CreateUser(gomock.Any(), testUserID, gomock.Eq(&testTagNumber)).Return(
					userservice.UserOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "user creation service returned unexpected result",
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

			got, err := h.HandleTagAvailable(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAvailable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagAvailable() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserHandlers_HandleTagUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testUserID := sharedtypes.DiscordID("98765432109876543")
	testTagNumber := sharedtypes.TagNumber(2)

	testPayload := &userevents.TagUnavailablePayload{
		UserID:    testUserID,
		TagNumber: testTagNumber,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-unavailable-id", payloadBytes)

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
			name: "Successfully handle TagUnavailable event",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*userevents.TagUnavailablePayload) = *testPayload
						return nil
					},
				)
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreationFailedPayload{UserID: testUserID, TagNumber: &testTagNumber, Reason: "tag not available"},
					userevents.UserCreationFailed,
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

			got, err := h.HandleTagUnavailable(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagUnavailable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleTagUnavailable() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleTagUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
