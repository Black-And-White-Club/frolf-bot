package scorehandlers

import (
	"context"
	"errors"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

func TestScoreHandlers_HandleReprocessAfterBulkScoreUpdate(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	userID1 := sharedtypes.DiscordID("user-1")
	userID2 := sharedtypes.DiscordID("user-2")
	score1 := sharedtypes.Score(10)
	score2 := sharedtypes.Score(12)
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(2)

	tests := []struct {
		name           string
		payload        *sharedevents.ScoreBulkUpdatedPayloadV1
		setupFake      func(*FakeScoreService)
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle bulk success with reprocess",
			payload: &sharedevents.ScoreBulkUpdatedPayloadV1{
				GuildID:        testGuildID,
				RoundID:        testRoundID,
				AppliedCount:   2,
				TotalRequested: 2,
				UserIDsApplied: []sharedtypes.DiscordID{userID1, userID2},
			},
			setupFake: func(f *FakeScoreService) {
				f.GetScoresForRoundFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{
						{UserID: userID1, Score: score1, TagNumber: &tag1},
						{UserID: userID2, Score: score2, TagNumber: &tag2},
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != sharedevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.ProcessRoundScoresRequestedV1, results[0].Topic)
				}
				reprocessPayload := results[0].Payload.(*sharedevents.ProcessRoundScoresRequestedPayloadV1)
				if len(reprocessPayload.Scores) != 2 {
					t.Errorf("expected 2 scores, got %d", len(reprocessPayload.Scores))
				}
			},
		},
		{
			name: "Bulk success with zero applied count - skip reprocess",
			payload: &sharedevents.ScoreBulkUpdatedPayloadV1{
				AppliedCount: 0,
			},
			setupFake: nil,
			wantErr:   false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 0 {
					t.Fatalf("expected 0 results, got %d", len(results))
				}
			},
		},
		{
			name:           "Nil payload",
			payload:        nil,
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "GetScoresForRound fails",
			payload: &sharedevents.ScoreBulkUpdatedPayloadV1{
				AppliedCount: 1,
				GuildID:      testGuildID,
				RoundID:      testRoundID,
			},
			setupFake: func(f *FakeScoreService) {
				f.GetScoresForRoundFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr:        true,
			expectedErrMsg: "failed to load stored scores for reprocess",
		},
		{
			name: "GetScoresForRound returns empty - skip reprocess",
			payload: &sharedevents.ScoreBulkUpdatedPayloadV1{
				AppliedCount: 1,
				GuildID:      testGuildID,
				RoundID:      testRoundID,
			},
			setupFake: func(f *FakeScoreService) {
				f.GetScoresForRoundFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 0 {
					t.Errorf("expected 0 results, got %d", len(results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeScoreService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &ScoreHandlers{
				service: fakeSvc,
				helpers: nil,
			}

			got, err := h.HandleReprocessAfterBulkScoreUpdate(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleReprocessAfterBulkScoreUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleReprocessAfterBulkScoreUpdate() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}

func TestScoreHandlers_HandleReprocessAfterSingleScoreUpdate(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	userID1 := sharedtypes.DiscordID("user-1")
	score1 := sharedtypes.Score(10)

	tests := []struct {
		name           string
		payload        *sharedevents.ScoreUpdatedPayloadV1
		setupFake      func(*FakeScoreService)
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Single score update triggers reprocess",
			payload: &sharedevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
			},
			setupFake: func(f *FakeScoreService) {
				f.GetScoresForRoundFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{{UserID: userID1, Score: score1}}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != sharedevents.ProcessRoundScoresRequestedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.ProcessRoundScoresRequestedV1, results[0].Topic)
				}
			},
		},
		{
			name:           "Nil payload",
			payload:        nil,
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "GetScoresForRound fails",
			payload: &sharedevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
			},
			setupFake: func(f *FakeScoreService) {
				f.GetScoresForRoundFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr:        true,
			expectedErrMsg: "failed to load stored scores for reprocess",
		},
		{
			name: "Empty scores - skip reprocess",
			payload: &sharedevents.ScoreUpdatedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
			},
			setupFake: func(f *FakeScoreService) {
				f.GetScoresForRoundFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 0 {
					t.Errorf("expected 0 results, got %d", len(results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeScoreService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &ScoreHandlers{
				service: fakeSvc,
				helpers: nil,
			}

			got, err := h.HandleReprocessAfterSingleScoreUpdate(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleReprocessAfterSingleScoreUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleReprocessAfterSingleScoreUpdate() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
