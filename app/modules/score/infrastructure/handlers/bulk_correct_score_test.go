package scorehandlers

import (
	"context"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
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

	testBulkPayload := &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{
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

	tests := []struct {
		name           string
		mockSetup      func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService
		payload        *sharedevents.ScoreBulkUpdateRequestedPayloadV1
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle bulk score updates with all succeeding",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService {
				mockSvc := scoremocks.NewMockService(ctrl)

				successPayload1 := &sharedevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				successPayload2 := &sharedevents.ScoreUpdatedPayloadV1{
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

				return mockSvc
			},
			payload: testBulkPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 4 {
					t.Fatalf("expected 4 results, got %d", len(results))
				}
				// First two: individual success events
				if results[0].Topic != sharedevents.ScoreUpdatedV1 {
					t.Errorf("expected first topic %s, got %s", sharedevents.ScoreUpdatedV1, results[0].Topic)
				}
				if results[1].Topic != sharedevents.ScoreUpdatedV1 {
					t.Errorf("expected second topic %s, got %s", sharedevents.ScoreUpdatedV1, results[1].Topic)
				}
				// Third: aggregate event
				if results[2].Topic != sharedevents.ScoreBulkUpdatedV1 {
					t.Errorf("expected third topic %s, got %s", sharedevents.ScoreBulkUpdatedV1, results[2].Topic)
				}
				aggPayload, ok := results[2].Payload.(*sharedevents.ScoreBulkUpdatedPayloadV1)
				if !ok {
					t.Fatalf("unexpected aggregate payload type: got %T", results[2].Payload)
				}
				if aggPayload.AppliedCount != 2 {
					t.Errorf("expected AppliedCount 2, got %d", aggPayload.AppliedCount)
				}
				if aggPayload.FailedCount != 0 {
					t.Errorf("expected FailedCount 0, got %d", aggPayload.FailedCount)
				}
				// Fourth: reprocess request
				if results[3].Topic != sharedevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected fourth topic %s, got %s", sharedevents.ProcessRoundScoresRequestedV1, results[3].Topic)
				}
			},
		},
		{
			name: "Nil payload",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService {
				return scoremocks.NewMockService(ctrl)
			},
			payload:        nil,
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "Service returns system error on first update",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService {
				mockSvc := scoremocks.NewMockService(ctrl)

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

				return mockSvc
			},
			payload:        testBulkPayload,
			wantErr:        true,
			expectedErrMsg: "system error during bulk score update",
		},
		{
			name: "All updates fail - no reprocessing",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService {
				mockSvc := scoremocks.NewMockService(ctrl)

				failurePayload1 := &sharedevents.ScoreUpdateFailedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Reason:  "score not found",
				}

				failurePayload2 := &sharedevents.ScoreUpdateFailedPayloadV1{
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

				return mockSvc
			},
			payload: testBulkPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 3 {
					t.Fatalf("expected 3 results, got %d", len(results))
				}
				// Two failure events
				if results[0].Topic != sharedevents.ScoreUpdateFailedV1 {
					t.Errorf("expected first topic %s, got %s", sharedevents.ScoreUpdateFailedV1, results[0].Topic)
				}
				if results[1].Topic != sharedevents.ScoreUpdateFailedV1 {
					t.Errorf("expected second topic %s, got %s", sharedevents.ScoreUpdateFailedV1, results[1].Topic)
				}
				// Aggregate message only (no reprocess since applied = 0)
				if results[2].Topic != sharedevents.ScoreBulkUpdatedV1 {
					t.Errorf("expected third topic %s, got %s", sharedevents.ScoreBulkUpdatedV1, results[2].Topic)
				}
				aggPayload, ok := results[2].Payload.(*sharedevents.ScoreBulkUpdatedPayloadV1)
				if !ok {
					t.Fatalf("unexpected aggregate payload type: got %T", results[2].Payload)
				}
				if aggPayload.AppliedCount != 0 {
					t.Errorf("expected AppliedCount 0, got %d", aggPayload.AppliedCount)
				}
				if aggPayload.FailedCount != 2 {
					t.Errorf("expected FailedCount 2, got %d", aggPayload.FailedCount)
				}
			},
		},
		{
			name: "GetScoresForRound fails after successful updates",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService {
				mockSvc := scoremocks.NewMockService(ctrl)

				successPayload1 := &sharedevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				successPayload2 := &sharedevents.ScoreUpdatedPayloadV1{
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

				mockSvc.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					nil, fmt.Errorf("db error"),
				)

				return mockSvc
			},
			payload: testBulkPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 3 {
					t.Fatalf("expected 3 results (no reprocess due to error), got %d", len(results))
				}
				// Two success events + aggregate (no reprocess message since GetScoresForRound failed)
				if results[0].Topic != sharedevents.ScoreUpdatedV1 {
					t.Errorf("expected first topic %s, got %s", sharedevents.ScoreUpdatedV1, results[0].Topic)
				}
				if results[1].Topic != sharedevents.ScoreUpdatedV1 {
					t.Errorf("expected second topic %s, got %s", sharedevents.ScoreUpdatedV1, results[1].Topic)
				}
				if results[2].Topic != sharedevents.ScoreBulkUpdatedV1 {
					t.Errorf("expected third topic %s, got %s", sharedevents.ScoreBulkUpdatedV1, results[2].Topic)
				}
			},
		},
		{
			name: "Mixed success and failure",
			mockSetup: func(t *testing.T, ctrl *gomock.Controller) *scoremocks.MockService {
				mockSvc := scoremocks.NewMockService(ctrl)

				successPayload1 := &sharedevents.ScoreUpdatedPayloadV1{
					GuildID: testGuildID,
					RoundID: testRoundID,
					UserID:  userID1,
					Score:   score1,
				}

				failurePayload2 := &sharedevents.ScoreUpdateFailedPayloadV1{
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
					scoreservice.ScoreOperationResult{Failure: failurePayload2},
					nil,
				)

				mockSvc.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(
					[]sharedtypes.ScoreInfo{
						{UserID: userID1, Score: score1, TagNumber: &tag1},
					},
					nil,
				)

				return mockSvc
			},
			payload: testBulkPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 4 {
					t.Fatalf("expected 4 results, got %d", len(results))
				}
				if results[0].Topic != sharedevents.ScoreUpdatedV1 {
					t.Errorf("expected first topic %s, got %s", sharedevents.ScoreUpdatedV1, results[0].Topic)
				}
				if results[1].Topic != sharedevents.ScoreUpdateFailedV1 {
					t.Errorf("expected second topic %s, got %s", sharedevents.ScoreUpdateFailedV1, results[1].Topic)
				}
				if results[2].Topic != sharedevents.ScoreBulkUpdatedV1 {
					t.Errorf("expected third topic %s, got %s", sharedevents.ScoreBulkUpdatedV1, results[2].Topic)
				}
				aggPayload, ok := results[2].Payload.(*sharedevents.ScoreBulkUpdatedPayloadV1)
				if !ok {
					t.Fatalf("unexpected aggregate payload type: got %T", results[2].Payload)
				}
				if aggPayload.AppliedCount != 1 {
					t.Errorf("expected AppliedCount 1, got %d", aggPayload.AppliedCount)
				}
				if aggPayload.FailedCount != 1 {
					t.Errorf("expected FailedCount 1, got %d", aggPayload.FailedCount)
				}
				if results[3].Topic != sharedevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected fourth topic %s, got %s", sharedevents.ProcessRoundScoresRequestedV1, results[3].Topic)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := tt.mockSetup(t, ctrl)

			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &scoremetrics.NoOpMetrics{}

			h := &ScoreHandlers{
				scoreService: mockSvc,
				logger:       logger,
				tracer:       tracer,
				metrics:      metrics,
			}

			ctx := context.Background()
			got, err := h.HandleBulkCorrectScoreRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleBulkCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleBulkCorrectScoreRequest() error = %v, expected %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
