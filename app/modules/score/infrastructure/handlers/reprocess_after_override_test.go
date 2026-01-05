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
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestScoreHandlers_HandleReprocessAfterScoreUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	userID1 := sharedtypes.DiscordID("user-1")
	userID2 := sharedtypes.DiscordID("user-2")
	score1 := sharedtypes.Score(10)
	score2 := sharedtypes.Score(12)
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)

	// Mock dependencies
	mockScoreService := scoremocks.NewMockService(ctrl)
	mockHelpers := mocks.NewMockHelpers(ctrl)

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &scoremetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		topic          string
		payload        interface{}
		metadata       map[string]string
		mockSetup      func()
		wantMsgCount   int
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:  "Successfully handle bulk success with reprocess",
			topic: scoreevents.ScoreBulkUpdatedV1,
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   2,
				FailedCount:    0,
				TotalRequested: 2,
				UserIDsApplied: []sharedtypes.DiscordID{userID1, userID2},
			},
			metadata: map[string]string{"topic": scoreevents.ScoreBulkUpdatedV1},
			mockSetup: func() {
				bulkPayload := &scoreevents.ScoreBulkUpdatedPayloadV1{
					GuildID:        testGuildID,
					RoundID:        testRoundID,
					AppliedCount:   2,
					FailedCount:    0,
					TotalRequested: 2,
					UserIDsApplied: []sharedtypes.DiscordID{userID1, userID2},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdatedPayloadV1) = *bulkPayload
						return nil
					},
				)

				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: userID1, Score: score1, TagNumber: &tag1},
					{UserID: userID2, Score: score2, TagNumber: &tag2},
				}, nil)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(), // ProcessRoundScoresRequestedPayloadV1
					scoreevents.ProcessRoundScoresRequestedV1,
				).Return(message.NewMessage("reprocess-msg", []byte("reprocess")), nil)
			},
			wantMsgCount: 1,
			wantErr:      false,
		},
		{
			name:  "Bulk success with zero applied count - skip reprocess",
			topic: scoreevents.ScoreBulkUpdatedV1,
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   0,
				FailedCount:    2,
				TotalRequested: 2,
				UserIDsApplied: []sharedtypes.DiscordID{},
			},
			metadata: map[string]string{"topic": scoreevents.ScoreBulkUpdatedV1},
			mockSetup: func() {
				bulkPayload := &scoreevents.ScoreBulkUpdatedPayloadV1{
					GuildID:        testGuildID,
					RoundID:        testRoundID,
					AppliedCount:   0,
					FailedCount:    2,
					TotalRequested: 2,
					UserIDsApplied: []sharedtypes.DiscordID{},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdatedPayloadV1) = *bulkPayload
						return nil
					},
				)
			},
			wantMsgCount: 0,
			wantErr:      false,
		},
		{
			name:  "Single success not part of bulk - trigger reprocess",
			topic: scoreevents.ScoreUpdatedV1,
			payload: &scoreevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
				Score:   score1,
			},
			metadata: map[string]string{"topic": scoreevents.ScoreUpdatedV1},
			mockSetup: func() {
				singlePayload := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdatedPayloadV1) = *singlePayload
						return nil
					},
				)

				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: userID1, Score: score1, TagNumber: &tag1},
				}, nil)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					scoreevents.ProcessRoundScoresRequestedV1,
				).Return(message.NewMessage("reprocess-msg", []byte("reprocess")), nil)
			},
			wantMsgCount: 1,
			wantErr:      false,
		},
		{
			name:  "Single success part of bulk batch - skip reprocess",
			topic: scoreevents.ScoreUpdatedV1,
			payload: &scoreevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
				Score:   score1,
			},
			metadata: map[string]string{
				"topic":           scoreevents.ScoreUpdatedV1,
				"override":        "true",
				"override_mode":   "bulk",
			},
			mockSetup: func() {
				singlePayload := &scoreevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreUpdatedPayloadV1) = *singlePayload
						return nil
					},
				).Times(0) // Should not unmarshal since we skip early
			},
			wantMsgCount: 0,
			wantErr:      false,
		},
		{
			name:  "Fail to unmarshal bulk payload",
			topic: scoreevents.ScoreBulkUpdatedV1,
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:      testGuildID,
				RoundID:      testRoundID,
				AppliedCount: 1,
			},
			metadata: map[string]string{"topic": scoreevents.ScoreBulkUpdatedV1},
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(
					fmt.Errorf("unmarshal failed"),
				)
			},
			wantMsgCount:   0,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal bulk success payload: unmarshal failed",
		},
		{
			name:  "Fail to unmarshal single payload",
			topic: scoreevents.ScoreUpdatedV1,
			payload: &scoreevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
				Score:   score1,
			},
			metadata: map[string]string{"topic": scoreevents.ScoreUpdatedV1},
			mockSetup: func() {
				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).Return(
					fmt.Errorf("unmarshal failed"),
				)
			},
			wantMsgCount:   0,
			wantErr:        true,
			expectedErrMsg: "failed to unmarshal success payload: unmarshal failed",
		},
		{
			name:  "GetScoresForRound fails",
			topic: scoreevents.ScoreBulkUpdatedV1,
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   1,
				FailedCount:    0,
				TotalRequested: 1,
				UserIDsApplied: []sharedtypes.DiscordID{userID1},
			},
			metadata: map[string]string{"topic": scoreevents.ScoreBulkUpdatedV1},
			mockSetup: func() {
				bulkPayload := &scoreevents.ScoreBulkUpdatedPayloadV1{
					GuildID:        testGuildID,
					RoundID:        testRoundID,
					AppliedCount:   1,
					FailedCount:    0,
					TotalRequested: 1,
					UserIDsApplied: []sharedtypes.DiscordID{userID1},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdatedPayloadV1) = *bulkPayload
						return nil
					},
				)

				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(nil, fmt.Errorf("db error"))
			},
			wantMsgCount:   0,
			wantErr:        true,
			expectedErrMsg: "failed to load stored scores for reprocess: db error",
		},
		{
			name:  "GetScoresForRound returns empty - skip reprocess",
			topic: scoreevents.ScoreBulkUpdatedV1,
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   1,
				FailedCount:    0,
				TotalRequested: 1,
				UserIDsApplied: []sharedtypes.DiscordID{userID1},
			},
			metadata: map[string]string{"topic": scoreevents.ScoreBulkUpdatedV1},
			mockSetup: func() {
				bulkPayload := &scoreevents.ScoreBulkUpdatedPayloadV1{
					GuildID:        testGuildID,
					RoundID:        testRoundID,
					AppliedCount:   1,
					FailedCount:    0,
					TotalRequested: 1,
					UserIDsApplied: []sharedtypes.DiscordID{userID1},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdatedPayloadV1) = *bulkPayload
						return nil
					},
				)

				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{}, nil)
			},
			wantMsgCount: 0,
			wantErr:      false,
		},
		{
			name:  "CreateResultMessage fails during reprocess",
			topic: scoreevents.ScoreBulkUpdatedV1,
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   1,
				FailedCount:    0,
				TotalRequested: 1,
				UserIDsApplied: []sharedtypes.DiscordID{userID1},
			},
			metadata: map[string]string{"topic": scoreevents.ScoreBulkUpdatedV1},
			mockSetup: func() {
				bulkPayload := &scoreevents.ScoreBulkUpdatedPayloadV1{
					GuildID:        testGuildID,
					RoundID:        testRoundID,
					AppliedCount:   1,
					FailedCount:    0,
					TotalRequested: 1,
					UserIDsApplied: []sharedtypes.DiscordID{userID1},
				}

				mockHelpers.EXPECT().UnmarshalPayload(gomock.Any(), gomock.Any()).DoAndReturn(
					func(msg *message.Message, out interface{}) error {
						*out.(*scoreevents.ScoreBulkUpdatedPayloadV1) = *bulkPayload
						return nil
					},
				)

				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: userID1, Score: score1, TagNumber: &tag1},
				}, nil)

				mockHelpers.EXPECT().CreateResultMessage(
					gomock.Any(),
					gomock.Any(),
					scoreevents.ProcessRoundScoresRequestedV1,
				).Return(nil, fmt.Errorf("create message failed"))
			},
			wantMsgCount:   0,
			wantErr:        true,
			expectedErrMsg: "failed to create reprocess request message: create message failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			// Build message with metadata
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage("test-id", payloadBytes)
			for key, val := range tt.metadata {
				msg.Metadata.Set(key, val)
			}

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

			got, err := h.HandleReprocessAfterScoreUpdate(msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleReprocessAfterScoreUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleReprocessAfterScoreUpdate() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if len(got) != tt.wantMsgCount {
				t.Errorf("HandleReprocessAfterScoreUpdate() returned %d messages, want %d", len(got), tt.wantMsgCount)
			}
		})
	}
}
