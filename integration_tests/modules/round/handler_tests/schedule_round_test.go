package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleDiscordMessageIDUpdated(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Schedule Events for Future Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round scheduled for 2 hours in the future
				startTime := time.Now().Add(2 * time.Hour)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				// Create schedule payload
				payload := createScheduleRoundPayload(roundID, "Test Round", &startTime, "test-message-123")

				publishAndExpectScheduleSuccess(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Schedule Events for Round Less Than 1 Hour Away",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round scheduled for 30 minutes in the future (should skip reminder)
				startTime := time.Now().Add(30 * time.Minute)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				// Create schedule payload
				payload := createScheduleRoundPayload(roundID, "Test Round", &startTime, "test-message-456")

				publishAndExpectScheduleSuccess(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Schedule Events for Round Far in Future",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round scheduled for 1 day in the future
				startTime := time.Now().Add(24 * time.Hour)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				// Create schedule payload
				payload := createScheduleRoundPayload(roundID, "Future Round", &startTime, "test-message-789")

				publishAndExpectScheduleSuccess(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Handle Round with Past Start Time",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round with start time in the past
				startTime := time.Now().Add(-1 * time.Hour)
				roundID := helper.CreateRoundInDBWithTime(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming, &startTime)

				// Create schedule payload
				payload := createScheduleRoundPayload(roundID, "Past Round", &startTime, "test-message-past")

				publishAndExpectScheduleSuccess(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoScheduleMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO SCHEDULE ROUND TESTS
func createScheduleRoundPayload(roundID sharedtypes.RoundID, title string, startTime *time.Time, eventMessageID string) roundevents.RoundScheduledPayloadV1 {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		converted := sharedtypes.StartTime(*startTime)
		sharedStartTime = &converted
	}

	desc := roundtypes.Description("Test Description")
	loc := roundtypes.Location("Test Location")
	return roundevents.RoundScheduledPayloadV1{
		GuildID: "test-guild",
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     roundID,
			Title:       roundtypes.Title(title),
			Description: &desc,
			Location:    &loc,
			StartTime:   sharedStartTime,
		},
		EventMessageID: eventMessageID,
	}
}

// Publishing functions - UNIQUE TO SCHEDULE ROUND TESTS
func publishScheduleRoundMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.RoundScheduledPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundEventMessageIDUpdatedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScheduleMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	// Publish to the correct topic that the handler listens to.
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundEventMessageIDUpdatedV1, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	errorMsgs := getScheduleErrorFromHandlerMessages(capture)

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO SCHEDULE ROUND TESTS
func waitForScheduleErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO SCHEDULE ROUND TESTS
func getScheduleErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundErrorV1)
}

// Validation functions - UNIQUE TO SCHEDULE ROUND TESTS
func validateScheduleErrorFromHandler(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse schedule error message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// Test expectation functions - UNIQUE TO SCHEDULE ROUND TESTS
func publishAndExpectScheduleSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundScheduledPayloadV1) {
	publishScheduleRoundMessage(t, deps, &payload)

	// Wait a bit to ensure processing completes
	time.Sleep(500 * time.Millisecond)

	// Since this handler returns empty slice on success, we just check that no errors occurred
	errorMsgs := getScheduleErrorFromHandlerMessages(capture)

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for successful scheduling, got %d", len(errorMsgs))
		// Log the error for debugging
		if len(errorMsgs) > 0 {
			result, err := testutils.ParsePayload[roundevents.RoundErrorPayloadV1](errorMsgs[0])
			if err == nil {
				t.Logf("Error message: %s", result.Error)
			}
		}
	}
}
