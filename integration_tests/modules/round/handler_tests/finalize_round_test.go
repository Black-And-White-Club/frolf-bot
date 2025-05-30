package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidAllScoresSubmittedPayload creates a valid AllScoresSubmittedPayload for testing
func createValidAllScoresSubmittedPayload(roundID sharedtypes.RoundID, participants []roundtypes.Participant, round roundtypes.Round) roundevents.AllScoresSubmittedPayload {
	return roundevents.AllScoresSubmittedPayload{
		RoundID:        roundID,
		EventMessageID: round.EventMessageID,
		RoundData:      round,
		Participants:   participants,
	}
}

// createValidRoundFinalizedPayload creates a valid RoundFinalizedPayload for testing
func createValidRoundFinalizedPayload(roundID sharedtypes.RoundID, roundData roundtypes.Round) roundevents.RoundFinalizedPayload {
	return roundevents.RoundFinalizedPayload{
		RoundID:   roundID,
		RoundData: roundData,
	}
}

// createExistingRoundForFinalization creates and stores a round that can be finalized
func createExistingRoundForFinalization(t *testing.T, userID sharedtypes.DiscordID, db bun.IDB) (sharedtypes.RoundID, []roundtypes.Participant, roundtypes.Round) {
	t.Helper()

	// Use the passed DB instance instead of creating new deps
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

	// Generate a complete round with participants and scores
	roundData := generator.GenerateRound(testutils.DiscordID(userID), 0, []testutils.User{}) // Start with 0 participants

	// Create sample participants with realistic disc golf scores
	tagNumber1 := sharedtypes.TagNumber(1)
	tagNumber2 := sharedtypes.TagNumber(2)
	score1 := sharedtypes.Score(-3) // 3 under par (excellent score)
	score2 := sharedtypes.Score(2)  // 2 over par (decent score)

	participants := []roundtypes.Participant{
		{
			UserID:    userID,
			TagNumber: &tagNumber1,
			Response:  roundtypes.ResponseAccept,
			Score:     &score1,
		},
		{
			UserID:    sharedtypes.DiscordID("123456789012345678"),
			TagNumber: &tagNumber2,
			Response:  roundtypes.ResponseAccept,
			Score:     &score2,
		},
	}

	// Add participants to round data
	roundData.Participants = participants

	// Convert to DB model and insert using the passed DB instance
	roundDB := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  *roundData.Description,
		Location:     *roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
	}

	_, err := db.NewInsert().Model(roundDB).Exec(context.Background()) // ✅ Use passed DB
	if err != nil {
		t.Fatalf("Failed to insert test round for finalization: %v", err)
	}

	return roundData.ID, participants, roundData
}

// TestHandleAllScoresSubmitted tests the all scores submitted handler
func TestHandleAllScoresSubmitted(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)
	const testTimeout = 5 * time.Second // Define a common timeout for waiting

	testCases := []struct {
		name                    string
		setupAndRun             func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) // ✅ Pass deps
		expectDiscordFinalized  bool
		expectFinalizationError bool
		expectNoMessages        bool
	}{
		{
			name:                   "Success - Valid All Scores Submitted",
			expectDiscordFinalized: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) { // ✅ Accept deps
				roundID, participants, round := createExistingRoundForFinalization(t, userID, deps.DB) // ✅ Pass DB
				payload := createValidAllScoresSubmittedPayload(roundID, participants, round)
				helper.PublishAllScoresSubmitted(t, context.Background(), payload)
			},
		},
		{
			name:                    "Failure - Non-Existent Round ID",
			expectFinalizationError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())

				// Create a minimal round for the payload
				dummyRound := roundtypes.Round{
					ID:             nonExistentRoundID,
					Title:          "Test Round",
					EventMessageID: "test_message_id",
				}

				tagNumber := sharedtypes.TagNumber(1)
				score := sharedtypes.Score(1)
				participants := []roundtypes.Participant{
					{
						UserID:    userID,
						TagNumber: &tagNumber,
						Response:  roundtypes.ResponseAccept,
						Score:     &score,
					},
				}

				payload := createValidAllScoresSubmittedPayload(nonExistentRoundID, participants, dummyRound)
				helper.PublishAllScoresSubmitted(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Invalid JSON Message",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundAllScoresSubmitted)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t) // ✅ Create deps once per test case
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			helper.ClearMessages()

			// Run the test, passing deps to avoid creating new instances
			tc.setupAndRun(t, helper, &deps) // ✅ Pass deps to the test function

			if tc.expectDiscordFinalized {
				if !helper.WaitForDiscordRoundFinalized(1, testTimeout) {
					t.Error("Timed out waiting for discord finalized message")
				}
			} else if tc.expectFinalizationError {
				if !helper.WaitForRoundFinalizationError(1, testTimeout) {
					t.Error("Timed out waiting for finalization error message")
				}
			} else if tc.expectNoMessages {
				time.Sleep(500 * time.Millisecond)
			}

			// Check results
			discordFinalizedMsgs := helper.GetDiscordRoundFinalizedMessages()
			finalizationErrorMsgs := helper.GetRoundFinalizationErrorMessages()

			if tc.expectDiscordFinalized {
				if len(discordFinalizedMsgs) == 0 {
					t.Error("Expected discord finalized message, got none")
				}
				if len(finalizationErrorMsgs) > 0 {
					t.Errorf("Expected no finalization error messages, got %d", len(finalizationErrorMsgs))
				}
			} else if tc.expectFinalizationError {
				if len(finalizationErrorMsgs) == 0 {
					t.Error("Expected finalization error message, got none")
				}
				if len(discordFinalizedMsgs) > 0 {
					t.Errorf("Expected no discord finalized messages, got %d", len(discordFinalizedMsgs))
				}
			} else if tc.expectNoMessages {
				if len(discordFinalizedMsgs) > 0 {
					t.Errorf("Expected no discord finalized messages, got %d", len(discordFinalizedMsgs))
				}
				if len(finalizationErrorMsgs) > 0 {
					t.Errorf("Expected no finalization error messages, got %d", len(finalizationErrorMsgs))
				}
			}
		})
	}
}

// Also fix TestHandleRoundFinalized with the same pattern
func TestHandleRoundFinalized(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)
	const testTimeout = 5 * time.Second

	testCases := []struct {
		name                    string
		setupAndRun             func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) // ✅ Pass deps
		expectScoreProcessing   bool
		expectFinalizationError bool
		expectNoMessages        bool
	}{
		{
			name:                  "Success - Valid Round Finalized",
			expectScoreProcessing: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) { // ✅ Accept deps
				roundID, _, existingRound := createExistingRoundForFinalization(t, userID, deps.DB) // ✅ Pass DB
				payload := createValidRoundFinalizedPayload(roundID, existingRound)
				helper.PublishRoundFinalized(t, context.Background(), payload)
			},
		},
		{
			name:                    "Failure - Non-Existent Round ID",
			expectFinalizationError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				dummyRoundData := roundtypes.Round{ID: nonExistentRoundID, Participants: []roundtypes.Participant{}}
				payload := createValidRoundFinalizedPayload(nonExistentRoundID, dummyRoundData)
				helper.PublishRoundFinalized(t, context.Background(), payload)
			},
		},
		{
			name:             "Failure - Invalid JSON Message",
			expectNoMessages: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundFinalized)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t) // ✅ Create deps once per test case
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			helper.ClearMessages()

			// Run the test, passing deps to avoid creating new instances
			tc.setupAndRun(t, helper, &deps) // ✅ Pass deps to the test function

			if tc.expectScoreProcessing {
				if !helper.WaitForProcessRoundScoresRequest(1, testTimeout) {
					t.Error("Timed out waiting for score processing message")
				}
			} else if tc.expectFinalizationError {
				if !helper.WaitForRoundFinalizationError(1, testTimeout) {
					t.Error("Timed out waiting for finalization error message")
				}
			} else if tc.expectNoMessages {
				time.Sleep(500 * time.Millisecond)
			}

			// Check results
			scoreProcessingMsgs := helper.GetProcessRoundScoresRequestMessages()
			finalizationErrorMsgs := helper.GetRoundFinalizationErrorMessages()

			if tc.expectScoreProcessing {
				if len(scoreProcessingMsgs) == 0 {
					t.Error("Expected score processing message, got none")
				}
				if len(finalizationErrorMsgs) > 0 {
					t.Errorf("Expected no finalization error messages, got %d", len(finalizationErrorMsgs))
				}
			} else if tc.expectFinalizationError {
				if len(finalizationErrorMsgs) == 0 {
					t.Error("Expected finalization error message, got none")
				}
				if len(scoreProcessingMsgs) > 0 {
					t.Errorf("Expected no score processing messages, got %d", len(scoreProcessingMsgs))
				}
			} else if tc.expectNoMessages {
				if len(scoreProcessingMsgs) > 0 {
					t.Errorf("Expected no score processing messages, got %d", len(scoreProcessingMsgs))
				}
				if len(finalizationErrorMsgs) > 0 {
					t.Errorf("Expected no finalization error messages, got %d", len(finalizationErrorMsgs))
				}
			}
		})
	}
}
