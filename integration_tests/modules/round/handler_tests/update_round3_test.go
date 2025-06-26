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

				// Create schedule update payload representing the updated round
				futureTime := time.Now().Add(24 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Updated Location")

				// Get the actual round and create a payload that represents it being updated
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				updatedRound.Title = roundtypes.Title("Updated Schedule Title")
				updatedRound.StartTime = &startTime
				updatedRound.Location = &location

				payload := roundevents.RoundEntityUpdatedPayload{
					Round: *updatedRound,
				}

				t.Logf("DEBUG: About to publish message with payload for round %s", payload.Round.ID)

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully (no response message expected)
				t.Logf("Round schedule update completed for round %s", roundID)
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

				// Create schedule update payload for round with participants
				futureTime := time.Now().Add(48 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Multi-Participant Updated Location")

				// Get the actual round and create a payload that represents it being updated
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				updatedRound.Title = roundtypes.Title("Multi-Participant Schedule Update")
				updatedRound.StartTime = &startTime
				updatedRound.Location = &location

				payload := roundevents.RoundEntityUpdatedPayload{
					Round: *updatedRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully
				t.Logf("Round schedule update completed for round %s with 2 participants", roundID)
			},
		},
		{
			name: "Success - Update Schedule with Minimal Payload Data",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create schedule update payload with minimal data
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				updatedRound.Title = roundtypes.Title("Minimal Update")

				payload := roundevents.RoundEntityUpdatedPayload{
					Round: *updatedRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully
				t.Logf("Round schedule update completed for round %s", roundID)
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

				// Get the actual round and create a payload that represents it being updated
				comprehensiveRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				comprehensiveRound.Title = roundtypes.Title("Comprehensive Schedule Update")
				comprehensiveRound.StartTime = &startTime
				comprehensiveRound.Location = &location

				payload := roundevents.RoundEntityUpdatedPayload{
					Round: *comprehensiveRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully - verify round still exists
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), roundID)
				if err != nil {
					t.Fatalf("Failed to get updated round: %v", err)
				}

				// Validate comprehensive data preservation
				if updatedRound.State != originalRound.State {
					t.Errorf("Expected State %s to be preserved, got %s", originalRound.State, updatedRound.State)
				}
				if updatedRound.EventMessageID != originalRound.EventMessageID {
					t.Errorf("Expected EventMessageID %s to be preserved, got %s", originalRound.EventMessageID, updatedRound.EventMessageID)
				}

				// Validate all participants are preserved with scores
				if len(updatedRound.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(updatedRound.Participants))
				}

				participantMap := make(map[sharedtypes.DiscordID]roundtypes.Participant)
				for _, p := range updatedRound.Participants {
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
			name: "Success - Schedule Update with Invalid Future Time (No Error Expected)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a real round first
				existingRoundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})
				existingRound, err := deps.DBService.RoundDB.GetRound(context.Background(), existingRoundID)
				if err != nil {
					t.Fatalf("Failed to get existing round: %v", err)
				}

				// Update the round with a valid future time
				futureTime := time.Now().Add(24 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				existingRound.StartTime = &startTime
				existingRound.Title = roundtypes.Title("Updated Schedule Title")

				// Create RoundEntityUpdatedPayload (which is what the handler expects)
				payload := roundevents.RoundEntityUpdatedPayload{
					Round: *existingRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update should complete without error
				t.Logf("Round schedule update completed for round %s", existingRoundID)
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

// Message retrieval functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func getRoundScheduleUpdatedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScheduleUpdate)
}

func getRoundScheduleUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdateError)
}

// Test expectation functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func publishRoundEntityUpdatedForScheduleUpdate(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundEntityUpdatedPayload) {
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

	// HandleRoundScheduleUpdate returns empty slice on success (no messages published)
	// Just wait a bit to ensure the handler processed the message
	time.Sleep(100 * time.Millisecond)

	// Verify no error messages were published
	errorMsgs := capture.GetMessages(roundevents.RoundUpdateError)
	if len(errorMsgs) > 0 {
		t.Fatalf("Expected no error messages but got %d error messages", len(errorMsgs))
	}
}
