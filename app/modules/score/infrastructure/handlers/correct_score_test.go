package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"
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

	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTagNumber := sharedtypes.TagNumber(1)

	testPayload := &scoreevents.ScoreUpdateRequestedPayloadV1{
		GuildID:   testGuildID,
		RoundID:   testRoundID,
		UserID:    testUserID,
		Score:     testScore,
		TagNumber: &testTagNumber,
	}
	payloadBytes, _ := json.Marshal(testPayload)
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
			name: "Successfully handle CorrectScoreRequest",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				successPayload := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: successPayload,
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				successMsg := message.NewMessage("success-msg-id", []byte("success"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successPayload,
					scoreevents.ScoreUpdatedV1,
				).Return(successMsg, nil)

				// Expect GetScoresForRound to be called for reprocessing
				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: testUserID, Score: testScore, TagNumber: &testTagNumber},
				}, nil)

				// Expect CreateResultMessage for the reprocess request
				reprocessMsg := message.NewMessage("reprocess-msg-id", []byte("reprocess"))
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // The ProcessRoundScoresRequestedPayloadV1
					scoreevents.ProcessRoundScoresRequestedV1,
				).Return(reprocessMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{message.NewMessage("success-msg-id", []byte("success")), message.NewMessage("reprocess-msg-id", []byte("reprocess"))},
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
						*out.(*scoreevents.ScoreUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				failurePayload := &scoreevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Reason:  "internal service error",
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: nil,
						Failure: failurePayload,
						Error:   nil,
					},
					nil,
				)

				// When there's a failure, the handler creates a failure message
				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					scoreevents.ScoreUpdateFailedV1,
				).Return(testMsg, nil)
			},
			msg:     testMsg,
			want:    []*message.Message{testMsg},
			wantErr: false, // No error because failure is handled gracefully
		},
		{
			name: "Service returns error",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{},
					fmt.Errorf("service error"),
				)
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "service error", // Changed from "technical error during CorrectScore service call: service error"
		},
		{
			name: "Service success but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				successPayload := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Score:   testScore,
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: successPayload,
						Failure: nil,
						Error:   nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					successPayload,
					scoreevents.ScoreUpdatedV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create ScoreUpdateSuccess message: failed to create result message",
		},
		{
			name: "Service failure but CreateResultMessage fails",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				failurePayload := &scoreevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  testUserID,
					Reason:  "internal service error",
				}

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					testUserID,
					testScore,
					&testTagNumber,
				).Return(
					scoreservice.ScoreOperationResult{
						Success: nil,
						Failure: failurePayload,
						Error:   nil,
					},
					nil,
				)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload,
					scoreevents.ScoreUpdateFailedV1,
				).Return(nil, fmt.Errorf("failed to create result message"))
			},
			msg:            testMsg,
			want:           nil,
			wantErr:        true,
			expectedErrMsg: "failed to create ScoreUpdateFailure message: failed to create result message",
		},
		{
			name: "Unknown result from CorrectScore",
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdateRequestedPayloadV1) = *testPayload
						return nil
					},
				)

				mockScoreService.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
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
			expectedErrMsg: "unexpected result from service: neither success nor failure",
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

			got, err := h.HandleCorrectScoreRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleCorrectScoreRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			// Compare message count and UUIDs instead of deep equality on pointers
			if len(got) != len(tt.want) {
				t.Errorf("HandleCorrectScoreRequest() returned %d messages, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if i < len(tt.want) && got[i].UUID != tt.want[i].UUID {
					t.Errorf("HandleCorrectScoreRequest() message[%d].UUID = %v, want %v", i, got[i].UUID, tt.want[i].UUID)
				}
			}
		})
	}
}
