package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// createValidRoundEntityCreatedPayload creates a valid RoundEntityCreatedPayload for testing
func createValidRoundEntityCreatedPayload(userID sharedtypes.DiscordID) roundevents.RoundEntityCreatedPayloadV1 {
	now := time.Now()
	startTime := sharedtypes.StartTime(now.Add(24 * time.Hour))
	description := roundtypes.Description("Test round for deletion")
	location := roundtypes.Location("Test Course")

	return roundevents.RoundEntityCreatedPayloadV1{
		GuildID: "test-guild",
		Round: roundtypes.Round{
			ID:          sharedtypes.RoundID(uuid.New()),
			Title:       "Test Round",
			Description: &description,
			Location:    &location,
			StartTime:   &startTime,
			CreatedBy:   userID,
			State:       roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{
				{UserID: userID, Response: roundtypes.ResponseAccept},
			},
			Finalized: roundtypes.Finalized(false),
		},
		DiscordChannelID: "test-channel-123",
		DiscordGuildID:   "test-guild",
	}
}

// createMinimalRoundEntityCreatedPayload creates a minimal but valid payload
func createMinimalRoundEntityCreatedPayload(userID sharedtypes.DiscordID) roundevents.RoundEntityCreatedPayloadV1 {
	roundID := sharedtypes.RoundID(uuid.New())
	now := time.Now()
	startTime := sharedtypes.StartTime(now.Add(24 * time.Hour))

	description := roundtypes.Description("Quick round")
	location := roundtypes.Location("Local Course")

	return roundevents.RoundEntityCreatedPayloadV1{
		GuildID: "test-guild",
		Round: roundtypes.Round{
			ID:          roundID,
			Title:       "Quick Round",
			Description: &description,
			Location:    &location,
			StartTime:   &startTime,
			CreatedBy:   userID,
			State:       roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{
				{UserID: userID, Response: roundtypes.ResponseAccept},
			},
			Finalized: roundtypes.Finalized(false),
		},
		DiscordChannelID: "test-channel-456",
		DiscordGuildID:   "test-guild",
	}
}

// expectStoreSuccess validates successful round storage
func expectStoreSuccess(t *testing.T, helper *testutils.RoundTestHelper, originalPayload roundevents.RoundEntityCreatedPayloadV1, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var matchingMsg *message.Message

	// Poll for the specific message until timeout
	for time.Now().Before(deadline) {
		msgs := helper.GetRoundCreatedMessages()
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.RoundCreatedPayloadV1](msg)
			if err == nil && parsed.RoundID == originalPayload.Round.ID {
				matchingMsg = msg
				break
			}
		}

		if matchingMsg != nil {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if matchingMsg == nil {
		msgs := helper.GetRoundCreatedMessages()
		t.Fatalf("Expected round.created message for round %s within %v, but none found among %d messages",
			originalPayload.Round.ID, timeout, len(msgs))
	}

	result := helper.ValidateRoundCreated(t, matchingMsg, originalPayload.Round.ID)

	// Validate that the stored round matches the original
	if result.RoundID != originalPayload.Round.ID {
		t.Errorf("RoundID mismatch: expected %s, got %s", originalPayload.Round.ID, result.RoundID)
	}
	// Add more detailed field comparisons if necessary
}

// expectNoMessages validates that no messages were published (for JSON errors)
func expectNoMessages(t *testing.T, helper *testutils.RoundTestHelper, timeout time.Duration) {
	t.Helper()
	time.Sleep(timeout)

	createdMsgs := helper.GetRoundCreatedMessages()
	if len(createdMsgs) > 0 {
		t.Errorf("Expected no '%s' messages, got %d", roundevents.RoundCreatedV1, len(createdMsgs))
	}

	creationFailedMsgs := helper.GetRoundCreationFailedMessages()
	if len(creationFailedMsgs) > 0 {
		t.Errorf("Expected no '%s' messages, got %d", roundevents.RoundCreationFailedV1, len(creationFailedMsgs))
	}
}

// TestHandleRoundEntityCreated runs integration tests for the round entity created handler
func TestHandleRoundEntityCreated(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper)
	}{
		{
			name: "Success - Handler Processes Valid Message and Publishes Success Event",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				// Use longer timeout for first test to allow router warm-up
				expectStoreSuccess(t, helper, payload, 2*time.Second)
			},
		},
		{
			name: "Success - Handler Processes Minimal Valid Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data := NewTestData()
				payload := createMinimalRoundEntityCreatedPayload(data.UserID)
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				expectStoreSuccess(t, helper, payload, 500*time.Millisecond)
			},
		},
		{
			name: "Success - Handler Processes Message with Complex Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				payload.Round.Participants = []roundtypes.Participant{
					{UserID: data.UserID, Response: roundtypes.ResponseAccept},
					{UserID: sharedtypes.DiscordID("user123"), Response: roundtypes.ResponseTentative},
					{UserID: sharedtypes.DiscordID("user456"), Response: roundtypes.ResponseDecline},
				}
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				expectStoreSuccess(t, helper, payload, 500*time.Millisecond)
			},
		},
		{
			name: "Success - Handler Processes Different Round States",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				payload.Round.State = roundtypes.RoundStateInProgress

				helper.PublishRoundEntityCreated(t, context.Background(), payload)

				// Wait for messages and capture what we get
				if !helper.WaitForRoundCreated(1, 200*time.Millisecond) {
					t.Fatalf("Expected round.created message within 200ms")
				}

				expectStoreSuccess(t, helper, payload, 200*time.Millisecond)
			},
		},
		{
			name: "Success - Handler Processes Multiple Messages Concurrently",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data1 := NewTestData()
				data2 := NewTestData()

				payload1 := createValidRoundEntityCreatedPayload(data1.UserID)
				payload2 := createValidRoundEntityCreatedPayload(data2.UserID)

				// Publish both quickly to test concurrent processing
				helper.PublishRoundEntityCreated(t, context.Background(), payload1)
				helper.PublishRoundEntityCreated(t, context.Background(), payload2)

				// Both should be processed successfully
				if !helper.WaitForRoundCreated(2, 500*time.Millisecond) {
					t.Fatalf("Expected 2 success messages (round.created) within 500ms, got %d", len(helper.GetRoundCreatedMessages()))
				}

				msgs := helper.GetRoundCreatedMessages()
				if len(msgs) < 2 {
					t.Fatalf("Expected at least 2 success messages (round.created), got %d", len(msgs))
				}
			},
		},
		{
			name: "Failure - Handler Rejects Invalid JSON and Doesn't Publish Events",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				// Count messages before
				createdMsgsBefore := len(helper.GetRoundCreatedMessages())
				creationFailedMsgsBefore := len(helper.GetRoundCreationFailedMessages())

				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundEntityCreatedV1)

				time.Sleep(400 * time.Millisecond)

				createdMsgsAfter := len(helper.GetRoundCreatedMessages())
				creationFailedMsgsAfter := len(helper.GetRoundCreationFailedMessages())

				newCreatedMsgs := createdMsgsAfter - createdMsgsBefore
				newFailedMsgs := creationFailedMsgsAfter - creationFailedMsgsBefore

				if newCreatedMsgs > 0 {
					t.Errorf("Expected no NEW '%s' messages, got %d new", roundevents.RoundCreatedV1, newCreatedMsgs)
				}

				if newFailedMsgs > 0 {
					t.Errorf("Expected no NEW '%s' messages, got %d new", roundevents.RoundCreationFailedV1, newFailedMsgs)
				}
			},
		},
		{
			name: "Success - Handler Preserves Message Correlation ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				originalMsg := helper.PublishRoundEntityCreated(t, context.Background(), payload)

				if !helper.WaitForRoundCreated(1, 500*time.Millisecond) {
					t.Fatalf("Expected success message (round.created)")
				}

				msgs := helper.GetRoundCreatedMessages()
				if len(msgs) == 0 {
					t.Fatalf("No round.created messages captured")
				}

				// Find the message for this specific round
				var resultMsg *message.Message
				for _, msg := range msgs {
					parsed, err := testutils.ParsePayload[roundevents.RoundCreatedPayloadV1](msg)
					if err == nil && parsed.RoundID == payload.Round.ID {
						resultMsg = msg
						break
					}
				}

				if resultMsg == nil {
					t.Fatalf("No round.created message found for round %s", payload.Round.ID)
				}

				// Verify correlation ID is preserved
				originalCorrelationID := originalMsg.Metadata.Get("correlation_id")
				resultCorrelationID := resultMsg.Metadata.Get("correlation_id")

				if originalCorrelationID == "" {
					t.Errorf("Original message correlation ID was empty")
				}
				if originalCorrelationID != resultCorrelationID {
					t.Errorf("Correlation ID not preserved: original=%s, result=%s",
						originalCorrelationID, resultCorrelationID)
				}
			},
		},
		{
			name: "Success - Handler Publishes Correct Event Topic",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)

				helper.PublishRoundEntityCreated(t, context.Background(), payload)

				// Poll for the specific message
				deadline := time.Now().Add(1 * time.Second)
				var foundMsg bool
				for time.Now().Before(deadline) {
					msgs := helper.GetRoundCreatedMessages()
					for _, msg := range msgs {
						parsed, err := testutils.ParsePayload[roundevents.RoundCreatedPayloadV1](msg)
						if err == nil && parsed.RoundID == payload.Round.ID {
							foundMsg = true
							break
						}
					}
					if foundMsg {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}

				if !foundMsg {
					t.Fatalf("Expected round.created message for round %s", payload.Round.ID)
				}
			},
		},
	}

	// Run all subtests with SHARED setup - no need to clear messages between tests!
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Clear message capture before each subtest
			deps.MessageCapture.Clear()
			// Create helper for each subtest
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test - no cleanup needed!
			tc.setupAndRun(t, helper)
		})
	}
}
