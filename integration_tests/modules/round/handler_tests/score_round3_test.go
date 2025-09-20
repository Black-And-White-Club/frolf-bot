package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestHandleParticipantScoreUpdated(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	users := generator.GenerateUsers(5)
	user1ID := sharedtypes.DiscordID(users[0].UserID)
	user2ID := sharedtypes.DiscordID(users[1].UserID)
	user3ID := sharedtypes.DiscordID(users[2].UserID)
	user4ID := sharedtypes.DiscordID(users[3].UserID)
	user5ID := sharedtypes.DiscordID(users[4].UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - All Scores Submitted (Single Participant)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with one participant who has a score
				score1 := sharedtypes.Score(-1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
				})

				// Create participant score updated payload
				payload := createParticipantScoreUpdatedPayload(roundID, user2ID, score1, []roundtypes.Participant{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
				})

				result := publishAndExpectAllScoresSubmitted(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if len(result.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Participants))
				}
				if result.Participants[0].Score == nil || *result.Participants[0].Score != score1 {
					t.Errorf("Expected participant score %d, got %v", score1, result.Participants[0].Score)
				}
			},
		},
		{
			name: "Success - All Scores Submitted (Multiple Participants)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with multiple participants, all with scores
				score1 := sharedtypes.Score(-2)
				score2 := sharedtypes.Score(1)
				score3 := sharedtypes.Score(0)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: user4ID, Response: roundtypes.ResponseAccept, Score: &score3},
				})

				// Simulate that user3 just updated their score
				payload := createParticipantScoreUpdatedPayload(roundID, user3ID, score2, []roundtypes.Participant{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: user4ID, Response: roundtypes.ResponseAccept, Score: &score3},
				})

				result := publishAndExpectAllScoresSubmitted(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if len(result.Participants) != 3 {
					t.Errorf("Expected 3 participants, got %d", len(result.Participants))
				}

				// Verify all participants have scores
				for _, p := range result.Participants {
					if p.Score == nil {
						t.Errorf("Expected all participants to have scores, but %s has nil", p.UserID)
					}
				}
			},
		},
		{
			name: "Success - Not All Scores Submitted (Single Participant Without Score)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with mixed participants - some with scores, some without
				score1 := sharedtypes.Score(2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: nil}, // No score yet
				})

				// Simulate that user2 just updated their score
				payload := createParticipantScoreUpdatedPayload(roundID, user2ID, score1, []roundtypes.Participant{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				result := publishAndExpectNotAllScoresSubmitted(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Participant != user2ID {
					t.Errorf("Expected Participant %s, got %s", user2ID, result.Participant)
				}
				if result.Score != score1 {
					t.Errorf("Expected Score %d, got %d", score1, result.Score)
				}
				if len(result.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(result.Participants))
				}
			},
		},
		{
			name: "Success - Not All Scores Submitted (Multiple Missing Scores)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round where multiple participants are missing scores
				score1 := sharedtypes.Score(-1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: nil},    // No score
					{UserID: user4ID, Response: roundtypes.ResponseAccept, Score: nil},    // No score
					{UserID: user5ID, Response: roundtypes.ResponseTentative, Score: nil}, // No score
				})

				// Simulate that user2 just updated their score
				payload := createParticipantScoreUpdatedPayload(roundID, user2ID, score1, []roundtypes.Participant{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: user4ID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: user5ID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				result := publishAndExpectNotAllScoresSubmitted(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if len(result.Participants) != 4 {
					t.Errorf("Expected 4 participants, got %d", len(result.Participants))
				}

				// Count participants without scores
				missingScores := 0
				for _, p := range result.Participants {
					if p.Score == nil {
						missingScores++
					}
				}
				if missingScores != 3 {
					t.Errorf("Expected 3 participants without scores, got %d", missingScores)
				}
			},
		},
		{
			name: "Success - Last Score Submitted Triggers All Scores Submitted",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round where this update completes all scores
				score1 := sharedtypes.Score(1)
				score2 := sharedtypes.Score(-1)
				score3 := sharedtypes.Score(0) // This will be the final score
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: user4ID, Response: roundtypes.ResponseAccept, Score: &score3}, // Just updated
				})

				// Simulate that user4 just submitted the final score
				payload := createParticipantScoreUpdatedPayload(roundID, user4ID, score3, []roundtypes.Participant{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: user4ID, Response: roundtypes.ResponseAccept, Score: &score3},
				})

				result := publishAndExpectAllScoresSubmitted(t, deps, deps.MessageCapture, payload)

				// Validate that we got all scores submitted
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if len(result.Participants) != 3 {
					t.Errorf("Expected 3 participants, got %d", len(result.Participants))
				}

				// Verify all participants have scores
				for _, p := range result.Participants {
					if p.Score == nil {
						t.Errorf("Expected all participants to have scores after final submission")
					}
				}
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				score := sharedtypes.Score(0)
				payload := createParticipantScoreUpdatedPayload(nonExistentRoundID, user1ID, score, []roundtypes.Participant{})

				publishAndExpectScoreCheckError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoScoreCheckMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func createParticipantScoreUpdatedPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score sharedtypes.Score, participants []roundtypes.Participant) roundevents.ParticipantScoreUpdatedPayload {
	return roundevents.ParticipantScoreUpdatedPayload{
		RoundID:        roundID,
		Participant:    participant,
		Score:          score,
		EventMessageID: "test-event-message-id",
		Participants:   participants,
		GuildID:        "test-guild",
	}
}

// Publishing functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func publishParticipantScoreUpdatedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantScoreUpdatedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	t.Logf("Publishing ParticipantScoreUpdated message for round %s", payload.RoundID)
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantScoreUpdated, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScoreCheckMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantScoreUpdated, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	allScoresMsgs := getAllScoresSubmittedFromHandlerMessages(capture)
	notAllScoresMsgs := getNotAllScoresSubmittedFromHandlerMessages(capture)
	errorMsgs := getScoreCheckErrorFromHandlerMessages(capture)

	if len(allScoresMsgs) > 0 {
		t.Errorf("Expected no all scores submitted messages for invalid JSON, got %d", len(allScoresMsgs))
	}

	if len(notAllScoresMsgs) > 0 {
		t.Errorf("Expected no not all scores submitted messages for invalid JSON, got %d", len(notAllScoresMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func waitForAllScoresSubmittedFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundAllScoresSubmitted, count, defaultTimeout)
}

func waitForNotAllScoresSubmittedFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundNotAllScoresSubmitted, count, defaultTimeout)
}

func waitForScoreCheckErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundError, count, 5*time.Second) // Increase timeout
}

// Message retrieval functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func getAllScoresSubmittedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundAllScoresSubmitted)
}

func getNotAllScoresSubmittedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundNotAllScoresSubmitted)
}

func getScoreCheckErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundError)
}

// Validation functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func validateAllScoresSubmittedFromHandler(t *testing.T, msg *message.Message) *roundevents.AllScoresSubmittedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.AllScoresSubmittedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse all scores submitted message: %v", err)
	}

	// Validate that required fields are set
	if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	if result.EventMessageID == "" {
		t.Error("Expected EventMessageID to be set")
	}

	if result.Participants == nil {
		t.Error("Expected Participants to be set")
	}

	// Verify all participants have scores
	for _, p := range result.Participants {
		if p.Score == nil {
			t.Errorf("Expected all participants to have scores in AllScoresSubmittedPayload, but %s has nil", p.UserID)
		}
	}

	t.Logf("All scores submitted for round: %s, participants: %d", result.RoundID, len(result.Participants))

	return result
}

func validateNotAllScoresSubmittedFromHandler(t *testing.T, msg *message.Message) *roundevents.NotAllScoresSubmittedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.NotAllScoresSubmittedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse not all scores submitted message: %v", err)
	}

	// Validate that required fields are set
	if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	if result.Participant == "" {
		t.Error("Expected Participant to be set")
	}

	if result.EventMessageID == "" {
		t.Error("Expected EventMessageID to be set")
	}

	if result.Participants == nil {
		t.Error("Expected Participants to be set")
	}

	// Log what we got for debugging
	scoreText := "even"
	if result.Score > 0 {
		scoreText = fmt.Sprintf("+%d over", result.Score)
	} else if result.Score < 0 {
		scoreText = fmt.Sprintf("%d under", result.Score)
	}

	t.Logf("Not all scores submitted for round: %s, updated participant: %s, score: %s par, total participants: %d",
		result.RoundID, result.Participant, scoreText, len(result.Participants))

	return result
}

func validateScoreCheckErrorFromHandler(t *testing.T, msg *message.Message) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse score check error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	// Log what we got for debugging
	t.Logf("Score check failed with error: %s", result.Error)
}

// Test expectation functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func publishAndExpectAllScoresSubmitted(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantScoreUpdatedPayload) *roundevents.AllScoresSubmittedPayload {
	publishParticipantScoreUpdatedMessage(t, deps, &payload)

	if !waitForAllScoresSubmittedFromHandler(capture, 1) {
		t.Fatalf("Expected all scores submitted message from %s", roundevents.RoundAllScoresSubmitted)
	}

	msgs := getAllScoresSubmittedFromHandlerMessages(capture)
	result := validateAllScoresSubmittedFromHandler(t, msgs[0])

	return result
}

func publishAndExpectNotAllScoresSubmitted(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantScoreUpdatedPayload) *roundevents.NotAllScoresSubmittedPayload {
	publishParticipantScoreUpdatedMessage(t, deps, &payload)

	if !waitForNotAllScoresSubmittedFromHandler(capture, 1) {
		t.Fatalf("Expected not all scores submitted message from %s", roundevents.RoundNotAllScoresSubmitted)
	}

	msgs := getNotAllScoresSubmittedFromHandlerMessages(capture)
	result := validateNotAllScoresSubmittedFromHandler(t, msgs[0])

	return result
}

func publishAndExpectScoreCheckError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantScoreUpdatedPayload) {
	t.Logf("Publishing ParticipantScoreUpdated for non-existent round %s", payload.RoundID)
	publishParticipantScoreUpdatedMessage(t, deps, &payload)

	// Add extra wait time for debugging
	time.Sleep(500 * time.Millisecond)

	t.Logf("Waiting for error message on topic %s", roundevents.RoundError)
	if !waitForScoreCheckErrorFromHandler(capture, 1) {
		// Debug: Check what messages we did receive
		allScoresMsgs := getAllScoresSubmittedFromHandlerMessages(capture)
		notAllScoresMsgs := getNotAllScoresSubmittedFromHandlerMessages(capture)
		errorMsgs := getScoreCheckErrorFromHandlerMessages(capture)

		t.Logf("DEBUG: Messages received - AllScores: %d, NotAllScores: %d, Errors: %d",
			len(allScoresMsgs), len(notAllScoresMsgs), len(errorMsgs))

		t.Fatalf("Expected score check error message from %s", roundevents.RoundError)
	}

	msgs := getScoreCheckErrorFromHandlerMessages(capture)
	validateScoreCheckErrorFromHandler(t, msgs[0])
}
