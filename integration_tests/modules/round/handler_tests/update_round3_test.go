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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Update Schedule for Round with No Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round with no participants
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Get original round for verification - use the correct DB reference
				originalRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get original round: %v", err)
				}

				// Also verify the round exists using both DB interfaces
				roundViaDirectDB, err2 := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err2 != nil {
					t.Fatalf("Failed to get round via direct DB: %v", err2)
				}

				// Verify both DB interfaces return the same data
				if originalRound.ID != roundViaDirectDB.ID {
					t.Errorf("DB interfaces return different IDs: DBService=%s, DirectDB=%s", originalRound.ID, roundViaDirectDB.ID)
				}

				// Create schedule update payload representing the updated round
				futureTime := time.Now().Add(24 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Updated Location")

				// Get the actual round and create a payload that represents it being updated
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				updatedRound.Title = roundtypes.Title("Updated Schedule Title")
				updatedRound.StartTime = &startTime
				updatedRound.Location = &location

				payload := roundevents.RoundEntityUpdatedPayloadV1{
					GuildID: "test-guild",
					Round:   *updatedRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully (no response message expected)
				t.Logf("Round schedule update completed for round %s", roundID)
			},
		},
		{
			name: "Success - Update Schedule for Round with Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				// Create a round with multiple participants
				score1 := sharedtypes.Score(2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Create schedule update payload for round with participants
				futureTime := time.Now().Add(48 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Multi-Participant Updated Location")

				// Get the actual round and create a payload that represents it being updated
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				updatedRound.Title = roundtypes.Title("Multi-Participant Schedule Update")
				updatedRound.StartTime = &startTime
				updatedRound.Location = &location

				payload := roundevents.RoundEntityUpdatedPayloadV1{
					GuildID: "test-guild",
					Round:   *updatedRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully
				t.Logf("Round schedule update completed for round %s with 2 participants", roundID)
			},
		},
		{
			name: "Success - Update Schedule with Minimal Payload Data",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				// Create a round
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create schedule update payload with minimal data
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				updatedRound.Title = roundtypes.Title("Minimal Update")

				payload := roundevents.RoundEntityUpdatedPayloadV1{
					GuildID: "test-guild",
					Round:   *updatedRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully
				t.Logf("Round schedule update completed for round %s", roundID)
			},
		},
		{
			name: "Success - Update Schedule for Round with All Field Types",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				// Create a round with comprehensive data
				score1 := sharedtypes.Score(5)
				score2 := sharedtypes.Score(-2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseDecline, Score: &score2},
				})

				// Get original round to have additional data for verification
				originalRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get original round: %v", err)
				}

				// Create comprehensive schedule update payload
				futureTime := time.Now().Add(72 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				location := roundtypes.Location("Comprehensive Update Location")

				// Get the actual round and create a payload that represents it being updated
				comprehensiveRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get round: %v", err)
				}
				comprehensiveRound.Title = roundtypes.Title("Comprehensive Schedule Update")
				comprehensiveRound.StartTime = &startTime
				comprehensiveRound.Location = &location

				payload := roundevents.RoundEntityUpdatedPayloadV1{
					GuildID: "test-guild",
					Round:   *comprehensiveRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update completed successfully - verify round still exists
				updatedRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
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
				if p, exists := participantMap[data2.UserID]; exists {
					if p.Score == nil || *p.Score != score1 {
						t.Errorf("Expected user2 score %d, got %v", score1, p.Score)
					}
				}
				if p, exists := participantMap[data3.UserID]; exists {
					if p.Score == nil || *p.Score != score2 {
						t.Errorf("Expected user3 score %d, got %v", score2, p.Score)
					}
				}
			},
		},
		{
			name: "Success - Schedule Update with Invalid Future Time (No Error Expected)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a real round first
				existingRoundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})
				existingRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", existingRoundID)
				if err != nil {
					t.Fatalf("Failed to get existing round: %v", err)
				}

				// Update the round with a valid future time
				futureTime := time.Now().Add(24 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				existingRound.StartTime = &startTime
				existingRound.Title = roundtypes.Title("Updated Schedule Title")

				// Create RoundEntityUpdatedPayload (which is what the handler expects)
				payload := roundevents.RoundEntityUpdatedPayloadV1{
					GuildID: "test-guild",
					Round:   *existingRound,
				}

				publishRoundEntityUpdatedForScheduleUpdate(t, deps, deps.MessageCapture, payload)

				// Schedule update should complete without error
				t.Logf("Round schedule update completed for round %s", existingRoundID)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoRoundScheduleUpdateMessages(t, deps, deps.MessageCapture)
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

func publishInvalidJSONAndExpectNoRoundScheduleUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	// FIX: Use the correct input topic that the handler listens to
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdatedV1, invalidMsg); err != nil {
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
	return capture.GetMessages(roundevents.RoundScheduleUpdatedV1)
}

func getRoundScheduleUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdateErrorV1)
}

// Test expectation functions - UNIQUE TO ROUND SCHEDULE UPDATE TESTS
func publishRoundEntityUpdatedForScheduleUpdate(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundEntityUpdatedPayloadV1) {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdatedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// HandleRoundScheduleUpdate returns empty slice on success (no messages published)
	// Just wait a bit to ensure the handler processed the message
	time.Sleep(100 * time.Millisecond)

	// Verify no error messages were published
	errorMsgs := capture.GetMessages(roundevents.RoundUpdateErrorV1)
	if len(errorMsgs) > 0 {
		t.Fatalf("Expected no error messages but got %d error messages", len(errorMsgs))
	}
}
