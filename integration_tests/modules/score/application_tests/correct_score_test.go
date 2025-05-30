package scoreintegrationtests

import (
	"fmt" // Added fmt import for Sprintf
	"testing"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// TestCorrectScore verifies that the CorrectScore method updates or inserts scores properly.
func TestCorrectScore(t *testing.T) {
	// Set up service dependencies
	deps := SetupTestScoreService(t)
	defer deps.Cleanup()

	// Create test data generator
	generator := testutils.NewTestDataGenerator(42)
	users := generator.GenerateUsers(3)

	testCases := []struct {
		name                 string
		userIndex            int
		score                sharedtypes.Score
		tag                  *sharedtypes.TagNumber
		expectFailurePayload bool
		// Changed expectedFailureError to a function to handle dynamic round IDs
		expectedFailureError func(roundID sharedtypes.RoundID) string
	}{
		{
			name:                 "Add initial score with tag number",
			userIndex:            0,
			score:                3,
			tag:                  ptrTag(7),
			expectFailurePayload: false,
			expectedFailureError: nil, // No failure expected
		},
		{
			name:                 "Add initial score without tag number",
			userIndex:            1,
			score:                -1,
			tag:                  nil,
			expectFailurePayload: false,
			expectedFailureError: nil, // No failure expected
		},
		{
			name:                 "Update existing score with new tag",
			userIndex:            0,
			score:                5,
			tag:                  ptrTag(1),
			expectFailurePayload: false,
			expectedFailureError: nil, // No failure expected
		},
		{
			name:                 "Update existing score without tag",
			userIndex:            1,
			score:                2,
			tag:                  nil,
			expectFailurePayload: false,
			expectedFailureError: nil, // No failure expected
		},
		{
			name:                 "Fails with invalid round ID",
			userIndex:            2,
			score:                0,
			tag:                  ptrTag(5),
			expectFailurePayload: true,
			// Dynamically generate the expected error message
			expectedFailureError: func(roundID sharedtypes.RoundID) string {
				return fmt.Sprintf("score record not found for round %s", roundID)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var roundID sharedtypes.RoundID
			var user sharedtypes.DiscordID

			user = sharedtypes.DiscordID(users[tc.userIndex].UserID)

			if tc.expectFailurePayload { // Check for expected failure payload
				// Create a completely new, unseeded round ID
				roundID = sharedtypes.RoundID(uuid.New())
			} else {
				// Create a fresh round with one user
				round := generator.GenerateRound(users[tc.userIndex].UserID, 1, []testutils.User{users[tc.userIndex]})
				parsedUUID, err := uuid.Parse(round.ID.String())
				if err != nil {
					t.Fatalf("Failed to parse round UUID: %v", err)
				}
				roundID = sharedtypes.RoundID(parsedUUID)

				// Seed the round with an initial score
				initial := []sharedtypes.ScoreInfo{{
					UserID:    user,
					Score:     0,
					TagNumber: nil,
				}}
				_, err = deps.Service.ProcessRoundScores(deps.Ctx, roundID, initial)
				if err != nil {
					t.Fatalf("Failed to seed round with initial score: %v", err)
				}
			}

			// Call CorrectScore
			result, err := deps.Service.CorrectScore(deps.Ctx, roundID, user, tc.score, tc.tag)

			// Validate based on whether a failure payload is expected
			if tc.expectFailurePayload {
				if err != nil {
					t.Errorf("Expected nil error for business failure, got: %v", err)
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, got %+v", result.Success)
				}
				if result.Failure == nil {
					t.Errorf("Expected non-nil failure payload, got nil")
				} else {
					failurePayload, ok := result.Failure.(*scoreevents.ScoreUpdateFailurePayload)
					if !ok {
						t.Errorf("Expected *ScoreUpdateFailurePayload, got %T", result.Failure)
					} else {
						// Get the expected error string by calling the function
						expectedErrStr := tc.expectedFailureError(roundID)
						if failurePayload.Error != expectedErrStr {
							t.Errorf("Mismatched failure error message, got: %q, expected: %q", failurePayload.Error, expectedErrStr)
						}
					}
				}
				return // End test case here as failure path is handled
			}

			// Original success path validation (only runs if tc.expectFailurePayload is false)
			if err != nil {
				t.Fatalf("CorrectScore returned unexpected error: %v", err)
			}

			successPayload, ok := result.Success.(*scoreevents.ScoreUpdateSuccessPayload)
			if !ok {
				t.Fatalf("Expected *ScoreUpdateSuccessPayload, got %T", result.Success)
			}
			if successPayload.UserID != user || successPayload.Score != tc.score {
				t.Errorf("Unexpected success result: %+v", successPayload)
			}

			// Verify DB result
			storedScores, err := deps.DB.GetScoresForRound(deps.Ctx, roundID)
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
