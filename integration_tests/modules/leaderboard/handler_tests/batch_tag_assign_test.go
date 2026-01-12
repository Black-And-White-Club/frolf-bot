package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// Helper for creating and publishing a batch tag assignment request message
func createBatchAssignmentRequestMessage(t *testing.T, guildID sharedtypes.GuildID, requestingUserID sharedtypes.DiscordID, assignments []sharedevents.TagAssignmentInfoV1) (*message.Message, error) {
	t.Helper() // Mark this as a helper function
	payload := sharedevents.BatchTagAssignmentRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{
			GuildID: guildID,
		},
		RequestingUserID: requestingUserID,
		BatchID:          uuid.New().String(),
		Assignments:      assignments,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg, nil
}

// Helper to validate basic success response properties
func validateSuccessResponse(t *testing.T, requestPayload *sharedevents.BatchTagAssignmentRequestedPayloadV1, responsePayload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1) {
	t.Helper() // Mark this as a helper function
	if responsePayload.RequestingUserID != requestPayload.RequestingUserID {
		t.Errorf("Success payload RequestingUserID mismatch: expected %q, got %q",
			requestPayload.RequestingUserID, responsePayload.RequestingUserID)
	}
	if responsePayload.BatchID != requestPayload.BatchID {
		t.Errorf("Success payload BatchID mismatch: expected %q, got %q",
			requestPayload.BatchID, responsePayload.BatchID)
	}
}

// TestHandleBatchTagAssignmentRequested runs integration tests for the batch tag assignment handler
func TestHandleBatchTagAssignmentRequested(t *testing.T) {
	var existingUserID sharedtypes.DiscordID // If you want parallel
	var newUserID sharedtypes.DiscordID      // move these inside t.Run block
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

	testCases := []struct {
		name                   string
		users                  []testutils.User
		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard
		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message
		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard)
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration
	}{
		{
			name:  "Success - Process Valid Batch Assignments",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 2},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				newUsers := generator.GenerateUsers(2)
				requestingUser := generator.GenerateUsers(1)[0]

				assignments := []sharedevents.TagAssignmentInfoV1{
					{UserID: sharedtypes.DiscordID(newUsers[0].UserID), TagNumber: 10},
					{UserID: sharedtypes.DiscordID(newUsers[1].UserID), TagNumber: 20},
				}

				msg, err := createBatchAssignmentRequestMessage(t, "test_guild", sharedtypes.DiscordID(requestingUser.UserID), assignments)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.LeaderboardBatchTagAssignmentRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := leaderboardevents.LeaderboardBatchTagAssignedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					// Multiple messages may be published as a result of related cross-module events.
					// Tests only require at least one result message — record a warning but do not fail.
					t.Logf("Warning: received %d messages on topic %q; using the first for validation", len(msgs), expectedTopic)
				}

				requestPayload, err := testutils.ParsePayload[sharedevents.BatchTagAssignmentRequestedPayloadV1](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.LeaderboardBatchTagAssignedPayloadV1](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				validateSuccessResponse(t, requestPayload, responsePayload)

				// The handler returns the complete leaderboard state after the batch operation
				// This includes: 2 initial users + 2 new assignments = 4 total entries
				expectedTotalCount := 4 // Complete leaderboard after updates
				if responsePayload.AssignmentCount != expectedTotalCount {
					t.Errorf("Success payload AssignmentCount mismatch: expected %d, got %d", expectedTotalCount, responsePayload.AssignmentCount)
				}

				// Validate correlation ID
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// Validate leaderboard state in database
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if len(leaderboards) != 2 {
					t.Fatalf("Expected 2 leaderboard records (old inactive, new active), got %d", len(leaderboards))
				}

				oldLeaderboard := leaderboards[0]
				newLeaderboard := leaderboards[1]

				if oldLeaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected old leaderboard ID %d, got %d", initialLeaderboard.ID, oldLeaderboard.ID)
				}
				if oldLeaderboard.IsActive {
					t.Error("Expected old leaderboard to be inactive")
				}

				if !newLeaderboard.IsActive {
					t.Error("Expected new leaderboard to be active")
				}

				// Instead of comparing expected merged data, directly examine the actual data
				// and verify it contains the expected entries
				actualData := testutils.ExtractLeaderboardDataMap(newLeaderboard.LeaderboardData)

				// Debug logging to see what we actually got
				t.Logf("Actual leaderboard data: %+v", actualData)

				// Verify all assignments were applied
				for _, assignment := range requestPayload.Assignments {
					tag, exists := actualData[assignment.UserID]
					if !exists {
						t.Errorf("User %s from assignments not found in leaderboard", assignment.UserID)
					} else if tag != assignment.TagNumber {
						t.Errorf("Tag mismatch for user %s: expected %d, got %d",
							assignment.UserID, assignment.TagNumber, tag)
					}
				}

				// Check that the initial data is preserved (unless overwritten)
				initialData := testutils.ExtractLeaderboardDataMap(initialLeaderboard.LeaderboardData)
				for userID, initialTag := range initialData {
					// Only check users that weren't in the new assignments
					var wasAssigned bool
					for _, assignment := range requestPayload.Assignments {
						if assignment.UserID == userID {
							wasAssigned = true
							break
						}
					}

					if !wasAssigned {
						tag, exists := actualData[userID]
						if !exists {
							t.Errorf("User %s from original leaderboard not found in new leaderboard", userID)
						} else if tag != initialTag {
							t.Errorf("Tag mismatch for original user %s: expected %d, got %d",
								userID, initialTag, tag)
						}
					}
				}

				// Check for error messages
				unexpectedTopic := leaderboardevents.LeaderboardBatchTagAssignmentFailedV1
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardBatchTagAssignedV1},
			expectHandlerError:     false,
		},
		{
			name:  "Success - Batch Assignment with Already Assigned Tag ",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				existingUserID = sharedtypes.DiscordID(users[0].UserID)
				newUserID = sharedtypes.DiscordID(users[1].UserID)
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: existingUserID, TagNumber: 10},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				assignments := []sharedevents.TagAssignmentInfoV1{
					{UserID: newUserID, TagNumber: 30},
					{UserID: existingUserID, TagNumber: 10}, // User came in with tag and leaving with same tag.
				}
				msg, err := createBatchAssignmentRequestMessage(t, "test_guild", existingUserID, assignments)
				if err != nil {
					t.Fatalf("%v", err)
				}
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.LeaderboardBatchTagAssignmentRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := leaderboardevents.LeaderboardBatchTagAssignedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					// Multiple messages are possible (e.g., related side-effects). Use the first message for assertions.
					t.Logf("Warning: received %d messages on topic %q; using the first for validation", len(msgs), expectedTopic)
				}
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("%v", err)
				}
				if len(leaderboards) != 2 {
					t.Fatalf("Expected 2 leaderboard records (old inactive, new active), got %d", len(leaderboards))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardBatchTagAssignedV1},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Invalid Message Payload (Unmarshal Error)",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid json payload"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.LeaderboardBatchTagAssignmentRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for unexpected messages
				unexpectedSuccessTopic := leaderboardevents.LeaderboardBatchTagAssignedV1
				if len(receivedMsgs[unexpectedSuccessTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedSuccessTopic, len(receivedMsgs[unexpectedSuccessTopic]))
				}
				unexpectedFailureTopic := leaderboardevents.LeaderboardBatchTagAssignmentFailedV1
				if len(receivedMsgs[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(receivedMsgs[unexpectedFailureTopic]))
				}

				// Validate leaderboard state in database
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if len(leaderboards) != 1 {
					t.Fatalf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
				}

				leaderboard := leaderboards[0]
				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
		{
			name: "Failure - Service Returns Error",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialLeaderboard := testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, true, sharedtypes.RoundID(uuid.New()))

				return initialLeaderboard
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				requestingUser := generator.GenerateUsers(1)[0]
				userX := generator.GenerateUsers(1)[0]

				assignments := []sharedevents.TagAssignmentInfoV1{
					{UserID: sharedtypes.DiscordID(userX.UserID), TagNumber: 9999},
				}

				msg, err := createBatchAssignmentRequestMessage(t, "test_guild", sharedtypes.DiscordID(requestingUser.UserID), assignments)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.LeaderboardBatchTagAssignmentRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for unexpected messages
				unexpectedSuccessTopic := leaderboardevents.LeaderboardBatchTagAssignedV1
				if len(receivedMsgs[unexpectedSuccessTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedSuccessTopic, len(receivedMsgs[unexpectedSuccessTopic]))
				}
				unexpectedFailureTopic := leaderboardevents.LeaderboardBatchTagAssignmentFailedV1
				if len(receivedMsgs[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(receivedMsgs[unexpectedFailureTopic]))
				}

				// Validate leaderboard state in database
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if len(leaderboards) < 1 {
					t.Fatalf("Expected at least 1 leaderboard record in DB, got %d", len(leaderboards))
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestLeaderboardHandler(t)
			users := tc.users

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, users)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, users)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, incomingMsg, receivedMsgs, initialState.(*leaderboarddb.Leaderboard))
				},
				ExpectError: tc.expectHandlerError,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
