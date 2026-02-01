package roundintegrationtests

import (
	"context"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestIngestNormalizedScorecard(t *testing.T) {
	tests := []struct {
		name            string
		setupTestEnv    func(ctx context.Context, deps RoundTestDeps) (roundtypes.ImportIngestScorecardInput, sharedtypes.DiscordID)
		expectedFailure bool
		validateResult  func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.IngestScorecardResult, userID sharedtypes.DiscordID)
	}{
		{
			name: "Successfully ingest normalized scorecard - Singles",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (roundtypes.ImportIngestScorecardInput, sharedtypes.DiscordID) {
				guildID := sharedtypes.GuildID("test-guild")
				roundID := sharedtypes.RoundID(uuid.New())

				// Create a user that can be resolved by name
				userID := sharedtypes.DiscordID("user-to-match")
				// We need to use testutils to insert a user into the global DB with a UDisc name
				err := testutils.InsertUser(t, deps.BunDB, userID, guildID, sharedtypes.UserRoleUser)
				if err != nil {
					t.Fatalf("Failed to insert user: %v", err)
				}

				// Also need to set the UDisc name for this user so resolveUserID finds it
				_, err = deps.BunDB.NewUpdate().
					Table("users").
					Set("udisc_name = ?", "matched player").
					Where("user_id = ?", userID).
					Exec(ctx)
				if err != nil {
					t.Fatalf("Failed to update udisc_name: %v", err)
				}

				return roundtypes.ImportIngestScorecardInput{
					GuildID:  guildID,
					RoundID:  roundID,
					ImportID: "import-123",
					UserID:   "uploader-id",
					NormalizedData: roundtypes.NormalizedScorecard{
						Mode: sharedtypes.RoundModeSingles,
						Players: []roundtypes.NormalizedPlayer{
							{DisplayName: "Matched Player", Total: 54, HoleScores: []int{3, 3, 3}},
						},
					},
				}, userID
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res *roundtypes.IngestScorecardResult, userID sharedtypes.DiscordID) {
				if res == nil {
					t.Fatalf("Expected success result, got nil")
				}
				if res.MatchedPlayers != 1 {
					t.Errorf("Expected 1 matched player, got %d", res.MatchedPlayers)
				}
				if len(res.Scores) != 1 {
					t.Fatalf("Expected 1 score, got %d", len(res.Scores))
				}
				if res.Scores[0].UserID != userID {
					t.Errorf("Expected userID %s, got %s", userID, res.Scores[0].UserID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req, userID := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.IngestNormalizedScorecard(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, deps.Ctx, deps, *result.Success, userID)
				}
			}
		})
	}
}

func TestNormalizeParsedScorecard(t *testing.T) {
	tests := []struct {
		name           string
		data           *roundtypes.ParsedScorecard
		meta           roundtypes.Metadata
		validateResult func(t *testing.T, res *roundtypes.NormalizedScorecard)
	}{
		{
			name: "Successfully normalize parsed scorecard - Singles",
			data: &roundtypes.ParsedScorecard{
				Mode: sharedtypes.RoundModeSingles,
				PlayerScores: []roundtypes.PlayerScoreRow{
					{PlayerName: "Player 1", Total: 54, HoleScores: []int{3, 3}},
				},
				ParScores: []int{3, 3},
			},
			meta: roundtypes.Metadata{
				RoundID:  sharedtypes.RoundID(uuid.New()),
				GuildID:  "test-guild",
				ImportID: "import-123",
			},
			validateResult: func(t *testing.T, res *roundtypes.NormalizedScorecard) {
				if res == nil {
					t.Fatalf("Expected result, got nil")
				}
				if res.Mode != sharedtypes.RoundModeSingles {
					t.Errorf("Expected mode %s, got %s", sharedtypes.RoundModeSingles, res.Mode)
				}
				if len(res.Players) != 1 {
					t.Errorf("Expected 1 player, got %d", len(res.Players))
				}
				if res.Players[0].DisplayName != "Player 1" {
					t.Errorf("Expected name Player 1, got %s", res.Players[0].DisplayName)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			result, err := deps.Service.NormalizeParsedScorecard(deps.Ctx, tt.data, tt.meta)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Success == nil {
				t.Fatalf("Expected success result, but got failure: %+v", result.Failure)
			}
			if tt.validateResult != nil {
				tt.validateResult(t, *result.Success)
			}
		})
	}
}
