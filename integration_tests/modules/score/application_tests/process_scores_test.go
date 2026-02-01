package scoreintegrationtests

import (
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestProcessRoundScores(t *testing.T) {
	deps := SetupTestScoreService(t)

	generator := testutils.NewTestDataGenerator(42)
	users := generator.GenerateUsers(5)

	testCases := []struct {
		name      string
		numScores int
		overwrite bool
		expectErr bool
	}{
		{
			name:      "Process initial scores",
			numScores: 3,
			overwrite: false,
			expectErr: false,
		},
		{
			name:      "Overwrite existing scores",
			numScores: 5,
			overwrite: true,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			roundID := sharedtypes.RoundID(uuid.New())
			guildID := sharedtypes.GuildID("test_guild")

			scores := make([]sharedtypes.ScoreInfo, tc.numScores)
			for i := 0; i < tc.numScores; i++ {
				tag := sharedtypes.TagNumber(i + 1)
				scores[i] = sharedtypes.ScoreInfo{
					UserID:    sharedtypes.DiscordID(users[i].UserID),
					Score:     sharedtypes.Score(i - 2),
					TagNumber: &tag,
				}
			}

			result, err := deps.Service.ProcessRoundScores(deps.Ctx, guildID, roundID, scores, tc.overwrite)

			if tc.expectErr {
				if err == nil && !result.IsFailure() {
					t.Fatal("Expected error or failure, got success")
				}
				return
			}

			if err != nil {
				t.Fatalf("ProcessRoundScores returned unexpected error: %v", err)
			}

			if !result.IsSuccess() {
				t.Fatal("Expected success result, got failure")
			}

			successPayload := result.Success
			if len(successPayload.TagMappings) != tc.numScores {
				t.Errorf("Expected %d tag mappings, got %d", tc.numScores, len(successPayload.TagMappings))
			}

			// Verify DB result
			storedScores, err := deps.DB.GetScoresForRound(deps.Ctx, nil, guildID, roundID)
			if err != nil {
				t.Fatalf("Failed to retrieve scores: %v", err)
			}

			if len(storedScores) != tc.numScores {
				t.Errorf("Expected %d scores in DB, got %d", tc.numScores, len(storedScores))
			}
		})
	}
}
