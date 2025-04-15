package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestScoreHandlers_HandleCorrectScoreRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTagNumber := sharedtypes.TagNumber(1)

	testPayload := &scoreevents.ScoreUpdateRequestPayload{
		RoundID:   testRoundID,
		UserID:    testUserID,
		Score:     testScore,
		TagNumber: &testTagNumber,
	}
	payloadBytes, _ := json.Marshal(testPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json")) // Corrupted payload

	// Mock dependencies
	mockScoreService := scoremocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &scoremetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		msg            *message.Message
		want           []*message.Message
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle CorrectScoreRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &scoreevents.ScoreUpdateSuccessPayload{
							RoundID: testRoundID,
							UserID:  testUserID,
							Score:   testScore,
						},
					},
					nil,
				)

				updateResultPayload := &scoreevents.ScoreUpdateSuccessPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					scoreevents.ScoreUpdateSuccess,
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
			name: "Service failure in CorrectScore",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: &scoreevents.ScoreUpdateFailurePayload{
							RoundID: testRoundID,
							UserID:  testUserID,
							Error:   "internal service error",
						},
						Error: fmt.Errorf("internal service error"),
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "internal service error",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &scoreevents.ScoreUpdateSuccessPayload{
							RoundID: testRoundID,
							UserID:  testUserID,
							Score:   testScore,
						},
					},
					nil,
				)

				updateResultPayload := &scoreevents.ScoreUpdateSuccessPayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					scoreevents.ScoreUpdateSuccess,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create response message: failed to create result message",
		},
		{
			name: "Service failure and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: &scoreevents.ScoreUpdateFailurePayload{
							RoundID: testRoundID,
							UserID:  testUserID,
							Error:   "internal service error",
						},
						Error: fmt.Errorf("internal service error"),
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "internal service error",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: &scoreevents.ScoreUpdateFailurePayload{
							RoundID: testRoundID,
							UserID:  testUserID,
							Error:   "non-error failure",
						},
					},
					nil,
				)

				failureResultPayload := &scoreevents.ScoreUpdateFailurePayload{
					RoundID: testRoundID,
					UserID:  testUserID,
					Error:   "non-error failure",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					scoreevents.ScoreUpdateFailure,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service failure with error result and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: &scoreevents.ScoreUpdateFailurePayload{
							RoundID: testRoundID,
							UserID:  testUserID,
							Error:   "internal service error",
						},
						Error: fmt.Errorf("internal service error"),
					},
					fmt.Errorf("internal service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "internal service error",
		},
		{
			name: "Unknown result from CorrectScore",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service",
		},
		{
			name: "Invalid payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload type: expected ScoreUpdateRequestPayload"))
			},
			msg:            invalidMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload type: expected ScoreUpdateRequestPayload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			h := &ScoreHandlers{
				scoreService: mockScoreService,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleCorrectScoreRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleCorrectScoreRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleCorrectScoreRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
