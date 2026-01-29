package scoreintegrationtests

import (
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// TestCorrectScore verifies that the CorrectScore method updates or inserts scores properly.
func TestCorrectScore(t *testing.T) {
	// Set up service dependencies
	deps := SetupTestScoreService(t)

	// Create test data generator
	generator := testutils.NewTestDataGenerator(42)
	users := generator.GenerateUsers(3)

	testCases := []struct {
		name                 string
		userIndex            int
		score                sharedtypes.Score
		tag                  *sharedtypes.TagNumber
		seedRound            bool
		expectFailurePayload bool
		expectedFailureError func(roundID sharedtypes.RoundID) string
	}{
		{
			name:                 "Add initial score with tag number",
			userIndex:            0,
			score:                3,
			tag:                  ptrTag(7),
			seedRound:            true,
			expectFailurePayload: false,
		},
		{
			name:                 "Add initial score without tag number",
			userIndex:            1,
			score:                -1,
			tag:                  nil,
			seedRound:            true,
			expectFailurePayload: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var roundID sharedtypes.RoundID
			var user sharedtypes.DiscordID

			user = sharedtypes.DiscordID(users[tc.userIndex].UserID)

			if tc.seedRound {
				round := generator.GenerateRound(users[tc.userIndex].UserID, 1, []testutils.User{users[tc.userIndex]})
				roundID = sharedtypes.RoundID(round.ID)

				initial := []sharedtypes.ScoreInfo{{
					UserID:    user,
					Score:     0,
					TagNumber: nil,
				}}
				_, err := deps.Service.ProcessRoundScores(deps.Ctx, sharedtypes.GuildID("test_guild"), roundID, initial, false)
				if err != nil {
					t.Fatalf("Failed to seed round with initial score: %v", err)
				}
			} else {
				roundID = sharedtypes.RoundID(uuid.New())
			}

			// Call CorrectScore
			guildID := sharedtypes.GuildID("test_guild")
			result, err := deps.Service.CorrectScore(deps.Ctx, guildID, roundID, user, tc.score, tc.tag)

			if err != nil {
				t.Fatalf("CorrectScore returned unexpected error: %v", err)
			}

			if tc.expectFailurePayload {
				if !result.IsFailure() {
					t.Fatal("Expected failure result, got success")
				}
				// Handle failure validation if needed
				return
			}

			if !result.IsSuccess() {
				t.Fatal("Expected success result, got failure")
			}

			successPayload := result.Success
			if successPayload.UserID != user || successPayload.Score != tc.score {
				t.Errorf("Unexpected success result: %+v", successPayload)
			}

			// Verify DB result
			storedScores, err := deps.DB.GetScoresForRound(deps.Ctx, nil, guildID, roundID)
			if err != nil {
				t.Fatalf("Failed to retrieve scores: %v", err)
			}

			var found bool
			for _, s := range storedScores {
				if s.UserID == user {
					found = true
					if s.Score != tc.score {
						t.Errorf("Expected score %.2f, got %.2f", float64(tc.score), float64(s.Score))
					}
					if (tc.tag == nil && s.TagNumber != nil) ||
						(tc.tag != nil && (s.TagNumber == nil || *s.TagNumber != *tc.tag)) {
						t.Errorf("Expected tag %v, got %v", tc.tag, s.TagNumber)
					}
				}
			}
			if !found {
				t.Errorf("Score for user %s not found in DB", user)
			}
		})
	}
}

func ptrTag(tn sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &tn
}
