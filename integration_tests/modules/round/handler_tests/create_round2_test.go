package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// createValidRoundEntityCreatedPayload creates a valid RoundEntityCreatedPayload for testing
func createValidRoundEntityCreatedPayload(userID sharedtypes.DiscordID) roundevents.RoundEntityCreatedPayload {
	now := time.Now()
	startTime := sharedtypes.StartTime(now.Add(24 * time.Hour))
	description := roundtypes.Description("Test round for deletion")
	location := roundtypes.Location("Test Course")

	return roundevents.RoundEntityCreatedPayload{
		Round: roundtypes.Round{
			ID:           sharedtypes.RoundID(uuid.New()),
			Title:        roundtypes.Title("Test Round"),
			Description:  &description,
			Location:     &location,
			StartTime:    &startTime,
			CreatedBy:    userID,
			State:        roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{},
			Finalized:    roundtypes.Finalized(false),
		},
	}
}

// createMinimalRoundEntityCreatedPayload creates a minimal but valid payload
func createMinimalRoundEntityCreatedPayload(userID sharedtypes.DiscordID) roundevents.RoundEntityCreatedPayload {
	roundID := sharedtypes.RoundID(uuid.New())
	now := time.Now()
	startTime := sharedtypes.StartTime(now.Add(24 * time.Hour))

	description := roundtypes.Description("Quick round")
	location := roundtypes.Location("Local Course")

	return roundevents.RoundEntityCreatedPayload{
		Round: roundtypes.Round{
			ID:           roundID,
			Title:        "Quick Round",
			Description:  &description,
			Location:     &location,
			StartTime:    &startTime,
			CreatedBy:    userID,
			State:        roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{},
			Finalized:    false,
		},
	}
}

// expectStoreSuccess validates successful round storage
func expectStoreSuccess(t *testing.T, helper *testutils.RoundTestHelper, originalPayload roundevents.RoundEntityCreatedPayload, timeout time.Duration) {
	t.Helper()

	if !helper.WaitForRoundCreated(1, timeout) {
		t.Fatalf("Expected round.created message within %v", timeout)
	}

	msgs := helper.GetRoundCreatedMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 success message, got %d", len(msgs))
	}

	result := helper.ValidateRoundCreated(t, msgs[0], originalPayload.Round.ID)

	// Validate that the stored round matches the original
	if result.RoundID != originalPayload.Round.ID {
		t.Errorf("RoundID mismatch: expected %s, got %s", originalPayload.Round.ID, result.RoundID)
	}
	// Add more detailed field comparisons if necessary
}

// TestHandleRoundEntityCreated runs integration tests for the round entity created handler
func TestHandleRoundEntityCreated(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper)
	}{
		{
			name: "Success - Handler Processes Valid Message and Publishes Success Event",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				payload := createValidRoundEntityCreatedPayload(userID)
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				expectStoreSuccess(t, helper, payload, 3*time.Second)
			},
		},
		{
			name: "Success - Handler Processes Minimal Valid Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				payload := createMinimalRoundEntityCreatedPayload(userID)
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				expectStoreSuccess(t, helper, payload, 3*time.Second)
			},
		},
		{
			name: "Success - Handler Processes Message with Complex Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				payload := createValidRoundEntityCreatedPayload(userID)
				payload.Round.Participants = []roundtypes.Participant{
					{UserID: userID, Response: roundtypes.ResponseAccept},
					{UserID: sharedtypes.DiscordID("user123"), Response: roundtypes.ResponseTentative},
					{UserID: sharedtypes.DiscordID("user456"), Response: roundtypes.ResponseDecline},
				}
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				expectStoreSuccess(t, helper, payload, 3*time.Second)
			},
		},
		{
			name: "Success - Handler Processes Different Round States",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				payload := createValidRoundEntityCreatedPayload(userID)
				payload.Round.State = roundtypes.RoundStateInProgress
				helper.PublishRoundEntityCreated(t, context.Background(), payload)
				expectStoreSuccess(t, helper, payload, 3*time.Second)
			},
		},
		{
			name: "Success - Handler Processes Multiple Messages Concurrently",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				payload1 := createValidRoundEntityCreatedPayload(userID)
				// Ensure payload2 has a different RoundID for distinctness
				payload2 := createValidRoundEntityCreatedPayload(userID)
				payload2.Round.ID = sharedtypes.RoundID(uuid.New())

				// Publish both quickly to test concurrent processing
				helper.PublishRoundEntityCreated(t, context.Background(), payload1)
				helper.PublishRoundEntityCreated(t, context.Background(), payload2)

				// Both should be processed successfully
				if !helper.WaitForRoundCreated(2, 5*time.Second) {
					t.Fatalf("Expected 2 success messages (round.created) within 5s, got %d", len(helper.GetRoundCreatedMessages()))
				}

				msgs := helper.GetRoundCreatedMessages()
				if len(msgs) != 2 {
					t.Fatalf("Expected 2 success messages (round.created), got %d", len(msgs))
				}
			},
		},
		{
			name: "Failure - Handler Rejects Invalid JSON and Doesn't Publish Events",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				helper.PublishInvalidJSON(t, context.Background(), roundevents.RoundEntityCreated)
				expectNoMessages(t, helper, 2*time.Second) // This will now check output topics
			},
		},
		{
			name: "Success - Handler Preserves Message Correlation ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper) {
				payload := createValidRoundEntityCreatedPayload(userID)
				originalMsg := helper.PublishRoundEntityCreated(t, context.Background(), payload)

				if !helper.WaitForRoundCreated(1, 3*time.Second) {
					t.Fatalf("Expected success message (round.created)")
				}

				msgs := helper.GetRoundCreatedMessages()
				if len(msgs) == 0 {
					t.Fatalf("No round.created messages captured")
				}
				resultMsg := msgs[0]

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
				payload := createValidRoundEntityCreatedPayload(userID)
				helper.PublishRoundEntityCreated(t, context.Background(), payload)

				if !helper.WaitForRoundCreated(1, 3*time.Second) {
					t.Fatalf("Expected success message (round.created)")
				}

				// Verify no messages on wrong topics
				failureMsgs := helper.GetRoundCreationFailedMessages()
				if len(failureMsgs) > 0 {
					t.Errorf("Handler published to wrong topic - got %d round.creation.failed messages", len(failureMsgs))
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			helper.ClearMessages()
			tc.setupAndRun(t, helper)
		})
	}
}
