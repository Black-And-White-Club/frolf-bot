package scoreintegrationtests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// TestProcessRoundScores verifies that the ProcessRoundScores method correctly
// processes, stores, and returns tag mappings from round scores
func TestProcessRoundScores(t *testing.T) {
	// Set up the test dependencies
	deps := SetupTestScoreService(t)
	defer deps.Cleanup()

	// Create a test data generator with a fixed seed for reproducibility
	generator := testutils.NewTestDataGenerator(42)

	// Define test cases using table-driven approach
	testCases := []struct {
		name                 string
		setupFunc            func() (sharedtypes.RoundID, []sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]sharedtypes.TagNumber)
		expectError          bool
		expectFailurePayload bool
		expectedFailureError string
		validateFunc         func(t *testing.T, roundID sharedtypes.RoundID, result scoreservice.ScoreOperationResult, err error)
		cleanupBefore        bool
		concurrent           bool
		concurrentSize       int
	}{
		{
			name: "Successfully processes round scores with tags",
			setupFunc: func() (sharedtypes.RoundID, []sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				// Generate users for the test
				users := generator.GenerateUsers(5)

				// Generate a round with participants
				round := generator.GenerateRound(users[0].UserID, len(users), users)

				// Create ScoreInfo slice from round participants
				var scores []sharedtypes.ScoreInfo
				expectedTagMappings := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)

				// Realistic disc golf scores relative to par
				discGolfScores := []float64{
					0,  // Even/par
					-2, // 2 under par
					+3, // 3 over par
					-4, // 4 under par
					+1, // 1 over par
				}

				for i, participant := range round.Participants {
					if i < len(discGolfScores) && participant.Score != nil {
						// Convert testutils types to sharedtypes
						userID := sharedtypes.DiscordID(participant.UserID)
						score := sharedtypes.Score(discGolfScores[i])

						scoreInfo := sharedtypes.ScoreInfo{
							UserID: userID,
							Score:  score,
						}

						// Set tag number if available
						if participant.TagNumber != nil {
							tagNum := sharedtypes.TagNumber(*participant.TagNumber)
							scoreInfo.TagNumber = &tagNum
							expectedTagMappings[userID] = tagNum
						}

						scores = append(scores, scoreInfo)
					}
				}

				// Correctly convert round.ID to sharedtypes.RoundID (which is uuid.UUID)
				parsedUUID, err := uuid.Parse(round.ID.String())
				if err != nil {
					t.Fatalf("Failed to parse UUID: %v", err)
				}

				return sharedtypes.RoundID(parsedUUID), scores, expectedTagMappings
			},
			expectError:          false,
			expectFailurePayload: false,
			validateFunc: func(t *testing.T, roundID sharedtypes.RoundID, result scoreservice.ScoreOperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("Expected success payload, got nil")
				}

				// Declare and assign successPayload here, where it's used
				successPayload, ok := result.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
				if !ok {
					t.Fatalf("Invalid success payload type, expected *ProcessRoundScoresSuccessPayload, got %T", result.Success)
				}

				// Verify scores were stored in DB
				storedScores, err := deps.DB.GetScoresForRound(deps.Ctx, sharedtypes.GuildID("test_guild"), roundID)
				if err != nil {
					t.Fatalf("Failed to get scores from DB: %v", err)
				}

				if len(storedScores) == 0 {
					t.Error("Expected scores in DB, got none")
				}

				// Optional: Check tag mappings from the success payload
				if len(successPayload.TagMappings) == 0 {
					t.Error("Expected tag mappings in result, got none")
				}
				// Add more specific checks for tag mappings if needed
			},
			cleanupBefore: true,
		},
		{
			name: "Successfully processes round scores without tags",
			setupFunc: func() (sharedtypes.RoundID, []sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				// Generate users for the test
				users := generator.GenerateUsers(3)

				// Create round options to ensure no tag numbers
				finalized := true
				options := testutils.RoundOptions{
					CreatedBy:        users[0].UserID,
					ParticipantCount: len(users),
					Users:            users,
					State:            "COMPLETED",
					Finalized:        (*roundtypes.Finalized)(&finalized),
				}

				// Generate a round with participants but no tags
				round := generator.GenerateRoundWithConstraints(options)

				// Create ScoreInfo slice from round participants with no tag numbers
				var scores []sharedtypes.ScoreInfo

				// Disc golf appropriate scores
				discGolfScores := []float64{-1, 0, +2} // 1 under, even, 2 over

				for i, participant := range round.Participants {
					if i < len(discGolfScores) {
						scoreInfo := sharedtypes.ScoreInfo{
							UserID: sharedtypes.DiscordID(participant.UserID),
							Score:  sharedtypes.Score(discGolfScores[i]),
							// No tag numbers
						}
						scores = append(scores, scoreInfo)
					}
				}

				// Correctly convert round.ID to sharedtypes.RoundID
				parsedUUID, err := uuid.Parse(round.ID.String())
				if err != nil {
					t.Fatalf("Failed to parse UUID: %v", err)
				}
				return sharedtypes.RoundID(parsedUUID), scores, nil
			},
			expectError:          false,
			expectFailurePayload: false,
			validateFunc: func(t *testing.T, roundID sharedtypes.RoundID, result scoreservice.ScoreOperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("Expected success payload, got nil")
				}

				successPayload, ok := result.Success.(*scoreevents.ProcessRoundScoresSuccessPayload)
				if !ok {
					t.Fatalf("Invalid success payload type, expected *ProcessRoundScoresSuccessPayload, got %T", result.Success)
				}
				// Use successPayload to avoid "declared and not used" warning
				if successPayload.RoundID != roundID {
					t.Errorf("Mismatched RoundID in success payload, got: %v, expected: %v", successPayload.RoundID, roundID)
				}

				// Verify scores were stored in DB
				storedScores, err := deps.DB.GetScoresForRound(deps.Ctx, sharedtypes.GuildID("test_guild"), roundID)
				if err != nil {
					t.Fatalf("Failed to get scores from DB: %v", err)
				}

				if len(storedScores) == 0 {
					t.Errorf("Expected scores in DB, got none")
				}
				// Add checks for the content of storedScores if needed
			},
			cleanupBefore: true,
		},
		{
			name: "Handles empty score list",
			setupFunc: func() (sharedtypes.RoundID, []sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				// Generate a valid round ID
				round := generator.GenerateRound("admin123", 0, nil)

				// Correctly convert round.ID to sharedtypes.RoundID (which is uuid.UUID)
				parsedUUID, err := uuid.Parse(round.ID.String())
				if err != nil {
					t.Fatalf("Failed to parse UUID: %v", err)
				}
				return sharedtypes.RoundID(parsedUUID), []sharedtypes.ScoreInfo{}, nil
			},
			expectError:          false, // Expect nil Go error
			expectFailurePayload: true,  // Expect a failure payload
			expectedFailureError: "cannot process empty score list",
			validateFunc: func(t *testing.T, roundID sharedtypes.RoundID, result scoreservice.ScoreOperationResult, err error) {
				if err != nil {
					t.Errorf("Expected nil error for business failure, got: %v", err)
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, got %+v", result.Success)
				}
				if result.Failure == nil {
					t.Errorf("Expected non-nil failure payload, got nil")
				} else {
					failurePayload, ok := result.Failure.(*scoreevents.ProcessRoundScoresFailurePayload)
					if !ok {
						t.Errorf("Expected *ProcessRoundScoresFailurePayload, got %T", result.Failure)
					} else if failurePayload.Error != "cannot process empty score list" {
						t.Errorf("Mismatched failure error message, got: %q, expected: %q", failurePayload.Error, "cannot process empty score list")
					}
				}
			},
			cleanupBefore: true,
		},
		{
			name: "Correctly sorts scores",
			setupFunc: func() (sharedtypes.RoundID, []sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				// Generate users for the test
				users := generator.GenerateUsers(5)

				// Generate a round with participants
				round := generator.GenerateRound(users[0].UserID, len(users), users)

				// Create unsorted scores with converted types - using proper disc golf scores
				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: +4}, // 4 over par
					{UserID: sharedtypes.DiscordID(users[1].UserID), Score: -2}, // 2 under par
					{UserID: sharedtypes.DiscordID(users[2].UserID), Score: +7}, // 7 over par
					{UserID: sharedtypes.DiscordID(users[3].UserID), Score: -5}, // 5 under par
					{UserID: sharedtypes.DiscordID(users[4].UserID), Score: 0},  // even par
				}

				// Correctly convert round.ID to sharedtypes.RoundID
				parsedUUID, err := uuid.Parse(round.ID.String())
				if err != nil {
					t.Fatalf("Failed to parse UUID: %v", err)
				}
				return sharedtypes.RoundID(parsedUUID), scores, nil
			},
			expectError:          false,
			expectFailurePayload: false,
			validateFunc: func(t *testing.T, roundID sharedtypes.RoundID, result scoreservice.ScoreOperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got: %v", err)
				}
				if result.Success == nil {
					t.Fatalf("Expected success payload, got nil")
				}

				// Verify scores were stored in DB in sorted order
				storedScores, err := deps.DB.GetScoresForRound(deps.Ctx, sharedtypes.GuildID("test_guild"), roundID)
				if err != nil {
					t.Fatalf("Failed to get scores from DB: %v", err)
				}

				// Check if the scores are sorted (lowest score first)
				for i := 1; i < len(storedScores); i++ {
					if storedScores[i-1].Score > storedScores[i].Score {
						t.Errorf("Scores not sorted correctly: %v > %v at positions %d and %d",
							storedScores[i-1].Score, storedScores[i].Score, i-1, i)
					}
				}
			},
			cleanupBefore: true,
		},
		{
			name:           "Handles concurrent score processing",
			concurrent:     true,
			concurrentSize: 3,
			setupFunc: func() (sharedtypes.RoundID, []sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				// This function won't be used directly since we're handling concurrency separately

				// Create a placeholder RoundID (uuid.UUID)
				placeholderID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
				return sharedtypes.RoundID(placeholderID), nil, nil
			},
			expectError:          false,
			expectFailurePayload: false,
			cleanupBefore:        true,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.concurrent {
				// Special handling for concurrent test case
				runConcurrentScoreTest(t, deps, generator, tc.concurrentSize)
			} else {
				// Standard test case execution
				roundID, scores, _ := tc.setupFunc()

				// Process round scores
				guildID := sharedtypes.GuildID("test_guild")
				result, err := deps.Service.ProcessRoundScores(deps.Ctx, guildID, roundID, scores, false)

				// Validate the result
				tc.validateFunc(t, roundID, result, err) // Pass the entire result struct
			}
		})
	}
}

// runConcurrentScoreTest handles the concurrent score processing test
func runConcurrentScoreTest(t *testing.T, deps TestDeps, generator *testutils.TestDataGenerator, numRounds int) {
	users := generator.GenerateUsers(5)
	rounds := make([]roundtypes.Round, numRounds)
	scoresList := make([][]sharedtypes.ScoreInfo, numRounds)

	// Generate test data for each round
	for i := 0; i < numRounds; i++ {
		// Generate different rounds with the same users
		rounds[i] = generator.GenerateRound(users[0].UserID, len(users), users)

		// Create scores for each round - using realistic disc golf scores
		var roundScores []sharedtypes.ScoreInfo
		for j, user := range users {
			// Create different scores for each round
			// Base scores on player skill (j) and round difficulty (i)
			// Players with lower j indices are better players
			baseScore := float64(j - 2)       // Player skill: -2, -1, 0, 1, 2
			roundAdjustment := float64(i - 1) // Round difficulty: -1, 0, 1
			finalScore := baseScore + roundAdjustment

			tagNum := sharedtypes.TagNumber(j + 1)

			roundScores = append(roundScores, sharedtypes.ScoreInfo{
				UserID:    sharedtypes.DiscordID(user.UserID),
				Score:     sharedtypes.Score(finalScore),
				TagNumber: &tagNum,
			})
		}
		scoresList[i] = roundScores
	}

	// Process scores concurrently
	errChan := make(chan error, numRounds)
	doneChan := make(chan bool, numRounds)

	for i := 0; i < numRounds; i++ {
		go func(idx int) {
			// Correctly convert round.ID to sharedtypes.RoundID (which is uuid.UUID)
			parsedUUID, err := uuid.Parse(rounds[idx].ID.String())
			if err != nil {
				errChan <- err
				return
			}
			roundID := sharedtypes.RoundID(parsedUUID)

			// Call ProcessRoundScores, which returns ScoreOperationResult
			guildID := sharedtypes.GuildID("test_guild")
			result, err := deps.Service.ProcessRoundScores(
				deps.Ctx,
				guildID,
				roundID,
				scoresList[idx],
				false,
			)
			// Check for both technical errors and business logic errors
			if err != nil {
				errChan <- err
				return
			}
			if result.Failure != nil {
				// Convert business logic failure to an error for the channel
				if failureErr, ok := result.Failure.(*scoreevents.ProcessRoundScoresFailurePayload); ok {
					errChan <- fmt.Errorf("business failure: %s", failureErr.Error)
				} else {
					errChan <- fmt.Errorf("concurrent test: unexpected failure result type: %T", result.Failure)
				}
				return
			}

			doneChan <- true
		}(i)
	}

	// Collect results
	for i := 0; i < numRounds; i++ {
		select {
		case err := <-errChan:
			t.Errorf("Error processing round scores concurrently: %v", err)
		case <-doneChan:
			// Success
		case <-time.After(5 * time.Second):
			t.Errorf("Timeout waiting for concurrent score processing")
		}
	}

	// Verify all rounds have scores in DB
	for i := 0; i < numRounds; i++ {
		// Correctly convert round.ID to sharedtypes.RoundID (which is uuid.UUID)
		parsedUUID, err := uuid.Parse(rounds[i].ID.String())
		if err != nil {
			t.Fatalf("Failed to parse UUID: %v", err)
		}
		roundID := sharedtypes.RoundID(parsedUUID)

		// Use the same guildID as above
		storedScores, err := deps.DB.GetScoresForRound(
			deps.Ctx,
			sharedtypes.GuildID("test_guild"),
			roundID,
		)
		if err != nil {
			t.Fatalf("Failed to get scores for round %s: %v", rounds[i].ID, err)
		}

		if len(storedScores) != len(scoresList[i]) {
			t.Errorf("Expected %d scores for round %s, got %d",
				len(scoresList[i]), rounds[i].ID, len(storedScores))
		}
	}
}

// TestProcessScoresForStorage tests the score processing and sorting functionality
func TestProcessScoresForStorage(t *testing.T) {
	// Set up the test dependencies
	deps := SetupTestScoreService(t)
	defer deps.Cleanup()

	// Create a test data generator
	generator := testutils.NewTestDataGenerator(42)

	// Define test cases
	testCases := []struct {
		name        string
		scores      []sharedtypes.ScoreInfo
		expectError bool
		validate    func(t *testing.T, scores []sharedtypes.ScoreInfo, err error)
	}{
		{
			name: "Correctly processes and sorts scores",
			scores: func() []sharedtypes.ScoreInfo {
				tag1 := sharedtypes.TagNumber(1)
				tag2 := sharedtypes.TagNumber(2)
				return []sharedtypes.ScoreInfo{
					{UserID: "user1", Score: +3, TagNumber: &tag1}, // 3 over par
					{UserID: "user2", Score: -2},                   // 2 under par
					{UserID: "user3", Score: +5, TagNumber: &tag2}, // 5 over par
					{UserID: "user4", Score: -4},                   // 4 under par
					{UserID: "user5", Score: 0},                    // even par
				}
			}(),
			expectError: false,
			validate: func(t *testing.T, scores []sharedtypes.ScoreInfo, err error) {
				if err != nil {
					t.Fatalf("Failed to process scores: %v", err)
				}

				// Verify scores are sorted (lowest score first)
				for i := 1; i < len(scores); i++ {
					if scores[i-1].Score > scores[i].Score {
						t.Errorf("Scores not sorted correctly: %v > %v",
							scores[i-1].Score, scores[i].Score)
					}
				}

				// Count tagged scores
				taggedCount := 0
				for _, score := range scores {
					if score.TagNumber != nil {
						taggedCount++
					}
				}

				if taggedCount != 2 {
					t.Errorf("Expected 2 tagged scores, got %d", taggedCount)
				}
			},
		},
		{
			name:        "Handles empty score list correctly",
			scores:      []sharedtypes.ScoreInfo{},
			expectError: true,
			validate: func(t *testing.T, scores []sharedtypes.ScoreInfo, err error) {
				if err == nil {
					t.Error("Expected error when processing empty score list, got nil")
				}
				// Check the error message content
				if err != nil && !strings.Contains(err.Error(), "cannot process empty score list") {
					t.Errorf("Expected error message to contain 'cannot process empty score list', got: %v", err)
				}
			},
		},
		{
			name: "Handles extreme scores",
			scores: []sharedtypes.ScoreInfo{
				{UserID: "user1", Score: -12}, // Excellent round (12 under par)
				{UserID: "user2", Score: +15}, // Very poor round (15 over par)
				{UserID: "user3", Score: 0},   // Par
			},
			expectError: false,
			validate: func(t *testing.T, scores []sharedtypes.ScoreInfo, err error) {
				if err != nil {
					t.Fatalf("Failed to process scores with extreme values: %v", err)
				}

				// Verify sorting
				if scores[0].Score != -12 || scores[2].Score != +15 {
					t.Errorf("Extreme scores not handled correctly: %v", scores)
				}
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate a valid round ID for each test
			round := generator.GenerateRound("admin123", 0, nil)

			// Correctly convert round.ID to sharedtypes.RoundID (which is uuid.UUID)
			parsedUUID, err := uuid.Parse(round.ID.String())
			if err != nil {
				t.Fatalf("Failed to parse UUID: %v", err)
			}
			roundID := sharedtypes.RoundID(parsedUUID)

			// Call the Service directly (not ScoreService)
			guildID := sharedtypes.GuildID("test_guild")
			processedScores, err := deps.Service.ProcessScoresForStorage(deps.Ctx, guildID, roundID, tc.scores)

			// Validate results based on test case expectations
			tc.validate(t, processedScores, err)
		})
	}
}
