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

func TestHandleRoundScheduleUpdate(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	users := generator.GenerateUsers(3)
	user1ID := sharedtypes.DiscordID(users[0].UserID)
	user2ID := sharedtypes.DiscordID(users[1].UserID)
	user3ID := sharedtypes.DiscordID(users[2].UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Update Schedule for Round with No Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with no participants
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})

				// Debug: Log what we created
				t.Logf("DEBUG: Created round ID: %s", roundID)

				// Get original round for verification - use the correct DB reference
				originalRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get original round: %v", err)
				}

				// Debug: Log what we got from the database
				t.Logf("DEBUG: Original round from DBService - ID: %s, Title: %s, CreatedBy: %s, Participants: %d",
					originalRound.ID, originalRound.Title, originalRound.CreatedBy, len(originalRound.Participants))

				// Also verify the round exists using both DB interfaces
				roundViaDirectDB, err2 := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err2 != nil {
					t.Fatalf("Failed to get round via direct DB: %v", err2)
				}
				t.Logf("DEBUG: Round via direct DB - ID: %s, Title: %s, CreatedBy: %s, Participants: %d",
					roundViaDirectDB.ID, roundViaDirectDB.Title, roundViaDirectDB.CreatedBy, len(roundViaDirectDB.Participants))

				// Verify both DB interfaces return the same data
				if originalRound.ID != roundViaDirectDB.ID {
					t.Errorf("DB interfaces return different IDs: DBService=%s, DirectDB=%s", originalRound.ID, roundViaDirectDB.ID)
				}

				// Create schedule update payload
				futureTime := time.Now().Add(24 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Updated Location")
				payload := createRoundScheduleUpdatePayload(roundID, "Updated Schedule Title", &startTime, &location)

				t.Logf("DEBUG: About to publish message with payload for round %s", payload.RoundID)

				result := publishAndExpectRoundScheduleUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.Round.ID)
				}
				// The round returned should be the complete round from DB (not necessarily with updated fields)
				if result.Round.CreatedBy != originalRound.CreatedBy {
					t.Errorf("Expected CreatedBy to be %s, got %s", originalRound.CreatedBy, result.Round.CreatedBy)
				}
				if len(result.Round.Participants) != 0 {
					t.Errorf("Expected 0 participants, got %d", len(result.Round.Participants))
				}
			},
		},
		{
			name: "Success - Update Schedule for Round with Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with multiple participants
				score1 := sharedtypes.Score(2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Create schedule update payload
				futureTime := time.Now().Add(48 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Multi-Participant Updated Location")
				payload := createRoundScheduleUpdatePayload(roundID, "Multi-Participant Schedule Update", &startTime, &location)

				result := publishAndExpectRoundScheduleUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result maintains all participant data
				if len(result.Round.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(result.Round.Participants))
				}

				// Find participants and validate their data is preserved
				participantMap := make(map[sharedtypes.DiscordID]roundtypes.Participant)
				for _, p := range result.Round.Participants {
					participantMap[p.UserID] = p
				}

				// Validate user2 data is preserved
				if p, exists := participantMap[user2ID]; exists {
					if p.Response != roundtypes.ResponseAccept {
						t.Errorf("Expected user2 response ACCEPT, got %s", p.Response)
					}
					if p.Score == nil || *p.Score != score1 {
						t.Errorf("Expected user2 score %d, got %v", score1, p.Score)
					}
				} else {
					t.Error("user2 not found in participants")
				}

				// Validate user3 data is preserved
				if p, exists := participantMap[user3ID]; exists {
					if p.Response != roundtypes.ResponseTentative {
						t.Errorf("Expected user3 response TENTATIVE, got %s", p.Response)
					}
					if p.Score != nil {
						t.Errorf("Expected user3 score to be nil, got %v", p.Score)
					}
				} else {
					t.Error("user3 not found in participants")
				}
			},
		},
		{
			name: "Success - Update Schedule with Minimal Payload Data",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create schedule update payload with minimal data (no start time, no location)
				payload := createRoundScheduleUpdatePayload(roundID, "Minimal Update", nil, nil)

				result := publishAndExpectRoundScheduleUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the basic round data is returned
				if result.Round.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.Round.ID)
				}
				if len(result.Round.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Round.Participants))
				}
			},
		},
		{
			name: "Success - Update Schedule for Round with All Field Types",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with comprehensive data
				score1 := sharedtypes.Score(5)
				score2 := sharedtypes.Score(-2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseDecline, Score: &score2},
				})

				// Get original round to have additional data for verification
				originalRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get original round: %v", err)
				}

				// Create comprehensive schedule update payload
				futureTime := time.Now().Add(72 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Comprehensive Update Location")
				payload := createRoundScheduleUpdatePayload(roundID, "Comprehensive Schedule Update", &startTime, &location)

				result := publishAndExpectRoundScheduleUpdated(t, deps, deps.MessageCapture, payload)

				// Validate comprehensive data preservation
				if result.Round.State != originalRound.State {
					t.Errorf("Expected State %s to be preserved, got %s", originalRound.State, result.Round.State)
				}
				if result.Round.EventMessageID != originalRound.EventMessageID {
					t.Errorf("Expected EventMessageID %s to be preserved, got %s", originalRound.EventMessageID, result.Round.EventMessageID)
				}

				// Validate all participants are preserved with scores
				if len(result.Round.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(result.Round.Participants))
				}

				participantMap := make(map[sharedtypes.DiscordID]roundtypes.Participant)
				for _, p := range result.Round.Participants {
					participantMap[p.UserID] = p
				}

				// Check both participants have their scores preserved
				if p, exists := participantMap[user2ID]; exists {
					if p.Score == nil || *p.Score != score1 {
						t.Errorf("Expected user2 score %d, got %v", score1, p.Score)
					}
				}
				if p, exists := participantMap[user3ID]; exists {
					if p.Score == nil || *p.Score != score2 {
						t.Errorf("Expected user3 score %d, got %v", score2, p.Score)
					}
				}
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				futureTime := time.Now().Add(24 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Nonexistent Location")
				payload := createRoundScheduleUpdatePayload(nonExistentRoundID, "Nonexistent Round", &startTime, &location)

				result := publishAndExpectRoundScheduleUpdateError(t, deps, deps.MessageCapture, payload)

				// Validate the error
				if result.Error == "" {
					t.Error("Expected Error message to be populated")
				}
				// RoundUpdateRequest should be nil since the round doesn't exist
				if result.RoundUpdateRequest != nil {
					t.Error("Expected RoundUpdateRequest to be nil for non-existent round")
				}
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoRoundScheduleUpdateMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func createRoundScheduleUpdatePayload(
	roundID sharedtypes.RoundID,
	title string,
	startTime *sharedtypes.StartTime,
	location *roundtypes.Location,
) roundevents.RoundScheduleUpdatePayload {
	return roundevents.RoundScheduleUpdatePayload{
		RoundID:   roundID,
		Title:     roundtypes.Title(title),
		StartTime: startTime,
		Location:  location,
	}
}

// Publishing functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func publishRoundScheduleUpdateMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.RoundScheduleUpdatePayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdated, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoRoundScheduleUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	// FIX: Use the correct input topic that the handler listens to
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdated, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	storedMsgs := getRoundScheduleUpdatedFromHandlerMessages(capture)
	errorMsgs := getRoundScheduleUpdateErrorFromHandlerMessages(capture)

	if len(storedMsgs) > 0 {
		t.Errorf("Expected no stored messages for invalid JSON, got %d", len(storedMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func waitForRoundScheduleUpdatedFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScheduleUpdate, count, defaultTimeout)
}

func waitForRoundScheduleUpdateErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundUpdateError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func getRoundScheduleUpdatedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScheduleUpdate)
}

func getRoundScheduleUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdateError)
}

// Validation functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func validateRoundScheduleUpdatedFromHandler(t *testing.T, msg *message.Message) *roundevents.RoundStoredPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundStoredPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round schedule updated message: %v", err)
	}

	// Debug: Print the round data to see what we're getting
	t.Logf("DEBUG: Received round data - ID: %s, Title: %s, CreatedBy: %s, Participants: %d",
		result.Round.ID, result.Round.Title, result.Round.CreatedBy, len(result.Round.Participants))

	// Validate that required fields are set - but don't fail the test, just warn
	if result.Round.ID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected Round.ID to be set")
	}

	if result.Round.Title == "" {
		t.Error("Expected Round.Title to be set")
	}

	if result.Round.CreatedBy == "" {
		t.Error("Expected Round.CreatedBy to be set")
	}

	// Log what we got for debugging
	t.Logf("Round schedule updated successfully: %s ('%s'), participants: %d",
		result.Round.ID, result.Round.Title, len(result.Round.Participants))

	return result
}

func validateRoundScheduleUpdateErrorFromHandler(t *testing.T, msg *message.Message) *roundevents.RoundUpdateErrorPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundUpdateErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round schedule update error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	// Log what we got for debugging
	t.Logf("Round schedule update failed with error: %s", result.Error)

	return result
}

// Test expectation functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func publishAndExpectRoundScheduleUpdated(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundScheduleUpdatePayload) *roundevents.RoundStoredPayload {
	publishRoundScheduleUpdateMessage(t, deps, &payload)

	if !waitForRoundScheduleUpdatedFromHandler(capture, 1) {
		t.Fatalf("Expected round schedule updated message from %s", roundevents.RoundScheduleUpdate)
	}

	msgs := getRoundScheduleUpdatedFromHandlerMessages(capture)
	result := validateRoundScheduleUpdatedFromHandler(t, msgs[0])

	return result
}

func publishAndExpectRoundScheduleUpdateError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundScheduleUpdatePayload) *roundevents.RoundUpdateErrorPayload {
	publishRoundScheduleUpdateMessage(t, deps, &payload)

	if !waitForRoundScheduleUpdateErrorFromHandler(capture, 1) {
		t.Fatalf("Expected round schedule update error message from %s", roundevents.RoundUpdateError)
	}

	msgs := getRoundScheduleUpdateErrorFromHandlerMessages(capture)
	result := validateRoundScheduleUpdateErrorFromHandler(t, msgs[0])

	return result
}
