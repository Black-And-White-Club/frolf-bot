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

func TestHandleRoundStarted(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	users := generator.GenerateUsers(4)
	user1ID := sharedtypes.DiscordID(users[0].UserID)
	user2ID := sharedtypes.DiscordID(users[1].UserID)
	user3ID := sharedtypes.DiscordID(users[2].UserID)
	user4ID := sharedtypes.DiscordID(users[3].UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Start Round with Single Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with one participant in UPCOMING state
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create round started payload
				startTime := time.Now().Add(time.Hour)
				location := roundtypes.Location("Test Course")
				payload := createRoundStartedPayload(roundID, "Test Round", &startTime, &location)

				result := publishAndExpectRoundStartSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Title != "Test Round" {
					t.Errorf("Expected Title 'Test Round', got %s", result.Title)
				}
				if len(result.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Participants))
				}
				if result.Participants[0].UserID != user2ID {
					t.Errorf("Expected participant %s, got %s", user2ID, result.Participants[0].UserID)
				}
				if result.Participants[0].Response != roundtypes.ResponseAccept {
					t.Errorf("Expected response ACCEPT, got %s", result.Participants[0].Response)
				}
				if result.EventMessageID == "" {
					t.Error("Expected EventMessageID to be set")
				}
			},
		},
		{
			name: "Success - Start Round with Multiple Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with multiple participants
				score1 := sharedtypes.Score(2) // Some participants already have scores
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user4ID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Create round started payload
				startTime := time.Now().Add(30 * time.Minute)
				location := roundtypes.Location("Multiple Player Course")
				payload := createRoundStartedPayload(roundID, "Multi-Player Round", &startTime, &location)

				result := publishAndExpectRoundStartSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if len(result.Participants) != 3 {
					t.Errorf("Expected 3 participants, got %d", len(result.Participants))
				}

				// Check that participants are correctly converted
				participantMap := make(map[sharedtypes.DiscordID]roundevents.RoundParticipant)
				for _, p := range result.Participants {
					participantMap[p.UserID] = p
				}

				// Validate user2 (no score)
				if p, exists := participantMap[user2ID]; exists {
					if p.Response != roundtypes.ResponseAccept {
						t.Errorf("Expected user2 response ACCEPT, got %s", p.Response)
					}
					if p.Score != nil {
						t.Errorf("Expected user2 score to be nil, got %v", p.Score)
					}
				} else {
					t.Error("user2 not found in participants")
				}

				// Validate user3 (has score)
				if p, exists := participantMap[user3ID]; exists {
					if p.Response != roundtypes.ResponseAccept {
						t.Errorf("Expected user3 response ACCEPT, got %s", p.Response)
					}
					if p.Score == nil || *p.Score != score1 {
						t.Errorf("Expected user3 score %d, got %v", score1, p.Score)
					}
				} else {
					t.Error("user3 not found in participants")
				}

				// Validate user4 (tentative)
				if p, exists := participantMap[user4ID]; exists {
					if p.Response != roundtypes.ResponseTentative {
						t.Errorf("Expected user4 response TENTATIVE, got %s", p.Response)
					}
					if p.Score != nil {
						t.Errorf("Expected user4 score to be nil, got %v", p.Score)
					}
				} else {
					t.Error("user4 not found in participants")
				}
			},
		},
		{
			name: "Success - Start Round with No Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with no participants
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{})

				// Create round started payload
				startTime := time.Now().Add(2 * time.Hour)
				location := roundtypes.Location("Empty Course")
				payload := createRoundStartedPayload(roundID, "Solo Round", &startTime, &location)

				result := publishAndExpectRoundStartSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if len(result.Participants) != 0 {
					t.Errorf("Expected 0 participants, got %d", len(result.Participants))
				}
			},
		},
		{
			name: "Success - Start Round with Participant Tag Numbers",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create participants with tag numbers
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create round started payload
				startTime := time.Now().Add(time.Hour)
				location := roundtypes.Location("Tagged Course")
				payload := createRoundStartedPayload(roundID, "Tagged Round", &startTime, &location)

				result := publishAndExpectRoundStartSuccess(t, deps, deps.MessageCapture, payload)

				// Validate that tag numbers are preserved
				if len(result.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(result.Participants))
				}

				for _, p := range result.Participants {
					if p.TagNumber == nil {
						t.Errorf("Expected participant %s to have a tag number", p.UserID)
					}
				}
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				startTime := time.Now().Add(time.Hour)
				location := roundtypes.Location("Nonexistent Course")
				payload := createRoundStartedPayload(nonExistentRoundID, "Nonexistent Round", &startTime, &location)

				publishAndExpectRoundStartError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoRoundStartMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO ROUND STARTED TESTS
func createRoundStartedPayload(roundID sharedtypes.RoundID, title string, startTime *time.Time, location *roundtypes.Location) roundevents.RoundStartedPayload {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		st := sharedtypes.StartTime(*startTime)
		sharedStartTime = &st
	}

	return roundevents.RoundStartedPayload{
		RoundID:   roundID,
		Title:     roundtypes.Title(title),
		Location:  location,
		StartTime: sharedStartTime,
		ChannelID: "test-channel-id",
	}
}

// Publishing functions - UNIQUE TO ROUND STARTED TESTS
func publishRoundStartedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.RoundStartedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundStarted, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoRoundStartMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundStarted, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	successMsgs := getRoundStartSuccessFromHandlerMessages(capture)
	errorMsgs := getRoundStartErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for invalid JSON, got %d", len(successMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO ROUND STARTED TESTS
func waitForRoundStartSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.DiscordRoundStarted, count, defaultTimeout)
}

func waitForRoundStartErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND STARTED TESTS
func getRoundStartSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.DiscordRoundStarted)
}

func getRoundStartErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundError)
}

// Validation functions - UNIQUE TO ROUND STARTED TESTS
func validateRoundStartSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.DiscordRoundStartPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.DiscordRoundStartPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse discord round started message: %v", err)
	}

	// Validate that required fields are set
	if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	if result.Title == "" {
		t.Error("Expected Title to be set")
	}

	if result.EventMessageID == "" {
		t.Error("Expected EventMessageID to be set")
	}

	if result.Participants == nil {
		t.Error("Expected Participants to be set (even if empty)")
	}

	t.Logf("Round started successfully: %s ('%s'), participants: %d",
		result.RoundID, result.Title, len(result.Participants))

	return result
}

func validateRoundStartErrorFromHandler(t *testing.T, msg *message.Message) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse round start error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	// Log what we got for debugging
	t.Logf("Round start failed with error: %s", result.Error)
}

// Test expectation functions - UNIQUE TO ROUND STARTED TESTS
func publishAndExpectRoundStartSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundStartedPayload) *roundevents.DiscordRoundStartPayload {
	publishRoundStartedMessage(t, deps, &payload)

	if !waitForRoundStartSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected discord round started message from %s", roundevents.DiscordRoundStarted)
	}

	msgs := getRoundStartSuccessFromHandlerMessages(capture)
	result := validateRoundStartSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectRoundStartError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.RoundStartedPayload) {
	publishRoundStartedMessage(t, deps, &payload)

	if !waitForRoundStartErrorFromHandler(capture, 1) {
		t.Fatalf("Expected round start error message from %s", roundevents.RoundError)
	}

	msgs := getRoundStartErrorFromHandlerMessages(capture)
	validateRoundStartErrorFromHandler(t, msgs[0])
}
