package scorehandlers

import (
	"context"
	"fmt"
	"testing"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/google/uuid"
)

func TestScoreHandlers_HandleCorrectScoreRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTagNumber := sharedtypes.TagNumber(1)

	testPayload := &sharedevents.ScoreUpdateRequestedPayloadV1{
		GuildID:   testGuildID,
		RoundID:   testRoundID,
		UserID:    testUserID,
		Score:     testScore,
		TagNumber: &testTagNumber,
	}

	tests := []struct {
		name           string
		setupFake      func(*FakeScoreService)
		payload        *sharedevents.ScoreUpdateRequestedPayloadV1
		wantErr        bool
		expectedErrMsg string
		checkResults   func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "Successfully handle CorrectScoreRequest",
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					return scoreservice.ScoreOperationResult{
						Success: &sharedtypes.ScoreInfo{UserID: uID, Score: s, TagNumber: tag},
					}, nil
				}
			},
			payload: testPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != sharedevents.ScoreUpdatedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.ScoreUpdatedV1, results[0].Topic)
				}
			},
		},
		{
			name:           "Nil payload",
			setupFake:      nil,
			payload:        nil,
			wantErr:        true,
			expectedErrMsg: "payload is nil",
		},
		{
			name: "Service failure in CorrectScore (Domain Error)",
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					domainErr := fmt.Errorf("internal service error")
					return scoreservice.ScoreOperationResult{
						Failure: &domainErr,
					}, nil
				}
			},
			payload: testPayload,
			wantErr: false,
			checkResults: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				if results[0].Topic != sharedevents.ScoreUpdateFailedV1 {
					t.Errorf("expected topic %s, got %s", sharedevents.ScoreUpdateFailedV1, results[0].Topic)
				}
				fail, _ := results[0].Payload.(*sharedevents.ScoreUpdateFailedPayloadV1)
				if fail.Reason != "internal service error" {
					t.Errorf("expected reason 'internal service error', got %s", fail.Reason)
				}
			},
		},
		{
			name: "Service returns error (Infrastructure Error)",
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					return scoreservice.ScoreOperationResult{}, fmt.Errorf("service error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "service error",
		},
		{
			name: "Unknown result from CorrectScore",
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					return scoreservice.ScoreOperationResult{}, nil
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "unexpected result from service: neither success nor failure",
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

			got, err := h.HandleCorrectScoreRequest(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleCorrectScoreRequest() error = %v, expectedErrMsg %v", err, tt.expectedErrMsg)
			}

			if !tt.wantErr && tt.checkResults != nil {
				tt.checkResults(t, got)
			}
		})
	}
}
