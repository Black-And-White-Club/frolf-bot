package scorehandlers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/google/uuid"
)

func ptrTagNumber(value sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &value
}

func TestScoreHandlers_HandleProcessRoundScoresRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("user1")
	testScore := sharedtypes.Score(72)
	testTagNumber := sharedtypes.TagNumber(1)

	basePayload := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
		GuildID:   testGuildID,
		RoundID:   testRoundID,
		Scores:    []sharedtypes.ScoreInfo{{UserID: testUserID, Score: testScore, TagNumber: &testTagNumber}},
		Overwrite: true,
		RoundMode: sharedtypes.RoundModeSingles,
	}

	tests := []struct {
		name           string
		payload        *sharedevents.ProcessRoundScoresRequestedPayloadV1
		setupFake      func(*FakeScoreService)
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name:    "Successfully handle ProcessRoundScoresRequest - Singles (Triggers Leaderboard)",
			payload: basePayload,
			setupFake: func(f *FakeScoreService) {
				f.ProcessRoundScoresFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
					return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{
						Success: &scoreservice.ProcessRoundScoresResult{
							TagMappings: []sharedtypes.TagMapping{
								{DiscordID: testUserID, TagNumber: testTagNumber},
							},
							FinishRanksByDiscordID: map[sharedtypes.DiscordID]int{
								testUserID: 1,
							},
						},
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, res []handlerwrapper.Result) {
				if len(res) != 1 {
					t.Fatalf("expected 1 result, got %d", len(res))
				}
				if res[0].Topic != sharedevents.LeaderboardBatchTagAssignmentRequestedV2 {
					t.Errorf("expected topic %s, got %s", sharedevents.LeaderboardBatchTagAssignmentRequestedV2, res[0].Topic)
				}
				batchPayload, ok := res[0].Payload.(*sharedevents.BatchTagAssignmentRequestedPayloadV1)
				if !ok {
					t.Fatalf("expected *BatchTagAssignmentRequestedPayloadV1, got %T", res[0].Payload)
				}
				if len(batchPayload.Assignments) != 1 {
					t.Fatalf("expected 1 assignment, got %d", len(batchPayload.Assignments))
				}
				if batchPayload.Assignments[0].FinishRank != 1 {
					t.Errorf("expected FinishRank=1, got %d", batchPayload.Assignments[0].FinishRank)
				}
			},
		},
		{
			name: "Singles flow skips ranked players missing both resolved and original tags but preserves finish rank for tagged players",
			payload: &sharedevents.ProcessRoundScoresRequestedPayloadV1{
				GuildID:   testGuildID,
				RoundID:   testRoundID,
				Overwrite: true,
				RoundMode: sharedtypes.RoundModeSingles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: "user-a", Score: 50, TagNumber: ptrTagNumber(7)},
					{UserID: "user-b", Score: 51},
					{UserID: "", Score: 52},
				},
			},
			setupFake: func(f *FakeScoreService) {
				f.ProcessRoundScoresFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
					return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{
						Success: &scoreservice.ProcessRoundScoresResult{
							FinishRanksByDiscordID: map[sharedtypes.DiscordID]int{
								"user-a": 1,
								"user-b": 2,
							},
						},
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, res []handlerwrapper.Result) {
				if len(res) != 1 {
					t.Fatalf("expected 1 result, got %d", len(res))
				}
				batchPayload, ok := res[0].Payload.(*sharedevents.BatchTagAssignmentRequestedPayloadV1)
				if !ok {
					t.Fatalf("expected *BatchTagAssignmentRequestedPayloadV1, got %T", res[0].Payload)
				}
				if len(batchPayload.Assignments) != 1 {
					t.Fatalf("expected 1 assignment, got %d", len(batchPayload.Assignments))
				}
				if batchPayload.Assignments[0].UserID != "user-a" {
					t.Fatalf("unexpected assignment user: %+v", batchPayload.Assignments[0])
				}
				if batchPayload.Assignments[0].TagNumber != 7 {
					t.Errorf("expected TagNumber=7, got %d", batchPayload.Assignments[0].TagNumber)
				}
				if batchPayload.Assignments[0].FinishRank != 1 {
					t.Errorf("expected FinishRank=1, got %d", batchPayload.Assignments[0].FinishRank)
				}
			},
		},
		{
			name: "Successfully handle ProcessRoundScoresRequest - Teams (Terminates Early)",
			payload: func() *sharedevents.ProcessRoundScoresRequestedPayloadV1 {
				p := *basePayload
				p.RoundMode = sharedtypes.RoundModeDoubles
				return &p
			}(),
			setupFake: func(f *FakeScoreService) {
				f.ProcessRoundScoresFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
					return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{
						Success: &scoreservice.ProcessRoundScoresResult{},
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, res []handlerwrapper.Result) {
				if len(res) != 0 {
					t.Errorf("expected 0 results for non-singles mode, got %d", len(res))
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
			name:    "Service failure in ProcessRoundScores (Domain Error)",
			payload: basePayload,
			setupFake: func(f *FakeScoreService) {
				f.ProcessRoundScoresFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
					err := errors.New("internal service error")
					return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{
						Failure: &err,
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, res []handlerwrapper.Result) {
				if len(res) != 1 {
					t.Fatalf("expected 1 result, got %d", len(res))
				}
				if res[0].Topic != sharedevents.ProcessRoundScoresFailedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.ProcessRoundScoresFailedV1, res[0].Topic)
				}
				fail := res[0].Payload.(*sharedevents.ProcessRoundScoresFailedPayloadV1)
				if fail.Reason != "internal service error" {
					t.Errorf("expected reason 'internal service error', got %s", fail.Reason)
				}
			},
		},
		{
			name:    "Service returns direct error (Infrastructure Error)",
			payload: basePayload,
			setupFake: func(f *FakeScoreService) {
				f.ProcessRoundScoresFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
					return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{}, fmt.Errorf("direct service error")
				}
			},
			wantErr:        true,
			expectedErrMsg: "direct service error",
		},
		{
			name: "FinishRank propagated to batch event for tied players",
			payload: &sharedevents.ProcessRoundScoresRequestedPayloadV1{
				GuildID:   testGuildID,
				RoundID:   testRoundID,
				Overwrite: true,
				RoundMode: sharedtypes.RoundModeSingles,
				Scores: []sharedtypes.ScoreInfo{
					{UserID: "user-a", Score: 50, TagNumber: ptrTagNumber(1)},
					{UserID: "user-b", Score: 50, TagNumber: ptrTagNumber(5)},
				},
			},
			setupFake: func(f *FakeScoreService) {
				tag1 := sharedtypes.TagNumber(1)
				tag2 := sharedtypes.TagNumber(5)
				f.ProcessRoundScoresFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, overwrite bool) (results.OperationResult[scoreservice.ProcessRoundScoresResult, error], error) {
					return results.OperationResult[scoreservice.ProcessRoundScoresResult, error]{
						Success: &scoreservice.ProcessRoundScoresResult{
							TagMappings: []sharedtypes.TagMapping{
								{DiscordID: "user-a", TagNumber: tag1},
								{DiscordID: "user-b", TagNumber: tag2},
							},
							// Both players tied at rank 1
							FinishRanksByDiscordID: map[sharedtypes.DiscordID]int{
								"user-a": 1,
								"user-b": 1,
							},
						},
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, res []handlerwrapper.Result) {
				if len(res) != 1 {
					t.Fatalf("expected 1 result, got %d", len(res))
				}
				batchPayload, ok := res[0].Payload.(*sharedevents.BatchTagAssignmentRequestedPayloadV1)
				if !ok {
					t.Fatalf("expected *BatchTagAssignmentRequestedPayloadV1, got %T", res[0].Payload)
				}
				rankByUser := make(map[sharedtypes.DiscordID]int, len(batchPayload.Assignments))
				for _, a := range batchPayload.Assignments {
					rankByUser[a.UserID] = a.FinishRank
				}
				if rankByUser["user-a"] != 1 {
					t.Errorf("user-a: expected FinishRank=1 in batch event, got %d", rankByUser["user-a"])
				}
				if rankByUser["user-b"] != 1 {
					t.Errorf("user-b: expected FinishRank=1 in batch event, got %d", rankByUser["user-b"])
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

			got, err := h.HandleProcessRoundScoresRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleProcessRoundScoresRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleProcessRoundScoresRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
