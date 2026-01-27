package scorehandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/google/uuid"
)

func TestScoreHandlers_HandleBulkCorrectScoreRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	userID1 := sharedtypes.DiscordID("user-1")
	score1 := sharedtypes.Score(10)

	tests := []struct {
		name      string
		ctx       context.Context
		payload   *sharedevents.ScoreBulkUpdateRequestedPayloadV1
		setupFake func(*FakeScoreService)
		wantErr   bool
		checkRes  func(t *testing.T, results []handlerwrapper.Result)
	}{
		{
			name: "success - bulk scores corrected",
			ctx:  context.WithValue(context.WithValue(context.Background(), "channel_id", "channel-1"), "message_id", "message-1"),
			payload: &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{
					{UserID: userID1, Score: score1},
				},
			},
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					// Correctly return a ScoreInfo pointer for the Success field
					return scoreservice.ScoreOperationResult{
						Success: &sharedtypes.ScoreInfo{
							UserID:    uID,
							Score:     s,
							TagNumber: tag,
						},
					}, nil
				}
			},
			wantErr: false,
			checkRes: func(t *testing.T, results []handlerwrapper.Result) {
				if len(results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(results))
				}
				bulk := results[0].Payload.(*roundevents.ScoreBulkUpdateRequestPayloadV1)
				if bulk.MessageID != "message-1" {
					t.Errorf("expected message_id message-1, got %s", bulk.MessageID)
				}
			},
		},
		{
			name: "success - uses discord_message_id fallback",
			ctx:  context.WithValue(context.Background(), "discord_message_id", "discord-msg-123"),
			payload: &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
				GuildID: testGuildID,
				RoundID: testRoundID,
				Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{{UserID: userID1, Score: score1}},
			},
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					// Return ScoreInfo instead of ScoreUpdatedPayloadV1
					return scoreservice.ScoreOperationResult{
						Success: &sharedtypes.ScoreInfo{
							UserID:    uID,
							Score:     s,
							TagNumber: tag,
						},
					}, nil
				}
			},
			wantErr: false,
			checkRes: func(t *testing.T, res []handlerwrapper.Result) {
				bulk := res[0].Payload.(*roundevents.ScoreBulkUpdateRequestPayloadV1)
				if bulk.MessageID != "discord-msg-123" {
					t.Errorf("expected message_id discord-msg-123, got %s", bulk.MessageID)
				}
			},
		},
		{
			name:    "error - nil payload",
			ctx:     context.Background(),
			payload: nil,
			wantErr: true,
		},
		{
			name: "error - service infrastructure failure",
			ctx:  context.Background(),
			payload: &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
				Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{{UserID: userID1}},
			},
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					return scoreservice.ScoreOperationResult{}, errors.New("db connection lost")
				}
			},
			wantErr: true,
		},
		{
			name: "error - domain failure (result.Failure)",
			ctx:  context.Background(),
			payload: &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
				Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{{UserID: userID1}},
			},
			setupFake: func(f *FakeScoreService) {
				f.CorrectScoreFunc = func(ctx context.Context, gID sharedtypes.GuildID, rID sharedtypes.RoundID, uID sharedtypes.DiscordID, s sharedtypes.Score, tag *sharedtypes.TagNumber) (scoreservice.ScoreOperationResult, error) {
					err := errors.New("round is locked")
					return scoreservice.ScoreOperationResult{Failure: &err}, nil
				}
			},
			wantErr: true,
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

			results, err := h.HandleBulkCorrectScoreRequest(tt.ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("HandleBulkCorrectScoreRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkRes != nil {
				tt.checkRes(t, results)
			}
		})
	}
}
