package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidRoundMessageIDUpdatePayload creates a valid RoundMessageIDUpdatePayload for testing
func createValidRoundMessageIDUpdatePayload(roundID sharedtypes.RoundID) roundevents.RoundMessageIDUpdatePayload {
	return roundevents.RoundMessageIDUpdatePayload{
		RoundID: roundID,
	}
}

// expectMessageIDUpdateSuccess validates successful message ID update
func expectMessageIDUpdateSuccess(t *testing.T, helper *testutils.RoundTestHelper, originalRoundID sharedtypes.RoundID, expectedDiscordMessageID string, timeout time.Duration) {
	t.Helper()

	if !helper.WaitForRoundEventMessageIDUpdated(1, timeout) {
		t.Fatalf("Expected round event message ID updated message within %v", timeout)
	}

	msgs := helper.GetRoundEventMessageIDUpdatedMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 success message, got %d", len(msgs))
	}

	result := helper.ValidateRoundEventMessageIDUpdated(t, msgs[0], originalRoundID)

	// Validate that the message ID matches
	if result.EventMessageID != expectedDiscordMessageID {
		t.Errorf("Discord message ID mismatch: expected %s, got %s", expectedDiscordMessageID, result.EventMessageID)
	}

	// Validate other fields
	if result.BaseRoundPayload.RoundID != originalRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", originalRoundID, result.BaseRoundPayload.RoundID)
	}
}

// expectMessageIDUpdateFailure validates message ID update failure scenarios
func expectMessageIDUpdateFailure(t *testing.T, helper *testutils.RoundTestHelper, timeout time.Duration) {
	t.Helper()

	// Wait to ensure no success messages are published
	time.Sleep(timeout)

	successMsgs := helper.GetRoundEventMessageIDUpdatedMessages()
	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for failed update, got %d", len(successMsgs))
	}
}

// createExistingRoundForUpdate creates and stores a round that can be updated
func createExistingRoundForUpdate(t *testing.T, helper *testutils.RoundTestHelper, userID sharedtypes.DiscordID, db bun.IDB) sharedtypes.RoundID {
	t.Helper()

	// Use the passed DB instance instead of creating new deps
	return helper.CreateRoundInDB(t, db, userID)
}

// TestHandleRoundEventMessageIDUpdate runs integration tests for the round event message ID update handler
func TestHandleRoundEventMessageIDUpdate(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) // ✅ Pass deps
		expectError bool
	}{
		{
			name: "Success - Update Valid Round Message ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) { // ✅ Accept deps
				// Create a round first
				roundID := createExistingRoundForUpdate(t, helper, userID, deps.DB) // ✅ Pass DB

				// Create update payload
				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_123456"

				// Publish with Discord message ID in metadata
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)
				expectMessageIDUpdateSuccess(t, helper, roundID, discordMessageID, 3*time.Second)
			},
		},
		{
			name: "Success - Update Round Message ID with Long Discord ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForUpdate(t, helper, userID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_987654321098765432" // Long ID

				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)
				expectMessageIDUpdateSuccess(t, helper, roundID, discordMessageID, 3*time.Second)
			},
		},
		{
			name:        "Failure - Missing Discord Message ID in Metadata",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForUpdate(t, helper, userID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)

				// Publish without Discord message ID in metadata
				helper.PublishRoundMessageIDUpdateWithoutDiscordID(t, context.Background(), payload)
				expectMessageIDUpdateFailure(t, helper, 2*time.Second)
			},
		},
		{
			name:        "Failure - Empty Discord Message ID in Metadata",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForUpdate(t, helper, userID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)

				// Publish with empty Discord message ID
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, "")
				expectMessageIDUpdateFailure(t, helper, 2*time.Second)
			},
		},
		{
			name:        "Failure - Non-Existent Round ID",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Use a random round ID that doesn't exist
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())

				payload := createValidRoundMessageIDUpdatePayload(nonExistentRoundID)
				discordMessageID := "discord_msg_123456"

				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)
				expectMessageIDUpdateFailure(t, helper, 2*time.Second)
			},
		},
		{
			name:        "Failure - Invalid JSON Message",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundEventMessageIDUpdate)
				expectMessageIDUpdateFailure(t, helper, 2*time.Second)
			},
		},
		{
			name: "Success - Handler Preserves Message Correlation ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForUpdate(t, helper, userID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_correlation_test"

				originalMsg := helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)

				if !helper.WaitForRoundEventMessageIDUpdated(1, 3*time.Second) {
					t.Fatalf("Expected success message")
				}

				msgs := helper.GetRoundEventMessageIDUpdatedMessages()
				resultMsg := msgs[0]

				// Verify correlation ID is preserved
				originalCorrelationID := originalMsg.Metadata.Get("correlation_id")
				resultCorrelationID := resultMsg.Metadata.Get("correlation_id")

				if originalCorrelationID != resultCorrelationID {
					t.Errorf("Correlation ID not preserved: original=%s, result=%s",
						originalCorrelationID, resultCorrelationID)
				}
			},
		},
		{
			name: "Success - Multiple Message ID Updates",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create two rounds using the same DB instance
				roundID1 := helper.CreateRoundInDB(t, deps.DB, userID)
				roundID2 := helper.CreateRoundInDB(t, deps.DB, userID)

				payload1 := createValidRoundMessageIDUpdatePayload(roundID1)
				payload2 := createValidRoundMessageIDUpdatePayload(roundID2)

				discordMessageID1 := "discord_msg_first"
				discordMessageID2 := "discord_msg_second"

				// Publish both updates
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload1, discordMessageID1)
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload2, discordMessageID2)

				// Both should succeed
				if !helper.WaitForRoundEventMessageIDUpdated(2, 5*time.Second) {
					t.Fatalf("Expected 2 success messages within 5s")
				}

				msgs := helper.GetRoundEventMessageIDUpdatedMessages()
				if len(msgs) != 2 {
					t.Fatalf("Expected 2 success messages, got %d", len(msgs))
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t) // ✅ Create deps once per test case
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Clear any existing captured messages
			helper.ClearMessages()

			// Run the test, passing deps to avoid creating new instances
			tc.setupAndRun(t, helper, &deps) // ✅ Pass deps to the test function
		})
	}
}
