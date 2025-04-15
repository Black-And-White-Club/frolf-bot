package userhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	utilmocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
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

	invalidMsg := message.NewMessage("test-id", []byte("invalid json")) // Corrupted payload

	// Mock dependencies
	mockUserService := userservice.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl) // Use the mock

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
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
					&userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber},
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber},
					userevents.UserCreated,
				).Return(testMsg, nil)
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordTagAvailabilityCheck(gomock.Any(), true, sharedtypes.TagNumber(testTagNumber)).Times(1) // Corrected here
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagAvailable", gomock.Any()).Times(1)
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
					nil,
					&userevents.UserCreationFailedPayload{
						UserID:    testUserID,
						TagNumber: &testTagNumber,
						Reason:    "failed",
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					&userevents.UserCreationFailedPayload{UserID: testUserID, TagNumber: &testTagNumber, Reason: "failed"},
					userevents.UserCreationFailed,
				).Return(testMsg, nil)
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordTagAvailabilityCheck(gomock.Any(), false, sharedtypes.TagNumber(testTagNumber)).Times(1)
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagAvailable", gomock.Any()).Times(1)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagAvailable", gomock.Any()).Times(1)
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

				mockUserService.EXPECT().CreateUser(gomock.Any(), testUserID, gomock.Eq(&testTagNumber)).Return(nil, nil, fmt.Errorf("internal service error"))
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagAvailable", gomock.Any()).Times(1)
			},
			msg:            testMsg,
			wantErr:        true,
			expectedErrMsg: "failed to create user: internal service error",
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
					&userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber},
					nil,
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(gomock.Any(), &userevents.UserCreatedPayload{UserID: testUserID, TagNumber: &testTagNumber}, userevents.UserCreated).Return(nil, fmt.Errorf("failed to create success message"))
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "HandleTagAvailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagAvailable", gomock.Any()).Times(1)
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
				metrics:     mockMetrics, // Use the mock
				helpers:     mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, mockMetrics, tracer, mockHelpers) // Pass mockMetrics
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
	mockUserService := userservice.NewMockService(ctrl)
	mockHelpers := utilmocks.NewMockHelpers(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

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
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagUnavailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerSuccess(gomock.Any(), "HandleTagUnavailable").Times(1)
				mockMetrics.EXPECT().RecordTagAvailabilityCheck(gomock.Any(), false, sharedtypes.TagNumber(testTagNumber)).Times(1) // Corrected here
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagUnavailable", gomock.Any()).Times(1)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
				mockMetrics.EXPECT().RecordHandlerAttempt(gomock.Any(), "HandleTagUnavailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerFailure(gomock.Any(), "HandleTagUnavailable").Times(1)
				mockMetrics.EXPECT().RecordHandlerDuration(gomock.Any(), "HandleTagUnavailable", gomock.Any()).Times(1)
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
				metrics:     mockMetrics,
				helpers:     mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, mockMetrics, tracer, mockHelpers)
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
