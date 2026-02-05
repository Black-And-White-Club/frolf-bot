package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// TestImportDoublesRound ensures that importing a doubles CSV results in Teams being present
func TestImportDoublesCreatesTeams(t *testing.T) {
	deps := SetupTestRoundHandler(t)
	// Ensure streams are created (import flow involves multiple streams)
	ensureStreams(t, deps.TestEnvironment)

	// 1. Insert an empty round to import into
	gen := testutils.NewTestDataGenerator()
	roundData := gen.GenerateRound(testutils.DiscordID("uploader-1"), 0, []testutils.User{})

	roundDBRec := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  roundData.Description,
		Location:     roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild",
	}

	if _, err := deps.DB.NewInsert().Model(roundDBRec).Exec(context.Background()); err != nil {
		t.Fatalf("failed to insert round: %v", err)
	}

	// Insert users matching fixture names so resolver can find them for doubles import
	urepo := userdb.NewRepository(deps.DB)
	ctx := context.Background()

	users := []struct {
		id       string
		username string
		name     string
	}{
		{"111111111111111111", "alec", "Alec"},
		{"222222222222222222", "jace", "Jace"},
		{"333333333333333333", "eff", "Eff"},
		{"444444444444444444", "thao", "Thao"},
	}

	for _, u := range users {
		uid := sharedtypes.DiscordID(u.id)
		user := &userdb.User{
			UserID:        &uid,
			UDiscUsername: ptrString(u.username),
			UDiscName:     ptrString(u.name),
		}
		if err := urepo.SaveGlobalUser(ctx, deps.DB, user); err != nil {
			t.Fatalf("failed to save global user %s: %v", u.username, err)
		}
		if err := urepo.CreateGuildMembership(ctx, deps.DB, &userdb.GuildMembership{UserID: uid, GuildID: "test-guild", Role: "User"}); err != nil {
			t.Fatalf("failed to create guild membership for %s: %v", u.username, err)
		}
	}

	genericCase := testutils.TestCase{
		Name: "Import doubles produces teams",
		SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
			return nil
		},
		PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
			data, err := os.ReadFile("fixtures/test_doubles.csv")
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}

			importID := uuid.New().String()
			payload := roundevents.ScorecardUploadedPayloadV1{
				GuildID:   "test-guild",
				RoundID:   roundData.ID,
				ImportID:  importID,
				UserID:    sharedtypes.DiscordID("uploader-1"),
				ChannelID: "chan-1",
				FileData:  data,
				FileName:  "test_doubles.csv",
				MessageID: "evt-1",
				Timestamp: time.Now().UTC(),
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}
			msg := message.NewMessage(uuid.New().String(), payloadBytes)
			msg.Metadata.Set("event_name", roundevents.ScorecardUploadedV1)
			if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.ScorecardUploadedV1, msg); err != nil {
				t.Fatalf("Publish failed: %v", err)
			}
			return msg
		},
		ExpectedTopics: []string{roundevents.RoundAllScoresSubmittedV1},
		ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
			msgs := receivedMsgs[roundevents.RoundAllScoresSubmittedV1]
			if len(msgs) == 0 {
				t.Fatalf("expected RoundAllScoresSubmitted message, got none")
			}
		},
		ExpectError:    false,
		MessageTimeout: 10 * time.Second,
	}

	testutils.RunTest(t, genericCase, deps.TestEnvironment)
}

func ptrString(s string) *string { return &s }

func TestImportSinglesCreatesPlayers(t *testing.T) {
	deps := SetupTestRoundHandler(t)
	// Ensure streams are created
	ensureStreams(t, deps.TestEnvironment)

	gen := testutils.NewTestDataGenerator()
	roundData := gen.GenerateRound(testutils.DiscordID("uploader-2"), 0, []testutils.User{})

	roundDBRec := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  roundData.Description,
		Location:     roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild",
	}

	if _, err := deps.DB.NewInsert().Model(roundDBRec).Exec(context.Background()); err != nil {
		t.Fatalf("failed to insert round: %v", err)
	}

	urepo := userdb.NewRepository(deps.DB)
	ctx := context.Background()
	uid1 := sharedtypes.DiscordID("111111111111111111")
	user1 := &userdb.User{
		UserID:        &uid1,
		UDiscUsername: ptrString("jace"),
		UDiscName:     ptrString("Jace"),
	}
	uid2 := sharedtypes.DiscordID("222222222222222222")
	user2 := &userdb.User{
		UserID:        &uid2,
		UDiscUsername: ptrString("sam"),
		UDiscName:     ptrString("Sam"),
	}
	if err := urepo.SaveGlobalUser(ctx, deps.DB, user1); err != nil {
		t.Fatalf("failed to save global user1: %v", err)
	}
	if err := urepo.SaveGlobalUser(ctx, deps.DB, user2); err != nil {
		t.Fatalf("failed to save global user2: %v", err)
	}
	if err := urepo.CreateGuildMembership(ctx, deps.DB, &userdb.GuildMembership{UserID: user1.GetUserID(), GuildID: "test-guild", Role: "User"}); err != nil {
		t.Fatalf("failed to create guild membership for user1: %v", err)
	}
	if err := urepo.CreateGuildMembership(ctx, deps.DB, &userdb.GuildMembership{UserID: user2.GetUserID(), GuildID: "test-guild", Role: "User"}); err != nil {
		t.Fatalf("failed to create guild membership for user2: %v", err)
	}

	participants := []roundtypes.Participant{
		{UserID: user1.GetUserID(), Response: roundtypes.ResponseAccept},
		{UserID: user2.GetUserID(), Response: roundtypes.ResponseAccept},
	}
	if _, err := deps.DB.NewUpdate().Model(&rounddb.Round{}).Set("participants = ?", participants).Where("id = ?", roundData.ID).Exec(context.Background()); err != nil {
		t.Fatalf("failed to update round participants: %v", err)
	}

	genericCase := testutils.TestCase{
		Name:    "Import singles produces participants",
		SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} { return nil },
		PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
			data, err := os.ReadFile("fixtures/test_singles.csv")
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}
			importID := uuid.New().String()
			payload := roundevents.ScorecardUploadedPayloadV1{
				GuildID:   "test-guild",
				RoundID:   roundData.ID,
				ImportID:  importID,
				UserID:    sharedtypes.DiscordID("uploader-2"),
				ChannelID: "chan-1",
				FileData:  data,
				FileName:  "test_singles.csv",
				MessageID: "evt-2",
				Timestamp: time.Now().UTC(),
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}
			msg := message.NewMessage(uuid.New().String(), payloadBytes)
			msg.Metadata.Set("event_name", roundevents.ScorecardUploadedV1)
			if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.ScorecardUploadedV1, msg); err != nil {
				t.Fatalf("Publish failed: %v", err)
			}
			return msg
		},
		ExpectedTopics: []string{roundevents.RoundAllScoresSubmittedV1},
		ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
			msgs := receivedMsgs[roundevents.RoundAllScoresSubmittedV1]
			if len(msgs) == 0 {
				t.Fatalf("expected RoundAllScoresSubmitted message, got none")
			}
		},
		ExpectError:    false,
		MessageTimeout: 10 * time.Second,
	}

	testutils.RunTest(t, genericCase, deps.TestEnvironment)
}
