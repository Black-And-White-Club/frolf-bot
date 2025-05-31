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
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	users := generator.GenerateUsers(3)
	user1ID := sharedtypes.DiscordID(users[0].UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Process Round Reminder for Upcoming Round with Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create an upcoming round with ACCEPTED participants
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, user1ID, response, &tagNumber)

				// Create reminder payload
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(roundID, "15min", []sharedtypes.DiscordID{user1ID}, &reminderTime)

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
				// Create a round but no accepted participants (just the creator)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateUpcoming)

				// Create reminder
				reminderTime := time.Now().Add(30 * time.Minute)
				payload := createRoundReminderPayload(roundID, "30min", []sharedtypes.DiscordID{user1ID}, &reminderTime)

				// Expect no messages since no participants
				publishAndExpectNoReminderMessages(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Process Round Reminder with Empty User List",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create round but send reminder with no users (edge case)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateUpcoming)

				// Create reminder with empty user list
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(roundID, "1hour", []sharedtypes.DiscordID{}, &reminderTime)

				publishAndExpectNoReminderMessages(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Process Reminder for Non-Existent Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				reminderTime := time.Now().Add(1 * time.Hour)
				payload := createRoundReminderPayload(nonExistentRoundID, "15min", []sharedtypes.DiscordID{user1ID}, &reminderTime)

				publishAndExpectReminderError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoReminderMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			helper.ClearMessages()
			tc.setupAndRun(t, helper, &deps)

			time.Sleep(1 * time.Second)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO ROUND REMINDER TESTS
func createRoundReminderPayload(roundID sharedtypes.RoundID, reminderType string, userIDs []sharedtypes.DiscordID, startTime *time.Time) roundevents.DiscordReminderPayload {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		converted := sharedtypes.StartTime(*startTime)
		sharedStartTime = &converted
	}

	return roundevents.DiscordReminderPayload{
		RoundID:          roundID,
		ReminderType:     reminderType,
		RoundTitle:       roundtypes.Title("Test Round"),
		StartTime:        sharedStartTime,
		Location:         (*roundtypes.Location)(stringPtr("Test Location")),
		UserIDs:          userIDs,
		DiscordChannelID: "test-channel-123",
		DiscordGuildID:   "test-guild-456",
		EventMessageID:   "test-message-789",
	}
}

// Publishing functions - UNIQUE TO ROUND REMINDER TESTS
func publishRoundReminderMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.DiscordReminderPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundReminder, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoReminderMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundReminder, invalidMsg); err != nil {
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
	return capture.WaitForMessages(roundevents.DiscordRoundReminder, count, defaultTimeout)
}

func waitForReminderErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND REMINDER TESTS
func getReminderSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.DiscordRoundReminder)
}

func getReminderErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundError)
}

// Validation functions - UNIQUE TO ROUND REMINDER TESTS
func validateReminderSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.DiscordReminderPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.DiscordReminderPayload](msg)
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

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayload](msg)
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
func publishAndExpectReminderSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.DiscordReminderPayload) *roundevents.DiscordReminderPayload {
	publishRoundReminderMessage(t, deps, &payload)

	if !waitForReminderSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected reminder success message from %s", roundevents.DiscordRoundReminder)
	}

	msgs := getReminderSuccessFromHandlerMessages(capture)
	result := validateReminderSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectReminderError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.DiscordReminderPayload) {
	publishRoundReminderMessage(t, deps, &payload)

	if !waitForReminderErrorFromHandler(capture, 1) {
		t.Fatalf("Expected reminder error message from %s", roundevents.RoundError)
	}

	msgs := getReminderErrorFromHandlerMessages(capture)
	validateReminderErrorFromHandler(t, msgs[0], payload.RoundID)
}

func publishAndExpectNoReminderMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.DiscordReminderPayload) {
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

// Helper utility functions
func stringPtr(s string) *string {
	return &s
}
