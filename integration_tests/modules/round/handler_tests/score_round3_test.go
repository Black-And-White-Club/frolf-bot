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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - All Scores Submitted (Single Participant)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				// Create a round with one participant who has a score
				score1 := sharedtypes.Score(-1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
				})

				// Create participant score updated payload
				payload := createParticipantScoreUpdatedPayload(roundID, data2.UserID, score1, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
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
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				// Create a round with multiple participants, all with scores
				score1 := sharedtypes.Score(-2)
				score2 := sharedtypes.Score(1)
				score3 := sharedtypes.Score(0)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
				})

				// Simulate that user3 just updated their score
				payload := createParticipantScoreUpdatedPayload(roundID, data3.UserID, score2, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
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
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				// Create a round with mixed participants - some with scores, some without
				score1 := sharedtypes.Score(2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil}, // No score yet
				})

				// Simulate that user2 just updated their score
				payload := createParticipantScoreUpdatedPayload(roundID, data2.UserID, score1, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				result := publishAndExpectNotAllScoresSubmitted(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Participant != data2.UserID {
					t.Errorf("Expected Participant %s, got %s", data2.UserID, result.Participant)
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
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				data5 := NewTestData()
				// Create a round where multiple participants are missing scores
				score1 := sharedtypes.Score(-1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},    // No score
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: nil},    // No score
					{UserID: data5.UserID, Response: roundtypes.ResponseTentative, Score: nil}, // No score
				})

				// Simulate that user2 just updated their score
				payload := createParticipantScoreUpdatedPayload(roundID, data2.UserID, score1, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data5.UserID, Response: roundtypes.ResponseTentative, Score: nil},
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
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				// Create a round where this update completes all scores
				score1 := sharedtypes.Score(1)
				score2 := sharedtypes.Score(-1)
				score3 := sharedtypes.Score(0) // This will be the final score
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3}, // Just updated
				})

				// Simulate that user4 just submitted the final score
				payload := createParticipantScoreUpdatedPayload(roundID, data4.UserID, score3, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
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
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				score := sharedtypes.Score(0)
				payload := createParticipantScoreUpdatedPayload(nonExistentRoundID, data.UserID, score, []roundtypes.Participant{})

				publishAndExpectScoreCheckError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoScoreCheckMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func createParticipantScoreUpdatedPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score sharedtypes.Score, participants []roundtypes.Participant) roundevents.ParticipantScoreUpdatedPayloadV1 {
	return roundevents.ParticipantScoreUpdatedPayloadV1{
		GuildID:        "test-guild",
		RoundID:        roundID,
		Participant:    participant,
		Score:          score,
		ChannelID:      "test_channel_123",
		EventMessageID: "test-event-message-id",
		Participants:   participants,
		Config:         nil,
	}
}

// Publishing functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func publishParticipantScoreUpdatedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantScoreUpdatedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	t.Logf("Publishing ParticipantScoreUpdated message for round %s", payload.RoundID)
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScoreCheckMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantScoreUpdatedV1, invalidMsg); err != nil {
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
	return capture.WaitForMessages(roundevents.RoundAllScoresSubmittedV1, count, defaultTimeout)
}

func waitForNotAllScoresSubmittedFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScoresPartiallySubmittedV1, count, defaultTimeout)
}

func waitForScoreCheckErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundErrorV1, count, 700*time.Millisecond)
}

// Message retrieval functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func getAllScoresSubmittedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundAllScoresSubmittedV1)
}

func getNotAllScoresSubmittedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoresPartiallySubmittedV1)
}

func getScoreCheckErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundErrorV1)
}

// Validation functions - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func validateAllScoresSubmittedFromHandler(t *testing.T, msg *message.Message) *roundevents.AllScoresSubmittedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.AllScoresSubmittedPayloadV1](msg)
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

func validateNotAllScoresSubmittedFromHandler(t *testing.T, msg *message.Message) *roundevents.ScoresPartiallySubmittedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ScoresPartiallySubmittedPayloadV1](msg)
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

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayloadV1](msg)
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
func publishAndExpectAllScoresSubmitted(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantScoreUpdatedPayloadV1) *roundevents.AllScoresSubmittedPayloadV1 {
	publishParticipantScoreUpdatedMessage(t, deps, &payload)

	if !waitForAllScoresSubmittedFromHandler(capture, 1) {
		t.Fatalf("Expected all scores submitted message from %s", roundevents.RoundAllScoresSubmittedV1)
	}

	msgs := getAllScoresSubmittedFromHandlerMessages(capture)
	result := validateAllScoresSubmittedFromHandler(t, msgs[0])

	return result
}

func publishAndExpectNotAllScoresSubmitted(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantScoreUpdatedPayloadV1) *roundevents.ScoresPartiallySubmittedPayloadV1 {
	publishParticipantScoreUpdatedMessage(t, deps, &payload)

	// Wait and filter by round ID
	deadline := time.Now().Add(defaultTimeout)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getNotAllScoresSubmittedFromHandlerMessages(capture)
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.ScoresPartiallySubmittedPayloadV1](msg)
			if err == nil && parsed.RoundID == payload.RoundID {
				foundMsg = msg
				break
			}
		}
		if foundMsg != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if foundMsg == nil {
		t.Fatalf("Expected not all scores submitted message from %s for round %s", roundevents.RoundScoresPartiallySubmittedV1, payload.RoundID)
	}

	result := validateNotAllScoresSubmittedFromHandler(t, foundMsg)
	return result
}

func publishAndExpectScoreCheckError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantScoreUpdatedPayloadV1) {
	t.Logf("Publishing ParticipantScoreUpdated for non-existent round %s", payload.RoundID)
	publishParticipantScoreUpdatedMessage(t, deps, &payload)

	// Add extra wait time for debugging
	time.Sleep(500 * time.Millisecond)

	t.Logf("Waiting for error message on topic %s", roundevents.RoundErrorV1)
	if !waitForScoreCheckErrorFromHandler(capture, 1) {
		// Debug: Check what messages we did receive
		allScoresMsgs := getAllScoresSubmittedFromHandlerMessages(capture)
		notAllScoresMsgs := getNotAllScoresSubmittedFromHandlerMessages(capture)
		errorMsgs := getScoreCheckErrorFromHandlerMessages(capture)

		t.Logf("DEBUG: Messages received - AllScores: %d, NotAllScores: %d, Errors: %d",
			len(allScoresMsgs), len(notAllScoresMsgs), len(errorMsgs))

		t.Fatalf("Expected score check error message from %s", roundevents.RoundErrorV1)
	}

	msgs := getScoreCheckErrorFromHandlerMessages(capture)
	validateScoreCheckErrorFromHandler(t, msgs[0])
}
