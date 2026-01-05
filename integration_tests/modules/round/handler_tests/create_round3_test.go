package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidRoundMessageIDUpdatePayload creates a valid RoundMessageIDUpdatePayload for testing
func createValidRoundMessageIDUpdatePayload(roundID sharedtypes.RoundID) roundevents.RoundMessageIDUpdatePayloadV1 {
	return roundevents.RoundMessageIDUpdatePayloadV1{
		RoundID: roundID,
		GuildID: "test-guild",
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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
		expectError bool
	}{
		{
			name: "Success - Update Valid Round Message ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				// Create a round first
				roundID := createExistingRoundForUpdate(t, helper, data.UserID, deps.DB)

				// Create update payload
				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_123456"

				// Publish with Discord message ID in metadata
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)
				expectMessageIDUpdateSuccess(t, helper, roundID, discordMessageID, 500*time.Millisecond)
			},
		},
		{
			name: "Success - Update Round Message ID with Long Discord ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForUpdate(t, helper, data.UserID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_987654321098765432" // Long ID

				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)
				expectMessageIDUpdateSuccess(t, helper, roundID, discordMessageID, 500*time.Millisecond)
			},
		},
		{
			name:        "Failure - Missing Discord Message ID in Metadata",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForUpdate(t, helper, data.UserID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)

				// Publish without Discord message ID in metadata
				helper.PublishRoundMessageIDUpdateWithoutDiscordID(t, context.Background(), payload)
				expectMessageIDUpdateFailure(t, helper, 500*time.Millisecond)
			},
		},
		{
			name:        "Failure - Empty Discord Message ID in Metadata",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForUpdate(t, helper, data.UserID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)

				// Publish with empty Discord message ID
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, "")
				expectMessageIDUpdateFailure(t, helper, 500*time.Millisecond)
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
				expectMessageIDUpdateFailure(t, helper, 500*time.Millisecond)
			},
		},
		{
			name:        "Failure - Invalid JSON Message",
			expectError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundEventMessageIDUpdateV1)
				expectMessageIDUpdateFailure(t, helper, 500*time.Millisecond)
			},
		},
		{
			name: "Success - Handler Preserves Message Correlation ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForUpdate(t, helper, data.UserID, deps.DB)

				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_correlation_test"

				originalMsg := helper.PublishRoundMessageIDUpdate(t, context.Background(), payload, discordMessageID)

				if !helper.WaitForRoundEventMessageIDUpdated(1, 500*time.Millisecond) {
					t.Fatalf("Expected success message")
				}

				msgs := helper.GetRoundEventMessageIDUpdatedMessages()

				// Find the message for this specific round
				var resultMsg *message.Message
				for _, msg := range msgs {
					parsed, err := testutils.ParsePayload[roundevents.RoundScheduledPayloadV1](msg)
					if err == nil && parsed.BaseRoundPayload.RoundID == roundID {
						resultMsg = msg
						break
					}
				}

				if resultMsg == nil {
					t.Fatalf("No message found for round %s", roundID)
				}

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
				data1 := NewTestData()
				data2 := NewTestData()

				// Create two rounds using the same DB instance
				roundID1 := helper.CreateRoundInDB(t, deps.DB, data1.UserID)
				roundID2 := helper.CreateRoundInDB(t, deps.DB, data2.UserID)

				payload1 := createValidRoundMessageIDUpdatePayload(roundID1)
				payload2 := createValidRoundMessageIDUpdatePayload(roundID2)

				discordMessageID1 := "discord_msg_first"
				discordMessageID2 := "discord_msg_second"

				// Publish both updates
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload1, discordMessageID1)
				helper.PublishRoundMessageIDUpdate(t, context.Background(), payload2, discordMessageID2)

				// Both should succeed
				if !helper.WaitForRoundEventMessageIDUpdated(2, 500*time.Millisecond) {
					t.Fatalf("Expected 2 success messages within 500ms")
				}

				msgs := helper.GetRoundEventMessageIDUpdatedMessages()
				if len(msgs) < 2 {
					t.Fatalf("Expected at least 2 success messages, got %d", len(msgs))
				}
			},
		},
	}

	// Run all subtests with SHARED setup - no need to clear messages between tests!
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {			// Clear message capture before each subtest
			deps.MessageCapture.Clear()			// Create helper for each subtest
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test - no cleanup needed!
			tc.setupAndRun(t, helper, &deps)
		})
	}
}
