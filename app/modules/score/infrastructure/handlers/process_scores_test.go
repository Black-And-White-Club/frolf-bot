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

func TestScoreHandlers_HandleProcessRoundScoresRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRoundID := sharedtypes.RoundID(uuid.New())

	testPayload := &scoreevents.ProcessRoundScoresRequestPayload{
		RoundID: testRoundID,
		Scores:  []sharedtypes.ScoreInfo{},
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
			name: "Successfully handle ProcessRoundScoresRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,        // Pass the RoundID directly
					testPayload.Scores, // Pass the Scores directly
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &scoreevents.ProcessRoundScoresSuccessPayload{
							RoundID:     testRoundID,
							TagMappings: []sharedtypes.TagMapping{},
						},
					},
					nil,
				)

				updateResultPayload := &scoreevents.ProcessRoundScoresSuccessPayload{
					RoundID:     testRoundID,
					TagMappings: []sharedtypes.TagMapping{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					scoreevents.ProcessRoundScoresSuccess,
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
			name: "Service failure in ProcessRoundScores",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,        // Pass the RoundID directly
					testPayload.Scores, // Pass the Scores directly
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: fmt.Errorf("internal service error"),
					},
					fmt.Errorf("internal service error"),
				)

				failureResultPayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: testRoundID,
					Error:   "internal service error",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failureResultPayload,
					scoreevents.ProcessRoundScoresFailure,
				).Return(testMsg, nil)
			},
			msg:            testMsg,
			want:           []*message.Message{testMsg},
			wantErr:        false,
			expectedErrMsg: "",
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,        // Pass the RoundID directly
					testPayload.Scores, // Pass the Scores directly
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &scoreevents.ProcessRoundScoresSuccessPayload{
							RoundID:     testRoundID,
							TagMappings: []sharedtypes.TagMapping{},
						},
					},
					nil,
				)

				updateResultPayload := &scoreevents.ProcessRoundScoresSuccessPayload{
					RoundID:     testRoundID,
					TagMappings: []sharedtypes.TagMapping{},
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					updateResultPayload,
					scoreevents.ProcessRoundScoresSuccess,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create success message: failed to create result message",
		},
		{
			name: "Service failure and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: fmt.Errorf("internal service error"),
					},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unknown result from ProcessRoundScores",
		},
		{
			name: "Unknown result from ProcessRoundScores",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,        // Pass the RoundID directly
					testPayload.Scores, // Pass the Scores directly
				).Return(
					scoreservice.ScoreOperationResult{},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unknown result from ProcessRoundScores",
		},
		{
			name: "Service failure with non-error result",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: "non-error failure",
					},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unknown result from ProcessRoundScores",
		},
		{
			name: "Service failure with error result and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: fmt.Errorf("internal service error"),
					},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unknown result from ProcessRoundScores",
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
				Helpers:      mockHelpers,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelpers)
				},
			}

			got, err := h.HandleProcessRoundScoresRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleProcessRoundScoresRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleProcessRoundScoresRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleProcessRoundScoresRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
