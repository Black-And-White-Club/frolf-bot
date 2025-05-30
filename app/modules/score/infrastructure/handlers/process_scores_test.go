package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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
	testUserID := sharedtypes.DiscordID("user1")
	testScore := sharedtypes.Score(72)
	testTagNumber := sharedtypes.TagNumber(1)

	testProcessRoundScoresRequestPayload := &scoreevents.ProcessRoundScoresRequestPayload{
		RoundID: testRoundID,
		Scores: []sharedtypes.ScoreInfo{
			{UserID: testUserID, Score: testScore, TagNumber: &testTagNumber},
		},
	}
	payloadBytes, _ := json.Marshal(testProcessRoundScoresRequestPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)

	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

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
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &scoreevents.ProcessRoundScoresSuccessPayload{
							RoundID: testRoundID,
							TagMappings: []sharedtypes.TagMapping{
								{DiscordID: testUserID, TagNumber: testTagNumber},
							},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // Use Any() because BatchID is dynamic
					sharedevents.LeaderboardBatchTagAssignmentRequested,
				).DoAndReturn(func(msg *message.Message, payload interface{}, eventType string) (*message.Message, error) {
					// Verify payload structure
					actualPayload, ok := payload.(*sharedevents.BatchTagAssignmentRequestedPayload)
					if !ok {
						return nil, fmt.Errorf("unexpected payload type: got %T", payload)
					}
					if actualPayload.RequestingUserID != "score-service" ||
						len(actualPayload.Assignments) != 1 ||
						actualPayload.Assignments[0].UserID != testUserID ||
						actualPayload.Assignments[0].TagNumber != testTagNumber {
						return nil, fmt.Errorf("mismatched batch payload content")
					}
					return message.NewMessage("mock-batch-assign-id", []byte("mock-batch-payload")), nil
				})
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("mock-batch-assign-id", []byte("mock-batch-payload"))},
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
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				// Service returns proper failure payload
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: testRoundID,
					Error:   "internal service error",
				}

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: failurePayload, // Proper failure payload type
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					scoreevents.ProcessRoundScoresFailure,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: &scoreevents.ProcessRoundScoresSuccessPayload{
							RoundID:     testRoundID,
							TagMappings: []sharedtypes.TagMapping{},
						},
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					sharedevents.LeaderboardBatchTagAssignmentRequested,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create batch tag assignment message: failed to create result message",
		},
		{
			name: "Service failure and CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				// Service returns proper failure payload
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: testRoundID,
					Error:   "internal service error",
				}

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: failurePayload, // Proper failure payload type
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					scoreevents.ProcessRoundScoresFailure,
				).Return(nil, fmt.Errorf("failed to create result message for failure"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create failure message from result failure payload: failed to create result message for failure",
		},
		{
			name: "Unknown result from ProcessRoundScores",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{}, // Neither success nor failure
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service: expected *scoreevents.ProcessRoundScoresSuccessPayload, got <nil>",
		},
		{
			name: "Service returns direct error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{}, // No failure payload
					fmt.Errorf("direct service error"),  // Direct error
				)

				// Handler creates failure payload from direct error
				failurePayload := &scoreevents.ProcessRoundScoresFailurePayload{
					RoundID: testRoundID,
					Error:   "direct service error",
				}

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					scoreevents.ProcessRoundScoresFailure,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false,
		},
		{
			name: "Service returns wrong failure payload type",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ProcessRoundScoresRequestPayload) = *testProcessRoundScoresRequestPayload
						return nil
					},
				)

				mockScoreService.EXPECT().ProcessRoundScores(
					gomock.Any(),
					testRoundID,
					testProcessRoundScoresRequestPayload.Scores,
				).Return(
					scoreservice.ScoreOperationResult{
						Failure: "wrong type", // Wrong type, should be *ProcessRoundScoresFailurePayload
					},
					nil,
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "unexpected failure payload type from service: expected *scoreevents.ProcessRoundScoresFailurePayload, got string",
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

			// Special handling for the successful case due to dynamic BatchID
			if tt.name == "Successfully handle ProcessRoundScoresRequest" && !tt.wantErr {
				if len(got) != 1 || got[0].UUID == "" || len(got[0].Payload) == 0 {
					t.Errorf("HandleProcessRoundScoresRequest() got = %v, want a single non-empty message", got)
				}
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HandleProcessRoundScoresRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
