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

func TestHandleScoreUpdateValidated(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Update Participant Score (Single Participant)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				// Create a round with one participant
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create score update validated payload
				score := sharedtypes.Score(-1) // 1 under par
				payload := createScoreUpdateValidatedPayload(roundID, data2.UserID, &score)

				result := publishAndExpectScoreUpdateSuccess2(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Participant != data2.UserID {
					t.Errorf("Expected Participant %s, got %s", data2.UserID, result.Participant)
				}
				if result.Score != score {
					t.Errorf("Expected Score %d, got %d", score, result.Score)
				}
				if len(result.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Participants))
				}
				if result.Participants[0].UserID != data2.UserID {
					t.Errorf("Expected participant %s, got %s", data2.UserID, result.Participants[0].UserID)
				}
				if result.Participants[0].Score == nil || *result.Participants[0].Score != score {
					t.Errorf("Expected participant score %d, got %v", score, result.Participants[0].Score)
				}
			},
		},
		{
			name: "Success - Update Participant Score (Multiple Participants)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				// Create a round with multiple participants
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: scorePtr(2)}, // Already has a score
					{UserID: data4.UserID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Update user2's score
				score := sharedtypes.Score(3) // 3 over par
				payload := createScoreUpdateValidatedPayload(roundID, data2.UserID, &score)

				result := publishAndExpectScoreUpdateSuccess2(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Participant != data2.UserID {
					t.Errorf("Expected Participant %s, got %s", data2.UserID, result.Participant)
				}
				if result.Score != score {
					t.Errorf("Expected Score %d, got %d", score, result.Score)
				}
				if len(result.Participants) != 3 {
					t.Errorf("Expected 3 participants, got %d", len(result.Participants))
				}

				// Find and validate user2's updated score
				found := false
				for _, p := range result.Participants {
					if p.UserID == data2.UserID {
						found = true
						if p.Score == nil || *p.Score != score {
							t.Errorf("Expected user2 score %d, got %v", score, p.Score)
						}
						break
					}
				}
				if !found {
					t.Error("Updated participant not found in result")
				}
			},
		},
		{
			name: "Success - Update Score for Participant with Existing Score",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				// Create a round with a participant who already has a score
				existingScore := sharedtypes.Score(1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &existingScore},
				})

				// Update to a new score
				newScore := sharedtypes.Score(-2) // Changed from +1 to -2
				payload := createScoreUpdateValidatedPayload(roundID, data2.UserID, &newScore)

				result := publishAndExpectScoreUpdateSuccess2(t, deps, deps.MessageCapture, payload)

				// Validate the score was updated
				if result.Score != newScore {
					t.Errorf("Expected updated Score %d, got %d", newScore, result.Score)
				}
				if len(result.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Participants))
				}
				if result.Participants[0].Score == nil || *result.Participants[0].Score != newScore {
					t.Errorf("Expected participant updated score %d, got %v", newScore, result.Participants[0].Score)
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
				payload := createScoreUpdateValidatedPayload(nonExistentRoundID, data.UserID, &score)

				publishAndExpectScoreUpdateError2(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Participant Not Found in Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				// Create a round without the participant we're trying to update
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Try to update score for a user not in the round
				score := sharedtypes.Score(1)
				payload := createScoreUpdateValidatedPayload(roundID, data3.UserID, &score) // data3.UserID not in round

				publishAndExpectScoreUpdateError2(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoScoreValidatedMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func createScoreUpdateValidatedPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score *sharedtypes.Score) roundevents.ScoreUpdateValidatedPayloadV1 {
	return roundevents.ScoreUpdateValidatedPayloadV1{
		GuildID: "test-guild",
		ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
			GuildID:     "test-guild",
			RoundID:     roundID,
			Participant: participant,
			Score:       score,
		},
	}
}

// Publishing functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func publishScoreUpdateValidatedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ScoreUpdateValidatedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateValidatedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScoreValidatedMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateValidatedV1, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	successMsgs := getScoreValidatedSuccessFromHandlerMessages(capture)
	errorMsgs := getScoreValidatedErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for invalid JSON, got %d", len(successMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func waitForScoreValidatedSuccessFromHandlerMatchingRound(t *testing.T, capture *testutils.MessageCapture, roundID sharedtypes.RoundID) *roundevents.ParticipantScoreUpdatedPayloadV1 {
	timeout := time.After(defaultTimeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil
		case <-ticker.C:
			msgs := capture.GetMessages(roundevents.RoundParticipantScoreUpdatedV1)
			for _, msg := range msgs {
				payload, err := testutils.ParsePayload[roundevents.ParticipantScoreUpdatedPayloadV1](msg)
				if err != nil {
					continue
				}
				if payload.RoundID == roundID {
					return payload
				}
			}
		}
	}
}

func waitForScoreValidatedErrorFromHandlerMatchingRound(t *testing.T, capture *testutils.MessageCapture, roundID sharedtypes.RoundID) *roundevents.RoundScoreUpdateErrorPayloadV1 {
	timeout := time.After(defaultTimeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil
		case <-ticker.C:
			msgs := capture.GetMessages(roundevents.RoundScoreUpdateErrorV1)
			for _, msg := range msgs {
				payload, err := testutils.ParsePayload[roundevents.RoundScoreUpdateErrorPayloadV1](msg)
				if err != nil {
					continue
				}
				if payload.ScoreUpdateRequest != nil && payload.ScoreUpdateRequest.RoundID == roundID {
					return payload
				}
			}
		}
	}
}

// Message retrieval functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func getScoreValidatedSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantScoreUpdatedV1)
}

func getScoreValidatedErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoreUpdateErrorV1)
}

// Validation functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func validateScoreValidatedSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.ParticipantScoreUpdatedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantScoreUpdatedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant score updated message: %v", err)
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

	t.Logf("Participant score updated successfully for round: %s, participant: %s, score: %s par, participants: %d",
		result.RoundID, result.Participant, scoreText, len(result.Participants))

	return result
}

func validateScoreValidatedErrorFromHandler(t *testing.T, msg *message.Message) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundScoreUpdateErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse score update error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	if result.ScoreUpdateRequest == nil {
		t.Error("Expected ScoreUpdateRequest to be populated")
	}

	// Log what we got for debugging
	t.Logf("Score update validation failed with error: %s", result.Error)
}

// Test expectation functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func publishAndExpectScoreUpdateSuccess2(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateValidatedPayloadV1) *roundevents.ParticipantScoreUpdatedPayloadV1 {
	publishScoreUpdateValidatedMessage(t, deps, &payload)

	result := waitForScoreValidatedSuccessFromHandlerMatchingRound(t, capture, payload.ScoreUpdateRequestPayload.RoundID)
	if result == nil {
		t.Fatalf("Timed out waiting for participant score updated message for round %s", payload.ScoreUpdateRequestPayload.RoundID)
	}

	// Log what we got for debugging
	scoreText := "even"
	if result.Score > 0 {
		scoreText = fmt.Sprintf("+%d over", result.Score)
	} else if result.Score < 0 {
		scoreText = fmt.Sprintf("%d under", result.Score)
	}

	t.Logf("Participant score updated successfully for round: %s, participant: %s, score: %s par, participants: %d",
		result.RoundID, result.Participant, scoreText, len(result.Participants))

	return result
}

func publishAndExpectScoreUpdateError2(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateValidatedPayloadV1) {
	publishScoreUpdateValidatedMessage(t, deps, &payload)

	result := waitForScoreValidatedErrorFromHandlerMatchingRound(t, capture, payload.ScoreUpdateRequestPayload.RoundID)
	if result == nil {
		t.Fatalf("Timed out waiting for score update error message for round %s", payload.ScoreUpdateRequestPayload.RoundID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// Helper utility functions
func scorePtr(score int) *sharedtypes.Score {
	s := sharedtypes.Score(score)
	return &s
}
