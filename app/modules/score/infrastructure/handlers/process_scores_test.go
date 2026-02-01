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
						},
					}, nil
				}
			},
			wantErr: false,
			checkResults: func(t *testing.T, res []handlerwrapper.Result) {
				if len(res) != 1 {
					t.Fatalf("expected 1 result, got %d", len(res))
				}
				if res[0].Topic != sharedevents.LeaderboardBatchTagAssignmentRequestedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.LeaderboardBatchTagAssignmentRequestedV1, res[0].Topic)
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
