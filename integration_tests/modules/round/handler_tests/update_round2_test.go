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

func TestHandleRoundUpdateValidated(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	users := generator.GenerateUsers(2)
	user1ID := sharedtypes.DiscordID(users[0].UserID)
	user2ID := sharedtypes.DiscordID(users[1].UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Update Title Only",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round to update
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Get original round for comparison
				originalRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get original round: %v", err)
				}

				// Create validated payload with title update
				newTitle := roundtypes.Title("Updated Round Title")
				payload := createRoundUpdateValidatedPayload(roundID, user1ID, &newTitle, nil, nil, nil, nil)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.Round.ID)
				}
				if result.Round.Title != newTitle {
					t.Errorf("Expected Title '%s', got '%s'", newTitle, result.Round.Title)
				}
				// Other fields should remain unchanged
				if result.Round.CreatedBy != originalRound.CreatedBy {
					t.Errorf("Expected CreatedBy to remain %s, got %s", originalRound.CreatedBy, result.Round.CreatedBy)
				}
				if len(result.Round.Participants) != len(originalRound.Participants) {
					t.Errorf("Expected participants count to remain %d, got %d", len(originalRound.Participants), len(result.Round.Participants))
				}
			},
		},
		{
			name: "Success - Update Description Only",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})

				originalRound, err := deps.DBService.RoundDB.GetRound(context.Background(), "test-guild", roundID)
				if err != nil {
					t.Fatalf("Failed to get original round: %v", err)
				}

				// Create validated payload with description update
				newDesc := roundtypes.Description("Updated description for the round")
				payload := createRoundUpdateValidatedPayload(roundID, user1ID, nil, &newDesc, nil, nil, nil)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.Description == nil || *result.Round.Description != newDesc {
					t.Errorf("Expected Description '%s', got %v", newDesc, result.Round.Description)
				}
				// Title should remain unchanged
				if result.Round.Title != originalRound.Title {
					t.Errorf("Expected Title to remain '%s', got '%s'", originalRound.Title, result.Round.Title)
				}
			},
		},
		{
			name: "Success - Update Location Only",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})

				// Create validated payload with location update
				newLocation := roundtypes.Location("Updated Course Location")
				payload := createRoundUpdateValidatedPayload(roundID, user1ID, nil, nil, &newLocation, nil, nil)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.Location == nil || *result.Round.Location != newLocation {
					t.Errorf("Expected Location '%s', got %v", newLocation, result.Round.Location)
				}
			},
		},
		{
			name: "Success - Update Start Time Only",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})

				// Create validated payload with start time update
				futureTime := time.Now().Add(48 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				payload := createRoundUpdateValidatedPayload(roundID, user1ID, nil, nil, nil, &startTime, nil)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.StartTime == nil {
					t.Error("Expected StartTime to be set")
				} else {
					if time.Time(*result.Round.StartTime).Unix() != futureTime.Unix() {
						t.Errorf("Expected StartTime %v, got %v", futureTime, time.Time(*result.Round.StartTime))
					}
				}
			},
		},
		{
			name: "Success - Update Event Type Only",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})

				// Create validated payload with event type update
				newEventType := roundtypes.DefaultEventType
				payload := createRoundUpdateValidatedPayload(roundID, user1ID, nil, nil, nil, nil, &newEventType)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.Round.EventType == nil || *result.Round.EventType != newEventType {
					t.Errorf("Expected EventType '%s', got %v", newEventType, result.Round.EventType)
				}
			},
		},
		{
			name: "Success - Update Multiple Fields",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Create validated payload with multiple field updates
				newTitle := roundtypes.Title("Multi-Update Round")
				newDesc := roundtypes.Description("Updated with multiple fields")
				newLocation := roundtypes.Location("New Multi-Field Location")
				futureTime := time.Now().Add(72 * time.Hour)
				startTime := sharedtypes.StartTime(futureTime)
				newEventType := roundtypes.DefaultEventType

				payload := createRoundUpdateValidatedPayload(roundID, user1ID, &newTitle, &newDesc, &newLocation, &startTime, &newEventType)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate all updated fields
				if result.Round.Title != newTitle {
					t.Errorf("Expected Title '%s', got '%s'", newTitle, result.Round.Title)
				}
				if result.Round.Description == nil || *result.Round.Description != newDesc {
					t.Errorf("Expected Description '%s', got %v", newDesc, result.Round.Description)
				}
				if result.Round.Location == nil || *result.Round.Location != newLocation {
					t.Errorf("Expected Location '%s', got %v", newLocation, result.Round.Location)
				}
				if result.Round.StartTime == nil {
					t.Error("Expected StartTime to be set")
				}
				if result.Round.EventType == nil || *result.Round.EventType != newEventType {
					t.Errorf("Expected EventType '%s', got %v", newEventType, result.Round.EventType)
				}
				// Participants should be preserved
				if len(result.Round.Participants) != 1 {
					t.Errorf("Expected 1 participant to be preserved, got %d", len(result.Round.Participants))
				}
			},
		},
		{
			name: "Success - Update Round with Existing Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create round with multiple participants
				score1 := sharedtypes.Score(3)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
				})

				// Update only the title
				newTitle := roundtypes.Title("Updated Round with Participants")
				payload := createRoundUpdateValidatedPayload(roundID, user1ID, &newTitle, nil, nil, nil, nil)

				result := publishAndExpectRoundEntityUpdated(t, deps, deps.MessageCapture, payload)

				// Validate participants are preserved exactly
				if len(result.Round.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Round.Participants))
				}
				if result.Round.Participants[0].UserID != user2ID {
					t.Errorf("Expected participant %s, got %s", user2ID, result.Round.Participants[0].UserID)
				}
				if result.Round.Participants[0].Score == nil || *result.Round.Participants[0].Score != score1 {
					t.Errorf("Expected participant score %d, got %v", score1, result.Round.Participants[0].Score)
				}
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				newTitle := roundtypes.Title("Title for Nonexistent Round")
				payload := createRoundUpdateValidatedPayload(nonExistentRoundID, user1ID, &newTitle, nil, nil, nil, nil)

				result := publishAndExpectRoundUpdateEntityError(t, deps, deps.MessageCapture, payload)

				// Validate the error
				if result.RoundUpdateRequest == nil {
					t.Error("Expected RoundUpdateRequest to be set in error payload")
				}
				if result.Error == "" {
					t.Error("Expected Error message to be populated")
				}
				if result.RoundUpdateRequest.RoundID != nonExistentRoundID {
					t.Errorf("Expected error RoundID %s, got %s", nonExistentRoundID, result.RoundUpdateRequest.RoundID)
				}
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoRoundUpdateEntityMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func createRoundUpdateValidatedPayload(
	roundID sharedtypes.RoundID,
	userID sharedtypes.DiscordID,
	title *roundtypes.Title,
	description *roundtypes.Description,
	location *roundtypes.Location,
	startTime *sharedtypes.StartTime,
	eventType *roundtypes.EventType,
) roundevents.RoundUpdateValidatedPayload {
	// Create the inner request payload
	requestPayload := roundevents.RoundUpdateRequestPayload{}
	requestPayload.RoundID = roundID
	requestPayload.UserID = userID
	requestPayload.GuildID = "test-guild" // Always set for multi-tenant correctness

	// Set optional fields if provided
	if title != nil {
		requestPayload.Title = *title
	}
	if description != nil {
		requestPayload.Description = description
	}
	if location != nil {
		requestPayload.Location = location
	}
	if startTime != nil {
		requestPayload.StartTime = startTime
	}
	if eventType != nil {
		requestPayload.EventType = eventType
	}

	return roundevents.RoundUpdateValidatedPayload{
		GuildID:                   "test-guild",
		RoundUpdateRequestPayload: requestPayload,
	}
}

// Publishing functions - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func publishRoundUpdateValidatedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.RoundUpdateValidatedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdateValidated, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoRoundUpdateEntityMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundUpdateValidated, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	updatedMsgs := getRoundEntityUpdatedFromHandlerMessages(capture)
	errorMsgs := getRoundUpdateEntityErrorFromHandlerMessages(capture)

	if len(updatedMsgs) > 0 {
		t.Errorf("Expected no updated messages for invalid JSON, got %d", len(updatedMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func waitForRoundEntityUpdatedFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundUpdated, count, defaultTimeout)
}

func waitForRoundUpdateEntityErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundUpdateError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func getRoundEntityUpdatedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdated)
}

func getRoundUpdateEntityErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdateError)
}

// Validation functions - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func validateRoundEntityUpdatedFromHandler(t *testing.T, msg *message.Message) *roundevents.RoundEntityUpdatedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundEntityUpdatedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round entity updated message: %v", err)
	}

	// Validate that required fields are set
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
	t.Logf("Round entity updated successfully: %s ('%s')",
		result.Round.ID, result.Round.Title)

	return result
}

func validateRoundUpdateEntityErrorFromHandler(t *testing.T, msg *message.Message) *roundevents.RoundUpdateErrorPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundUpdateErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round update entity error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	if result.RoundUpdateRequest == nil {
		t.Error("Expected RoundUpdateRequest to be set")
	}

	// Log what we got for debugging
	t.Logf("Round entity update failed with error: %s", result.Error)

	return result
}

// Test expectation functions - UNIQUE TO ROUND UPDATE VALIDATED TESTS
func publishAndExpectRoundEntityUpdated(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundUpdateValidatedPayload) *roundevents.RoundEntityUpdatedPayload {
	publishRoundUpdateValidatedMessage(t, deps, &payload)

	if !waitForRoundEntityUpdatedFromHandler(capture, 1) {
		t.Fatalf("Expected round entity updated message from %s", roundevents.RoundUpdated)
	}

	msgs := getRoundEntityUpdatedFromHandlerMessages(capture)
	result := validateRoundEntityUpdatedFromHandler(t, msgs[0])

	return result
}

func publishAndExpectRoundUpdateEntityError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundUpdateValidatedPayload) *roundevents.RoundUpdateErrorPayload {
	publishRoundUpdateValidatedMessage(t, deps, &payload)

	if !waitForRoundUpdateEntityErrorFromHandler(capture, 1) {
		t.Fatalf("Expected round update entity error message from %s", roundevents.RoundUpdateError)
	}

	msgs := getRoundUpdateEntityErrorFromHandlerMessages(capture)
	result := validateRoundUpdateEntityErrorFromHandler(t, msgs[0])

	return result
}
