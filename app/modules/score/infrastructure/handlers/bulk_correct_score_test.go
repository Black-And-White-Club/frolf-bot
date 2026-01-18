package scorehandlers

import (
	"context"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoremocks "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestScoreHandlers_HandleBulkCorrectScoreRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	userID1 := sharedtypes.DiscordID("user-1")
	userID2 := sharedtypes.DiscordID("user-2")
	score1 := sharedtypes.Score(10)
	score2 := sharedtypes.Score(12)

	bulkPayload := &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
		GuildID: testGuildID,
		RoundID: testRoundID,
		Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{
			{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID1,
				Score:   score1,
			},
			{
				GuildID: testGuildID,
				RoundID: testRoundID,
				UserID:  userID2,
				Score:   score2,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// no-op observability in handler tests

	mockSvc := scoremocks.NewMockService(ctrl)
	mockSvc.EXPECT().CorrectScore(gomock.Any(), testGuildID, testRoundID, userID1, score1, nil).
		Return(scoreservice.ScoreOperationResult{Success: &sharedevents.ScoreUpdatedPayloadV1{GuildID: testGuildID, RoundID: testRoundID, UserID: userID1, Score: score1}}, nil)
	mockSvc.EXPECT().CorrectScore(gomock.Any(), testGuildID, testRoundID, userID2, score2, nil).
		Return(scoreservice.ScoreOperationResult{Success: &sharedevents.ScoreUpdatedPayloadV1{GuildID: testGuildID, RoundID: testRoundID, UserID: userID2, Score: score2}}, nil)

	h := &ScoreHandlers{
		service: mockSvc,
		helpers: nil,
	}

	ctx := context.WithValue(context.Background(), "channel_id", "channel-1")
	ctx = context.WithValue(ctx, "message_id", "message-1")

	results, err := h.HandleBulkCorrectScoreRequest(ctx, bulkPayload)
	if err != nil {
		t.Fatalf("HandleBulkCorrectScoreRequest() unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Topic != roundevents.RoundScoreBulkUpdateRequestedV1 {
		t.Fatalf("expected topic %s, got %s", roundevents.RoundScoreBulkUpdateRequestedV1, results[0].Topic)
	}

	bulk, ok := results[0].Payload.(*roundevents.ScoreBulkUpdateRequestPayloadV1)
	if !ok {
		t.Fatalf("expected payload *ScoreBulkUpdateRequestPayloadV1, got %T", results[0].Payload)
	}
	if bulk.GuildID != testGuildID {
		t.Errorf("expected guild_id %s, got %s", testGuildID, bulk.GuildID)
	}
	if bulk.RoundID != testRoundID {
		t.Errorf("expected round_id %s, got %s", testRoundID, bulk.RoundID)
	}
	if bulk.ChannelID != "channel-1" {
		t.Errorf("expected channel_id channel-1, got %s", bulk.ChannelID)
	}
	if bulk.MessageID != "message-1" {
		t.Errorf("expected message_id message-1, got %s", bulk.MessageID)
	}
	if len(bulk.Updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(bulk.Updates))
	}
	if bulk.Updates[0].UserID != userID1 {
		t.Errorf("expected first update user_id %s, got %s", userID1, bulk.Updates[0].UserID)
	}
	if bulk.Updates[0].Score == nil || *bulk.Updates[0].Score != score1 {
		t.Errorf("expected first update score %d", score1)
	}
	if bulk.Updates[1].UserID != userID2 {
		t.Errorf("expected second update user_id %s, got %s", userID2, bulk.Updates[1].UserID)
	}
	if bulk.Updates[1].Score == nil || *bulk.Updates[1].Score != score2 {
		t.Errorf("expected second update score %d", score2)
	}
}

func TestScoreHandlers_HandleBulkCorrectScoreRequest_UsesDiscordMessageID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// no-op observability in handler tests

	mockSvc := scoremocks.NewMockService(ctrl)
	mockSvc.EXPECT().CorrectScore(gomock.Any(), sharedtypes.GuildID("guild-1"), gomock.Any(), sharedtypes.DiscordID("user-1"), sharedtypes.Score(5), nil).
		Return(scoreservice.ScoreOperationResult{Success: &sharedevents.ScoreUpdatedPayloadV1{GuildID: sharedtypes.GuildID("guild-1"), UserID: sharedtypes.DiscordID("user-1"), Score: sharedtypes.Score(5)}}, nil)

	h := &ScoreHandlers{
		service: mockSvc,
		helpers: nil,
	}

	ctx := context.WithValue(context.Background(), "discord_message_id", "discord-msg-1")

	payload := &sharedevents.ScoreBulkUpdateRequestedPayloadV1{
		GuildID: sharedtypes.GuildID("guild-1"),
		RoundID: sharedtypes.RoundID(uuid.New()),
		Updates: []sharedevents.ScoreUpdateRequestedPayloadV1{
			{
				GuildID: sharedtypes.GuildID("guild-1"),
				RoundID: sharedtypes.RoundID(uuid.New()),
				UserID:  sharedtypes.DiscordID("user-1"),
				Score:   sharedtypes.Score(5),
			},
		},
	}

	results, err := h.HandleBulkCorrectScoreRequest(ctx, payload)
	if err != nil {
		t.Fatalf("HandleBulkCorrectScoreRequest() unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	resPayload, ok := results[0].Payload.(*roundevents.ScoreBulkUpdateRequestPayloadV1)
	if !ok {
		t.Fatalf("expected payload *ScoreBulkUpdateRequestPayloadV1, got %T", results[0].Payload)
	}
	if resPayload.MessageID != "discord-msg-1" {
		t.Errorf("expected message_id discord-msg-1, got %s", resPayload.MessageID)
	}
}

func TestScoreHandlers_HandleBulkCorrectScoreRequest_NilPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// no-op observability in handler tests
	h := &ScoreHandlers{
		service: scoremocks.NewMockService(ctrl),
		helpers: nil,
	}

	_, err := h.HandleBulkCorrectScoreRequest(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil payload")
	}
}
