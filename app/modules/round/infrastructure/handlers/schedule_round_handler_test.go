package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundStored(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testStartTime := sharedtypes.StartTime(time.Now())
	testTitle := roundtypes.Title("Test Round")
	testUserID := sharedtypes.DiscordID("1234567890")

	testPayload := &roundevents.RoundStoredPayload{
		Round: roundtypes.Round{
			ID:        testRoundID,
			StartTime: &testStartTime,
			Title:     testTitle,
			CreatedBy: testUserID,
		},
	}

	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
		mockSetup      func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers)
	}{
		{
			name: "Successfully handle RoundStored",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundStoredPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        testRoundID,
							StartTime: &testStartTime,
							Title:     testTitle,
							CreatedBy: testUserID,
						},
					},
					testStartTime,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundScheduledPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							EventMessageID: nil,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScheduled,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service failure in ScheduleRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundStoredPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        testRoundID,
							StartTime: &testStartTime,
							Title:     testTitle,
							CreatedBy: testUserID,
						},
					},
					testStartTime,
				).Return(
					roundservice.RoundOperationResult{},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to handle RoundStored event: internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundStoredPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        testRoundID,
							StartTime: &testStartTime,
							Title:     testTitle,
							CreatedBy: testUserID,
						},
					},
					testStartTime,
				).Return(
					roundservice.RoundOperationResult{
						Success: &roundevents.RoundScheduledPayload{
							BaseRoundPayload: roundtypes.BaseRoundPayload{
								RoundID:     testRoundID,
								Title:       testTitle,
								Description: nil,
								Location:    nil,
								StartTime:   &testStartTime,
								UserID:      testUserID,
							},
							EventMessageID: nil,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundScheduled,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Unknown result from ScheduleRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundStoredPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        testRoundID,
							StartTime: &testStartTime,
							Title:     testTitle,
							CreatedBy: testUserID,
						},
					},
					testStartTime,
				).Return(
					roundservice.RoundOperationResult{}, // Return empty result
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Failure result from ScheduleRoundEvents",
			mockSetup: func(mockRoundService *roundmocks.MockService, mockHelpers *mocks.MockHelpers) {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*roundevents.RoundStoredPayload) = *testPayload
						return nil
					},
				)

				mockRoundService.EXPECT().ScheduleRoundEvents(
					gomock.Any(),
					roundevents.RoundStoredPayload{
						Round: roundtypes.Round{
							ID:        testRoundID,
							StartTime: &testStartTime,
							Title:     testTitle,
							CreatedBy: testUserID,
						},
					},
					testStartTime,
				).Return(
					roundservice.RoundOperationResult{
						Failure: &roundevents.RoundErrorPayload{
							RoundID: testRoundID,
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					roundevents.RoundError,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundService := roundmocks.NewMockService(ctrl)
			mockHelpers := mocks.NewMockHelpers(ctrl)

			tt.mockSetup(mockRoundService, mockHelpers)

			h := &RoundHandlers{
				roundService: mockRoundService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleRoundStored(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundStored() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleRoundStored() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleRoundStored() = %v, want %v", got, tt.want)
			}
		})
	}
}
