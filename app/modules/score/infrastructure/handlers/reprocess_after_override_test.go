package scorehandlers

import (
	"context"
	"fmt"
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
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

	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := &scoremetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockSetup      func()
		payload        interface{}
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle bulk success with reprocess",
			mockSetup: func() {
				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: userID1, Score: score1, TagNumber: &tag1},
					{UserID: userID2, Score: score2, TagNumber: &tag2},
				}, nil)
			},
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   2,
				FailedCount:    0,
				TotalRequested: 2,
				UserIDsApplied: []sharedtypes.DiscordID{userID1, userID2},
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != scoreevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected topic %s, got %s", scoreevents.ProcessRoundScoresRequestedV1, results[0].Topic)
				}
				reprocessPayload, ok := results[0].Payload.(*scoreevents.ProcessRoundScoresRequestedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: got %T", results[0].Payload)
				}
				if len(reprocessPayload.Scores) != 2 {
					t.Errorf("expected 2 scores, got %d", len(reprocessPayload.Scores))
				}
			},
		},
		{
			name: "Bulk success with zero applied count - skip reprocess",
			mockSetup: func() {
				// No service calls expected
			},
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   0,
				FailedCount:    2,
				TotalRequested: 2,
				UserIDsApplied: []sharedtypes.DiscordID{},
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 0 {
					t.Fatalf("expected 0 results (skip reprocess), got %d", len(results))
				}
			},
		},
		{
			name: "Single success not part of bulk - trigger reprocess",
			mockSetup: func() {
				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{
					{UserID: userID1, Score: score1, TagNumber: &tag1},
				}, nil)
			},
			payload: &scoreevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
				Score:   score1,
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != scoreevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected topic %s, got %s", scoreevents.ProcessRoundScoresRequestedV1, results[0].Topic)
				}
			},
		},
		{
			name: "Nil payload",
			mockSetup: func() {
				// No expectations
			},
			payload:        nil,
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "GetScoresForRound fails",
			mockSetup: func() {
				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return(nil, fmt.Errorf("db error"))
			},
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   1,
				FailedCount:    0,
				TotalRequested: 1,
				UserIDsApplied: []sharedtypes.DiscordID{userID1},
			},
			wantErr:        true,
			expectedErrMsg: "failed to load stored scores for reprocess",
		},
		{
			name: "GetScoresForRound returns empty - skip reprocess",
			mockSetup: func() {
				mockScoreService.EXPECT().GetScoresForRound(
					gomock.Any(),
					testGuildID,
					testRoundID,
				).Return([]sharedtypes.ScoreInfo{}, nil)
			},
			payload: &scoreevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   1,
				FailedCount:    0,
				TotalRequested: 1,
				UserIDsApplied: []sharedtypes.DiscordID{userID1},
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 0 {
					t.Fatalf("expected 0 results (no scores to reprocess), got %d", len(results))
				}
			},
		},
		{
			name: "Unexpected payload type",
			mockSetup: func() {
				// No expectations
			},
			payload:        "wrong type",
			wantErr:        true,
			expectedErrMsg: "unexpected payload type",
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
			}

			ctx := context.Background()
			got, err := h.HandleReprocessAfterScoreUpdate(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleReprocessAfterScoreUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleReprocessAfterScoreUpdate() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
