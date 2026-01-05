package userintegrationtests

import (
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
)

func TestUDiscIntegration(t *testing.T) {
	deps := SetupTestUserService(t)
	defer deps.Cleanup()

	// Setup test data
	guildID := sharedtypes.GuildID("guild-integration-test")
	userID1 := sharedtypes.DiscordID("user-1")
	userID2 := sharedtypes.DiscordID("user-2")

	// Create users first
	user1 := &userdb.User{
		UserID:  userID1,
		GuildID: guildID,
		Role:    sharedtypes.UserRoleUser,
	}
	user2 := &userdb.User{
		UserID:  userID2,
		GuildID: guildID,
		Role:    sharedtypes.UserRoleUser,
	}

	if err := deps.DB.CreateUser(deps.Ctx, user1); err != nil {
		t.Fatalf("Failed to create user 1: %v", err)
	}
	if err := deps.DB.CreateUser(deps.Ctx, user2); err != nil {
		t.Fatalf("Failed to create user 2: %v", err)
	}

	t.Run("UpdateUDiscIdentity and FindByUDiscUsername", func(t *testing.T) {
		username := "TestUser1"
		name := "Test Name 1"

		// Update identity
		result, err := deps.Service.UpdateUDiscIdentity(deps.Ctx, guildID, userID1, &username, &name)
		if err != nil {
			t.Fatalf("UpdateUDiscIdentity failed: %v", err)
		}
		if result.Success != true {
			t.Errorf("Expected success true, got %v", result.Success)
		}

		// Verify DB update via FindByUDiscUsername (case insensitive)
		foundUser, err := deps.Service.FindByUDiscUsername(deps.Ctx, guildID, "testuser1")
		if err != nil {
			t.Fatalf("FindByUDiscUsername failed: %v", err)
		}
		foundUserPayload := foundUser.Success.(*userdb.User)
		if foundUserPayload.UserID != userID1 {
			t.Errorf("Expected user ID %s, got %s", userID1, foundUserPayload.UserID)
		}
		if *foundUserPayload.UDiscUsername != "testuser1" {
			t.Errorf("Expected normalized username 'testuser1', got '%s'", *foundUserPayload.UDiscUsername)
		}
	})

	t.Run("UpdateUDiscIdentity and FindByUDiscName", func(t *testing.T) {
		username := "TestUser2"
		name := "Test Name 2"

		// Update identity
		_, err := deps.Service.UpdateUDiscIdentity(deps.Ctx, guildID, userID2, &username, &name)
		if err != nil {
			t.Fatalf("UpdateUDiscIdentity failed: %v", err)
		}

		// Verify DB update via FindByUDiscName (case insensitive)
		foundUser, err := deps.Service.FindByUDiscName(deps.Ctx, guildID, "test name 2")
		if err != nil {
			t.Fatalf("FindByUDiscName failed: %v", err)
		}
		foundUserPayload := foundUser.Success.(*userdb.User)
		if foundUserPayload.UserID != userID2 {
			t.Errorf("Expected user ID %s, got %s", userID2, foundUserPayload.UserID)
		}
		if *foundUserPayload.UDiscName != "test name 2" {
			t.Errorf("Expected normalized name 'test name 2', got '%s'", *foundUserPayload.UDiscName)
		}
	})

	t.Run("MatchParsedScorecard", func(t *testing.T) {
		// User 1 has username "testuser1"
		// User 2 has name "test name 2"

		payload := roundevents.ParsedScorecardPayload{
			ImportID: "import-1",
			GuildID:  guildID,
			RoundID:  sharedtypes.RoundID(uuid.New()),
			UserID:   userID1,
			ParsedData: &roundtypes.ParsedScorecard{
				PlayerScores: []roundtypes.PlayerScoreRow{
					{PlayerName: "TestUser1"},   // Should match user 1 by username
					{PlayerName: "Test Name 2"}, // Should match user 2 by name
					{PlayerName: "Unknown"},     // Should not match
				},
			},
		}

		result, err := deps.Service.MatchParsedScorecard(deps.Ctx, payload)
		if err != nil {
			t.Fatalf("MatchParsedScorecard failed: %v", err)
		}

		matchPayload, ok := result.Success.(*userevents.UDiscMatchConfirmedPayloadV1)
		if !ok {
			t.Fatalf("Expected UDiscMatchConfirmedPayload, got %T", result.Success)
		}

		if len(matchPayload.Mappings) != 2 {
			t.Errorf("Expected 2 mappings, got %d", len(matchPayload.Mappings))
		}

		// Verify mappings
		mappingMap := make(map[string]sharedtypes.DiscordID)
		for _, m := range matchPayload.Mappings {
			mappingMap[m.PlayerName] = m.DiscordUserID
		}

		if uid, ok := mappingMap["TestUser1"]; !ok || uid != userID1 {
			t.Errorf("Expected TestUser1 to map to %s, got %s", userID1, uid)
		}
		if uid, ok := mappingMap["Test Name 2"]; !ok || uid != userID2 {
			t.Errorf("Expected Test Name 2 to map to %s, got %s", userID2, uid)
		}
	})
}
