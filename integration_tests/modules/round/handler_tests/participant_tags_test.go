package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleScheduledRoundTagUpdate(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Tag Update for Single Round with Multiple Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data1 := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				// Create rounds with participants using existing helpers
				oldTag1 := sharedtypes.TagNumber(10)
				oldTag2 := sharedtypes.TagNumber(20)

				// Use the helper that explicitly creates rounds in "upcoming" state
				roundID1 := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data1.UserID, roundtypes.ResponseAccept, &oldTag1)
				roundID2 := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data2.UserID, roundtypes.ResponseAccept, &oldTag2)
				roundID3 := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data3.UserID, roundtypes.ResponseTentative, nil)

				// Debug: Log what we created
				t.Logf("Created rounds: %s, %s, %s", roundID1, roundID2, roundID3)
				t.Logf("User1: %s (tag %d), User2: %s (tag %d), User3: %s (no tag)",
					data1.UserID, oldTag1, data2.UserID, oldTag2, data3.UserID)

				// Create tag update payload
				newTag1 := sharedtypes.TagNumber(42)
				newTag2 := sharedtypes.TagNumber(99)
				changedTags := map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					data1.UserID: &newTag1,
					data2.UserID: &newTag2,
					// data3 not in the map - should not be updated
				}

				t.Logf("Changing tags for: User1 %s -> %d, User2 %s -> %d",
					data1.UserID, newTag1, data2.UserID, newTag2)

				payload := createScheduledRoundTagUpdatePayload(changedTags)

				result := publishAndExpectTagUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate specific results using the new payload structure
				if len(result.UpdatedRounds) != 2 {
					t.Errorf("Expected 2 rounds to be updated, got %d. UpdatedRounds: %v", len(result.UpdatedRounds), result.UpdatedRounds)
				}

				if result.Summary.ParticipantsUpdated != 2 {
					t.Errorf("Expected 2 participants to be updated, got %d", result.Summary.ParticipantsUpdated)
				}

				// Check that the correct rounds were updated
				expectedRounds := map[sharedtypes.RoundID]bool{
					roundID1: false,
					roundID2: false,
				}
				for _, roundInfo := range result.UpdatedRounds {
					if _, exists := expectedRounds[roundInfo.RoundID]; exists {
						expectedRounds[roundInfo.RoundID] = true
					} else {
						t.Errorf("Unexpected round ID in result: %s", roundInfo.RoundID)
					}
				}
				for roundID, found := range expectedRounds {
					if !found {
						t.Errorf("Expected round ID %s not found in result", roundID)
					}
				}
			},
		},
		{
			name: "Success - Tag Update for Multiple Rounds with Same Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create multiple upcoming rounds with the same participant
				oldTag := sharedtypes.TagNumber(50)
				helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, roundtypes.ResponseAccept, &oldTag)
				helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, roundtypes.ResponseAccept, &oldTag)

				// Create tag update payload
				newTag := sharedtypes.TagNumber(123)
				changedTags := map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					data.UserID: &newTag,
				}
				payload := createScheduledRoundTagUpdatePayload(changedTags)

				publishAndExpectTagUpdateSuccess(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Empty Tag Update (No Upcoming Rounds with Matching Participants)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round with a participant that won't match the tag update
				oldTag := sharedtypes.TagNumber(75)
				helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, roundtypes.ResponseAccept, &oldTag)

				// Create tag update payload for a different user
				newTag := sharedtypes.TagNumber(456)
				changedTags := map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					sharedtypes.DiscordID("nonexistent-user"): &newTag,
				}
				payload := createScheduledRoundTagUpdatePayload(changedTags)

				// When no rounds need updating, the handler returns empty message array, so no message is published
				publishAndExpectNoTagUpdateMessages(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Tag Update Only Affects Upcoming Rounds",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create upcoming round with participant that should be updated
				oldTag := sharedtypes.TagNumber(100)
				helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, roundtypes.ResponseAccept, &oldTag)

				// Create tag update payload
				newTag := sharedtypes.TagNumber(789)
				changedTags := map[sharedtypes.DiscordID]*sharedtypes.TagNumber{
					data.UserID: &newTag,
				}
				payload := createScheduledRoundTagUpdatePayload(changedTags)

				publishAndExpectTagUpdateSuccess(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Invalid JSON - Scheduled Round Tag Update Handler",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoTagUpdateMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO SCHEDULED ROUND TAG UPDATE TESTS
func createScheduledRoundTagUpdatePayload(changedTags map[sharedtypes.DiscordID]*sharedtypes.TagNumber) roundevents.ScheduledRoundTagUpdatePayloadV1 {
	return roundevents.ScheduledRoundTagUpdatePayloadV1{
		GuildID:     "test-guild",
		ChangedTags: changedTags,
	}
}

// Publishing functions - UNIQUE TO SCHEDULED ROUND TAG UPDATE TESTS
func publishScheduledRoundTagUpdateMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ScheduledRoundTagUpdatePayloadV1) *message.Message {
	t.Helper()

	// The handler expects a map with a "changed_tags" key, not the direct struct
	mapPayload := map[string]interface{}{
		"guild_id":     payload.GuildID,
		"changed_tags": payload.ChangedTags,
	}

	payloadBytes, err := json.Marshal(mapPayload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.TagUpdateForScheduledRoundsV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// Wait functions - UNIQUE TO SCHEDULED ROUND TAG UPDATE TESTS
func waitForTagUpdateSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.TagsUpdatedForScheduledRoundsV1, count, defaultTimeout)
}

func waitForTagUpdateErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundUpdateErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO SCHEDULED ROUND TAG UPDATE TESTS
func getTagUpdateSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.TagsUpdatedForScheduledRoundsV1)
}

func getTagUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundUpdateErrorV1)
}

// Validation functions - UNIQUE TO SCHEDULED ROUND TAG UPDATE TESTS
func validateTagUpdateSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.TagsUpdatedForScheduledRoundsPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.TagsUpdatedForScheduledRoundsPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse tag update success message: %v", err)
	}

	// Validate the structure - these should never be nil, even if empty
	if result.UpdatedRounds == nil {
		t.Error("Expected UpdatedRounds to be initialized (not nil)")
	}

	// Log what we got for debugging
	t.Logf("Tag update success: %d rounds updated, %d total processed, %d participants updated",
		len(result.UpdatedRounds), result.Summary.TotalRoundsProcessed, result.Summary.ParticipantsUpdated)

	return result
}

func validateTagUpdateErrorFromHandler(t *testing.T, msg *message.Message) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundUpdateErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse tag update error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// Test expectation functions - UNIQUE TO SCHEDULED ROUND TAG UPDATE TESTS
func publishAndExpectTagUpdateSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScheduledRoundTagUpdatePayloadV1) *roundevents.TagsUpdatedForScheduledRoundsPayloadV1 {
	publishScheduledRoundTagUpdateMessage(t, deps, &payload)

	if !waitForTagUpdateSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected tag update success message from %s", roundevents.TagsUpdatedForScheduledRoundsV1)
	}

	msgs := getTagUpdateSuccessFromHandlerMessages(capture)
	result := validateTagUpdateSuccessFromHandler(t, msgs[0])

	// Return the result so test cases can do additional validation
	return result
}

func publishAndExpectNoTagUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScheduledRoundTagUpdatePayloadV1) {
	publishScheduledRoundTagUpdateMessage(t, deps, &payload)

	// Wait a bit to ensure no messages are published
	time.Sleep(200 * time.Millisecond)

	successMsgs := getTagUpdateSuccessFromHandlerMessages(capture)
	errorMsgs := getTagUpdateErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 || len(errorMsgs) > 0 {
		t.Errorf("Expected no messages when no rounds need updating, got %d success, %d error msgs",
			len(successMsgs), len(errorMsgs))
	}
}

func publishInvalidJSONAndExpectNoTagUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.TagUpdateForScheduledRoundsV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	successMsgs := getTagUpdateSuccessFromHandlerMessages(capture)
	errorMsgs := getTagUpdateErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 || len(errorMsgs) > 0 {
		t.Errorf("Expected no messages for invalid JSON on %s, got %d success, %d error msgs",
			sharedevents.TagUpdateForScheduledRoundsV1, len(successMsgs), len(errorMsgs))
	}
}
