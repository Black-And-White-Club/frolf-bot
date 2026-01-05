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

func TestScoreHandlers_HandleBulkCorrectScoreRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	userID1 := sharedtypes.DiscordID("user-1")
	userID2 := sharedtypes.DiscordID("user-2")
	score1 := sharedtypes.Score(10)
	score2 := sharedtypes.Score(12)
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)

	testBulkPayload := &scoreevents.ScoreBulkUpdateRequestPayload{
		GuildID: testGuildID,
		RoundID: testRoundID,
		Updates: []scoreevents.ScoreUpdateRequestPayload{
			{
				GuildID:   testGuildID,
				RoundID:   testRoundID,
				UserID:    userID1,
				Score:     score1,
				TagNumber: &tag1,
			},
			{
				GuildID:   testGuildID,
				RoundID:   testRoundID,
				UserID:    userID2,
				Score:     score2,
				TagNumber: &tag2,
			},
		},
	}
	payloadBytes, _ := json.Marshal(testBulkPayload)
	testMsg := message.NewMessage("test-id", payloadBytes)
	invalidMsg := message.NewMessage("test-id", []byte("invalid json"))

	tests := []struct {
		name           string
		mockSetup      func(t *testing.T, ctrl *gomock.Controller) (*scoremocks.MockService, *mocks.MockHelpers)
		msg            *message.Message
		wantMsgCount   int
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successfully handle bulk score updates with all succeeding",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) (*scoremocks.MockService, *mocks.MockHelpers) {
				mockSvc := scoremocks.NewMockService(ctrl)
				mockHelper := mocks.NewMockHelpers(ctrl)

				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdateRequestPayload) = *testBulkPayload
						return nil
					},
				)

				successPayload1 := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				successPayload2 := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID2,
					Score:   score2,
				}

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID1,
					score1,
					&tag1,
				).Return(
					scoreservice.ScoreOperationResult{Success: successPayload1},
					nil,
				)

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID2,
					score2,
					&tag2,
				).Return(
					scoreservice.ScoreOperationResult{Success: successPayload2},
					nil,
				)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					successPayload1,
					scoreevents.ScoreUpdatedV1,
				).Return(message.NewMessage("msg1", []byte("data")), nil)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					successPayload2,
					scoreevents.ScoreUpdatedV1,
				).Return(message.NewMessage("msg2", []byte("data")), nil)

				mockSvc.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					[]sharedtypes.ScoreInfo{
						{UserID: userID1, Score: score1, TagNumber: &tag1},
						{UserID: userID2, Score: score2, TagNumber: &tag2},
					},
					nil,
				)

				// Aggregate message
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // ScoreBulkUpdateSuccessPayload
					scoreevents.ScoreBulkUpdatedV1,
				).Return(message.NewMessage("msg3", []byte("data")), nil)

				// Reprocess message
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // ProcessRoundScoresRequestedPayloadV1
					scoreevents.ProcessRoundScoresRequestedV1,
				).Return(message.NewMessage("msg4", []byte("data")), nil)

				return mockSvc, mockHelper
			},
			msg:          testMsg,
			wantMsgCount: 4,
			wantErr:      false,
		},
		{
			name: "Fail to unmarshal payload",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) (*scoremocks.MockService, *mocks.MockHelpers) {
				mockSvc := scoremocks.NewMockService(ctrl)
				mockHelper := mocks.NewMockHelpers(ctrl)

				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(fmt.Errorf("invalid payload"))

				return mockSvc, mockHelper
			},
			msg:            invalidMsg,
			wantMsgCount:   0,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal payload: invalid payload",
		},
		{
			name: "Service returns system error on first update",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) (*scoremocks.MockService, *mocks.MockHelpers) {
				mockSvc := scoremocks.NewMockService(ctrl)
				mockHelper := mocks.NewMockHelpers(ctrl)

				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdateRequestPayload) = *testBulkPayload
						return nil
					},
				)

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID1,
					score1,
					&tag1,
				).Return(
					scoreservice.ScoreOperationResult{},
					fmt.Errorf("system error"),
				)

				return mockSvc, mockHelper
			},
			msg:            testMsg,
			wantMsgCount:   0,
			wantErr:        true,
			expectedErrMsg: "system error during bulk score update for user user-1: system error",
		},
		{
			name: "All updates fail - no reprocessing",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) (*scoremocks.MockService, *mocks.MockHelpers) {
				mockSvc := scoremocks.NewMockService(ctrl)
				mockHelper := mocks.NewMockHelpers(ctrl)

				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdateRequestPayload) = *testBulkPayload
						return nil
					},
				)

				failurePayload1 := &scoreevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Reason:  "score not found",
				}

				failurePayload2 := &scoreevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID2,
					Reason:  "score not found",
				}

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID1,
					score1,
					&tag1,
				).Return(
					scoreservice.ScoreOperationResult{Failure: failurePayload1},
					nil,
				)

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID2,
					score2,
					&tag2,
				).Return(
					scoreservice.ScoreOperationResult{Failure: failurePayload2},
					nil,
				)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload1,
					scoreevents.ScoreUpdateFailedV1,
				).Return(message.NewMessage("fail1", []byte("data")), nil)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					failurePayload2,
					scoreevents.ScoreUpdateFailedV1,
				).Return(message.NewMessage("fail2", []byte("data")), nil)

				// Aggregate message only (no reprocess since applied = 0)
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // ScoreBulkUpdateSuccessPayload
					scoreevents.ScoreBulkUpdatedV1,
				).Return(message.NewMessage("msg3", []byte("data")), nil)

				return mockSvc, mockHelper
			},
			msg:          testMsg,
			wantMsgCount: 3,
			wantErr:      false,
		},
		{
			name: "GetScoresForRound fails after successful updates",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) (*scoremocks.MockService, *mocks.MockHelpers) {
				mockSvc := scoremocks.NewMockService(ctrl)
				mockHelper := mocks.NewMockHelpers(ctrl)

				mockHelper.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdateRequestPayload) = *testBulkPayload
						return nil
					},
				)

				successPayload1 := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				successPayload2 := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID2,
					Score:   score2,
				}

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID1,
					score1,
					&tag1,
				).Return(
					scoreservice.ScoreOperationResult{Success: successPayload1},
					nil,
				)

				mockSvc.EXPECT().CorrectScore(
					gomock.Any(),
					testGuildID,
					testRoundID,
					userID2,
					score2,
					&tag2,
				).Return(
					scoreservice.ScoreOperationResult{Success: successPayload2},
					nil,
				)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					successPayload1,
					scoreevents.ScoreUpdatedV1,
				).Return(message.NewMessage("msg1", []byte("data")), nil)

				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					successPayload2,
					scoreevents.ScoreUpdatedV1,
				).Return(message.NewMessage("msg2", []byte("data")), nil)

				mockSvc.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					nil, fmt.Errorf("db error"),
				)

				// Aggregate message (no reprocess message since GetScoresForRound failed)
				mockHelper.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // ScoreBulkUpdateSuccessPayload
					scoreevents.ScoreBulkUpdatedV1,
				).Return(message.NewMessage("msg3", []byte("data")), nil)

				return mockSvc, mockHelper
			},
			msg:          testMsg,
			wantMsgCount: 3,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc, mockHelper := tt.mockSetup(t, ctrl)

			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &scoremetrics.NoOpMetrics{}

			h := &ScoreHandlers{
				scoreService: mockSvc,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
				Helpers:      mockHelper,
				handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
					return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, mockHelper)
				},
			}

			got, err := h.HandleBulkCorrectScoreRequest(tt.msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleBulkCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleBulkCorrectScoreRequest() error = %v, expected %v", err, tt.expectedErrMsg)
			}

			if len(got) != tt.wantMsgCount {
				t.Errorf("HandleBulkCorrectScoreRequest() returned %d messages, want %d", len(got), tt.wantMsgCount)
			}
		})
	}
}
