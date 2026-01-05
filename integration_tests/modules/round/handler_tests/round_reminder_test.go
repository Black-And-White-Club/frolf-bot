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

func TestHandleRoundReminder(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Process Round Reminder for Upcoming Round with Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create an upcoming round with ACCEPTED participants
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, response, &tagNumber)

				// Create reminder payload
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(roundID, "15min", []sharedtypes.DiscordID{data.UserID}, &reminderTime)

				result := publishAndExpectReminderSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
			},
		},
		{
			name: "Success - Process Round Reminder with No Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round but no accepted participants (just the creator)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)

				// Create reminder
				reminderTime := time.Now().Add(30 * time.Minute)
				payload := createRoundReminderPayload(roundID, "30min", []sharedtypes.DiscordID{data.UserID}, &reminderTime)

				// Expect no messages since no participants
				publishAndExpectNoReminderMessages(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Process Round Reminder with Empty User List",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create round but send reminder with no users (edge case)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)

				// Create reminder with empty user list
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(roundID, "1hour", []sharedtypes.DiscordID{}, &reminderTime)

				publishAndExpectNoReminderMessages(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Process Reminder for Non-Existent Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(nonExistentRoundID, "15min", []sharedtypes.DiscordID{data.UserID}, &reminderTime)

				publishAndExpectReminderError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoReminderMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create helper for each subtest since these tests use custom helper functions
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO ROUND REMINDER TESTS
func createRoundReminderPayload(roundID sharedtypes.RoundID, reminderType string, userIDs []sharedtypes.DiscordID, startTime *time.Time) roundevents.DiscordReminderPayloadV1 {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		converted := sharedtypes.StartTime(*startTime)
		sharedStartTime = &converted
	}

	location := roundtypes.Location("Test Location")
	return roundevents.DiscordReminderPayloadV1{
		GuildID:          "test-guild",
		RoundID:          roundID,
		ReminderType:     reminderType,
		RoundTitle:       roundtypes.Title("Test Round"),
		StartTime:        sharedStartTime,
		Location:         &location,
		UserIDs:          userIDs,
		DiscordChannelID: "test-channel-123",
		DiscordGuildID:   "test-guild-456",
		EventMessageID:   "test-message-789",
	}
}

// Publishing functions - UNIQUE TO ROUND REMINDER TESTS
func publishRoundReminderMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.DiscordReminderPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundReminderScheduledV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoReminderMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundReminderScheduledV1, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	successMsgs := getReminderSuccessFromHandlerMessages(capture)
	errorMsgs := getReminderErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for invalid JSON, got %d", len(successMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO ROUND REMINDER TESTS
func waitForReminderSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundReminderSentV1, count, defaultTimeout)
}

func waitForReminderErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND REMINDER TESTS
func getReminderSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundReminderSentV1)
}

func getReminderErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundErrorV1)
}

// Validation functions - UNIQUE TO ROUND REMINDER TESTS
func validateReminderSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.DiscordReminderPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.DiscordReminderPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse reminder success message: %v", err)
	}

	// Validate that RoundID is set
	if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	// Log what we got for debugging
	t.Logf("Reminder processed successfully for round: %s", result.RoundID)

	return result
}

func validateReminderErrorFromHandler(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse reminder error message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// Test expectation functions - UNIQUE TO ROUND REMINDER TESTS
func publishAndExpectReminderSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.DiscordReminderPayloadV1) *roundevents.DiscordReminderPayloadV1 {
	publishRoundReminderMessage(t, deps, &payload)

	if !waitForReminderSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected reminder success message from %s", roundevents.RoundReminderSentV1)
	}

	msgs := getReminderSuccessFromHandlerMessages(capture)
	result := validateReminderSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectReminderError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.DiscordReminderPayloadV1) {
	publishRoundReminderMessage(t, deps, &payload)

	if !waitForReminderErrorFromHandler(capture, 1) {
		t.Fatalf("Expected reminder error message from %s", roundevents.RoundErrorV1)
	}

	msgs := getReminderErrorFromHandlerMessages(capture)
	validateReminderErrorFromHandler(t, msgs[0], payload.RoundID)
}

func publishAndExpectNoReminderMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.DiscordReminderPayloadV1) {
	publishRoundReminderMessage(t, deps, &payload)

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	successMsgs := getReminderSuccessFromHandlerMessages(capture)
	errorMsgs := getReminderErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for empty participants, got %d", len(successMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for empty participants, got %d", len(errorMsgs))
	}
}
