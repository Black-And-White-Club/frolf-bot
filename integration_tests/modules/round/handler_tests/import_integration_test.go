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

	// 1. Insert an empty round to import into
	gen := testutils.NewTestDataGenerator()
	roundData := gen.GenerateRound(testutils.DiscordID("uploader-1"), 0, []testutils.User{})

	roundDB := &rounddb.Round{
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

	if _, err := deps.DB.NewInsert().Model(roundDB).Exec(context.Background()); err != nil {
		t.Fatalf("failed to insert round: %v", err)
	}

	// Insert users matching fixture names so resolver can find them for doubles import
	// The fixture has "Alec + Jace" and "Eff + Thao" - we create users for these names
	urepo := userdb.NewRepository(deps.DB)
	ctx := context.Background()

	// Create users for the doubles fixture (Alec, Jace, Eff, Thao)
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
		if err := urepo.CreateGlobalUser(ctx, &userdb.User{UserID: uid}); err != nil {
			t.Fatalf("failed to create global user %s: %v", u.username, err)
		}
		if err := urepo.UpdateUDiscIdentityGlobal(ctx, uid, &u.username, &u.name); err != nil {
			t.Fatalf("failed to update udisc identity for %s: %v", u.username, err)
		}
		if err := urepo.CreateGuildMembership(ctx, &userdb.GuildMembership{UserID: uid, GuildID: "test-guild", Role: "User"}); err != nil {
			t.Fatalf("failed to create guild membership for %s: %v", u.username, err)
		}
	}

	genericCase := testutils.TestCase{
		Name: "Import doubles produces teams",
		SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
			return nil
		},
		PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
			// Read fixture file
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
			var payload roundevents.AllScoresSubmittedPayloadV1
			if err := json.Unmarshal(msgs[0].Payload, &payload); err != nil {
				t.Fatalf("failed to unmarshal AllScoresSubmitted payload: %v", err)
			}
			if len(payload.Teams) == 0 {
				t.Fatalf("expected teams to be present, got none")
			}
			// Ensure members have RawName set
			for _, tm := range payload.Teams {
				if len(tm.Members) == 0 {
					t.Fatalf("team has no members")
				}
				for _, m := range tm.Members {
					if m.RawName == "" {
						t.Fatalf("expected member RawName to be set")
					}
				}
			}
		},
		ExpectError:    false,
		MessageTimeout: 10 * time.Second,
	}

	testutils.RunTest(t, genericCase, deps.TestEnvironment)
}

// ptrString is a small helper for test construction
func ptrString(s string) *string { return &s }

// TestImportSingles ensures that importing a singles CSV results in no Teams and participants present
func TestImportSinglesCreatesPlayers(t *testing.T) {
	deps := SetupTestRoundHandler(t)

	// Insert an empty round to import into
	gen := testutils.NewTestDataGenerator()
	roundData := gen.GenerateRound(testutils.DiscordID("uploader-2"), 0, []testutils.User{})

	roundDB := &rounddb.Round{
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

	if _, err := deps.DB.NewInsert().Model(roundDB).Exec(context.Background()); err != nil {
		t.Fatalf("failed to insert round: %v", err)
	}

	// Seed users so resolver can find them for singles import
	urepo := userdb.NewRepository(deps.DB)
	ctx := context.Background()
	user1 := &userdb.User{
		UserID:        sharedtypes.DiscordID("111111111111111111"),
		UDiscUsername: ptrString("jace"),
		UDiscName:     ptrString("Jace"),
	}
	user2 := &userdb.User{
		UserID:        sharedtypes.DiscordID("222222222222222222"),
		UDiscUsername: ptrString("sam"),
		UDiscName:     ptrString("Sam"),
	}
	if err := urepo.CreateGlobalUser(ctx, user1); err != nil {
		t.Fatalf("failed to create global user1: %v", err)
	}
	if err := urepo.CreateGlobalUser(ctx, user2); err != nil {
		t.Fatalf("failed to create global user2: %v", err)
	}
	if err := urepo.UpdateUDiscIdentityGlobal(ctx, user1.UserID, user1.UDiscUsername, user1.UDiscName); err != nil {
		t.Fatalf("failed to update udisc identity for user1: %v", err)
	}
	if err := urepo.UpdateUDiscIdentityGlobal(ctx, user2.UserID, user2.UDiscUsername, user2.UDiscName); err != nil {
		t.Fatalf("failed to update udisc identity for user2: %v", err)
	}
	if err := urepo.CreateGuildMembership(ctx, &userdb.GuildMembership{UserID: user1.UserID, GuildID: "test-guild", Role: "User"}); err != nil {
		t.Fatalf("failed to create guild membership for user1: %v", err)
	}
	if err := urepo.CreateGuildMembership(ctx, &userdb.GuildMembership{UserID: user2.UserID, GuildID: "test-guild", Role: "User"}); err != nil {
		t.Fatalf("failed to create guild membership for user2: %v", err)
	}

	// Update the round to include participants so UpdateParticipantScore can find them
	participants := []roundtypes.Participant{
		{UserID: user1.UserID, Response: roundtypes.ResponseAccept},
		{UserID: user2.UserID, Response: roundtypes.ResponseAccept},
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
			var payload roundevents.AllScoresSubmittedPayloadV1
			if err := json.Unmarshal(msgs[0].Payload, &payload); err != nil {
				t.Fatalf("failed to unmarshal AllScoresSubmitted payload: %v", err)
			}
			if len(payload.Teams) != 0 {
				t.Fatalf("expected no teams for singles import, got %d", len(payload.Teams))
			}
			if len(payload.Participants) == 0 {
				t.Fatalf("expected participants to be present, got none")
			}
			// Ensure at least one participant has a score
			hasScore := false
			for _, p := range payload.Participants {
				if p.Score != nil {
					hasScore = true
					break
				}
			}
			if !hasScore {
				t.Fatalf("expected at least one participant with a score")
			}
		},
		ExpectError:    false,
		MessageTimeout: 10 * time.Second,
	}

	testutils.RunTest(t, genericCase, deps.TestEnvironment)
}

// TestImportSinglesAddsOnlyGuildMembersNotGuests ensures that singles imports:
// 1. Add non-RSVP participants when the matched user has a guild_membership for the round's guild
// 2. Do NOT create guest participants for singles (unmatched names are skipped)
func TestImportSinglesAddsOnlyGuildMembersNotGuests(t *testing.T) {
	t.Skip("Integration test - run locally with test infrastructure")

	deps := SetupTestRoundHandler(t)
	ctx := context.Background()

	// Create a round with NO participants (empty RSVP list)
	gen := testutils.NewTestDataGenerator()
	roundData := gen.GenerateRound(testutils.DiscordID("uploader-3"), 0, []testutils.User{})

	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  roundData.Description,
		Location:     roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: []roundtypes.Participant{}, // Empty - no RSVP participants
		GuildID:      "test-guild",
	}

	if _, err := deps.DB.NewInsert().Model(roundDB).Exec(ctx); err != nil {
		t.Fatalf("failed to insert round: %v", err)
	}

	// Seed ONE user with guild_membership (Jace) - this user should be added as participant
	urepo := userdb.NewRepository(deps.DB)
	userWithMembership := &userdb.User{
		UserID:        sharedtypes.DiscordID("333333333333333333"),
		UDiscUsername: ptrString("jace"),
		UDiscName:     ptrString("Jace"),
	}
	if err := urepo.CreateGlobalUser(ctx, userWithMembership); err != nil {
		t.Fatalf("failed to create global user: %v", err)
	}
	if err := urepo.UpdateUDiscIdentityGlobal(ctx, userWithMembership.UserID, userWithMembership.UDiscUsername, userWithMembership.UDiscName); err != nil {
		t.Fatalf("failed to update udisc identity: %v", err)
	}
	if err := urepo.CreateGuildMembership(ctx, &userdb.GuildMembership{UserID: userWithMembership.UserID, GuildID: "test-guild", Role: "User"}); err != nil {
		t.Fatalf("failed to create guild membership: %v", err)
	}

	// Note: "Sam" from the CSV will NOT have a guild_membership, so should NOT be added as participant

	genericCase := testutils.TestCase{
		Name:    "Singles import adds only guild members, no guests",
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
				UserID:    sharedtypes.DiscordID("uploader-3"),
				ChannelID: "chan-3",
				FileData:  data,
				FileName:  "test_singles.csv",
				MessageID: "evt-3",
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
			var payload roundevents.AllScoresSubmittedPayloadV1
			if err := json.Unmarshal(msgs[0].Payload, &payload); err != nil {
				t.Fatalf("failed to unmarshal AllScoresSubmitted payload: %v", err)
			}

			// Verify no teams (singles mode)
			if len(payload.Teams) != 0 {
				t.Fatalf("expected no teams for singles import, got %d", len(payload.Teams))
			}

			// Verify only the matched guild member was added as participant
			if len(payload.Participants) != 1 {
				t.Fatalf("expected exactly 1 participant (matched guild member), got %d", len(payload.Participants))
			}

			// Verify the participant is the guild member (Jace)
			p := payload.Participants[0]
			if p.UserID != userWithMembership.UserID {
				t.Fatalf("expected participant UserID %s, got %s", userWithMembership.UserID, p.UserID)
			}

			// Verify no guest participants (UserID should not be empty)
			for _, participant := range payload.Participants {
				if participant.UserID == "" {
					t.Fatalf("found guest participant (empty UserID) - singles should not create guests")
				}
			}

			// Verify the participant has a score
			if p.Score == nil {
				t.Fatalf("expected participant to have a score")
			}
		},
		ExpectError:    false,
		MessageTimeout: 10 * time.Second,
	}

	testutils.RunTest(t, genericCase, deps.TestEnvironment)
}
