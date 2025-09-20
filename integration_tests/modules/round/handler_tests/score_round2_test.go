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
			name: "Success - Update Participant Score (Single Participant)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with one participant
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Create score update validated payload
				score := sharedtypes.Score(-1) // 1 under par
				payload := createScoreUpdateValidatedPayload(roundID, user2ID, &score)

				result := publishAndExpectScoreUpdateSuccess2(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Participant != user2ID {
					t.Errorf("Expected Participant %s, got %s", user2ID, result.Participant)
				}
				if result.Score != score {
					t.Errorf("Expected Score %d, got %d", score, result.Score)
				}
				if len(result.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(result.Participants))
				}
				if result.Participants[0].UserID != user2ID {
					t.Errorf("Expected participant %s, got %s", user2ID, result.Participants[0].UserID)
				}
				if result.Participants[0].Score == nil || *result.Participants[0].Score != score {
					t.Errorf("Expected participant score %d, got %v", score, result.Participants[0].Score)
				}
			},
		},
		{
			name: "Success - Update Participant Score (Multiple Participants)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round with multiple participants
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: user3ID, Response: roundtypes.ResponseAccept, Score: scorePtr(2)}, // Already has a score
					{UserID: user4ID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Update user2's score
				score := sharedtypes.Score(3) // 3 over par
				payload := createScoreUpdateValidatedPayload(roundID, user2ID, &score)

				result := publishAndExpectScoreUpdateSuccess2(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.RoundID)
				}
				if result.Participant != user2ID {
					t.Errorf("Expected Participant %s, got %s", user2ID, result.Participant)
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
					if p.UserID == user2ID {
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
				// Create a round with a participant who already has a score
				existingScore := sharedtypes.Score(1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: &existingScore},
				})

				// Update to a new score
				newScore := sharedtypes.Score(-2) // Changed from +1 to -2
				payload := createScoreUpdateValidatedPayload(roundID, user2ID, &newScore)

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
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				score := sharedtypes.Score(0)
				payload := createScoreUpdateValidatedPayload(nonExistentRoundID, user1ID, &score)

				publishAndExpectScoreUpdateError2(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Participant Not Found in Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round without the participant we're trying to update
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, user1ID, []testutils.ParticipantData{
					{UserID: user2ID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Try to update score for a user not in the round
				score := sharedtypes.Score(1)
				payload := createScoreUpdateValidatedPayload(roundID, user3ID, &score) // user3ID not in round

				publishAndExpectScoreUpdateError2(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoScoreValidatedMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func createScoreUpdateValidatedPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score *sharedtypes.Score) roundevents.ScoreUpdateValidatedPayload {
	return roundevents.ScoreUpdateValidatedPayload{
		ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayload{
			RoundID:     roundID,
			Participant: participant,
			Score:       score,
			GuildID:     "test-guild",
		},
	}
}

// Publishing functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func publishScoreUpdateValidatedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ScoreUpdateValidatedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateValidated, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScoreValidatedMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateValidated, invalidMsg); err != nil {
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
func waitForScoreValidatedSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantScoreUpdated, count, defaultTimeout)
}

func waitForScoreValidatedErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScoreUpdateError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func getScoreValidatedSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantScoreUpdated)
}

func getScoreValidatedErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoreUpdateError)
}

// Validation functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func validateScoreValidatedSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.ParticipantScoreUpdatedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantScoreUpdatedPayload](msg)
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

	result, err := testutils.ParsePayload[roundevents.RoundScoreUpdateErrorPayload](msg)
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
func publishAndExpectScoreUpdateSuccess2(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateValidatedPayload) *roundevents.ParticipantScoreUpdatedPayload {
	publishScoreUpdateValidatedMessage(t, deps, &payload)

	if !waitForScoreValidatedSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected participant score updated message from %s", roundevents.RoundParticipantScoreUpdated)
	}

	msgs := getScoreValidatedSuccessFromHandlerMessages(capture)
	result := validateScoreValidatedSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectScoreUpdateError2(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateValidatedPayload) {
	publishScoreUpdateValidatedMessage(t, deps, &payload)

	if !waitForScoreValidatedErrorFromHandler(capture, 1) {
		t.Fatalf("Expected score update error message from %s", roundevents.RoundScoreUpdateError)
	}

	msgs := getScoreValidatedErrorFromHandlerMessages(capture)
	validateScoreValidatedErrorFromHandler(t, msgs[0])
}

// Helper utility functions
func scorePtr(score int) *sharedtypes.Score {
	s := sharedtypes.Score(score)
	return &s
}
