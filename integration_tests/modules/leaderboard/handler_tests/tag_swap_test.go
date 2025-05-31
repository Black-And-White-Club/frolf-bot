package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// Helper for creating and publishing a tag swap request message
func createTagSwapRequestMessage(t *testing.T, requestorID, targetID sharedtypes.DiscordID) (*message.Message, error) {
	t.Helper() // Mark this as a helper function
	payload := leaderboardevents.TagSwapRequestedPayload{
		RequestorID: requestorID,
		TargetID:    targetID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg, nil
}

// Helper to validate success response properties
func validateTagSwapSuccessResponse(t *testing.T, requestPayload *leaderboardevents.TagSwapRequestedPayload, responsePayload *leaderboardevents.TagSwapProcessedPayload) {
	t.Helper() // Mark this as a helper function
	if responsePayload.RequestorID != requestPayload.RequestorID {
		t.Errorf("Success payload RequestorID mismatch: expected %q, got %q",
			requestPayload.RequestorID, responsePayload.RequestorID)
	}
	if responsePayload.TargetID != requestPayload.TargetID {
		t.Errorf("Success payload TargetID mismatch: expected %q, got %q",
			requestPayload.TargetID, responsePayload.TargetID)
	}
}

// Helper to validate failure response properties
func validateTagSwapFailureResponse(t *testing.T, requestPayload *leaderboardevents.TagSwapRequestedPayload, responsePayload *leaderboardevents.TagSwapFailedPayload) {
	t.Helper() // Mark this as a helper function
	if responsePayload.RequestorID != requestPayload.RequestorID {
		t.Errorf("Failure payload RequestorID mismatch: expected %q, got %q",
			requestPayload.RequestorID, responsePayload.RequestorID)
	}
	if responsePayload.TargetID != requestPayload.TargetID {
		t.Errorf("Failure payload TargetID mismatch: expected %q, got %q",
			requestPayload.TargetID, responsePayload.TargetID)
	}
	if responsePayload.Reason == "" {
		t.Error("Failure payload should contain a reason")
	}
}

// TestHandleTagSwapRequested runs integration tests for the tag swap handler
func TestHandleTagSwapRequested(t *testing.T) {
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
			name:  "Success - Swap Tags Between Two Users",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 20},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				requestorID := sharedtypes.DiscordID(users[0].UserID)
				targetID := sharedtypes.DiscordID(users[1].UserID)

				msg, err := createTagSwapRequestMessage(t, requestorID, targetID)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for success message
				expectedTopic := leaderboardevents.TagSwapProcessed
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				// Parse and validate payloads
				requestPayload, err := testutils.ParsePayload[leaderboardevents.TagSwapRequestedPayload](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.TagSwapProcessedPayload](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				validateTagSwapSuccessResponse(t, requestPayload, responsePayload)

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

				// Extract data maps
				actualData := testutils.ExtractLeaderboardDataMap(newLeaderboard.LeaderboardData)
				initialData := testutils.ExtractLeaderboardDataMap(initialLeaderboard.LeaderboardData)

				// Debug logging
				t.Logf("Initial leaderboard data: %+v", initialData)
				t.Logf("Actual leaderboard data: %+v", actualData)

				// Verify the tags were actually swapped
				user1ID := sharedtypes.DiscordID(requestPayload.RequestorID)
				user2ID := sharedtypes.DiscordID(requestPayload.TargetID)

				user1InitialTag := initialData[user1ID]
				user2InitialTag := initialData[user2ID]

				user1FinalTag := actualData[user1ID]
				user2FinalTag := actualData[user2ID]

				if user1FinalTag != user2InitialTag {
					t.Errorf("Tag swap incorrect for requestor: expected %d, got %d",
						user2InitialTag, user1FinalTag)
				}

				if user2FinalTag != user1InitialTag {
					t.Errorf("Tag swap incorrect for target: expected %d, got %d",
						user1InitialTag, user2FinalTag)
				}

				// Check for error messages
				unexpectedTopic := leaderboardevents.TagSwapFailed
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapProcessed},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - One User Does Not Exist on Leaderboard",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				// Only add one user to the leaderboard
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				requestorID := sharedtypes.DiscordID(users[0].UserID)
				targetID := sharedtypes.DiscordID(users[1].UserID) // This user doesn't have a tag

				msg, err := createTagSwapRequestMessage(t, requestorID, targetID)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for failure message
				expectedTopic := leaderboardevents.TagSwapFailed
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				// Parse and validate payloads
				requestPayload, err := testutils.ParsePayload[leaderboardevents.TagSwapRequestedPayload](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.TagSwapFailedPayload](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				validateTagSwapFailureResponse(t, requestPayload, responsePayload)

				// Validate correlation ID
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// Validate leaderboard state in database (should be unchanged)
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if len(leaderboards) != 1 {
					t.Fatalf("Expected 1 leaderboard record (unchanged), got %d", len(leaderboards))
				}

				leaderboard := leaderboards[0]
				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}

				// Verify data is unchanged
				actualData := testutils.ExtractLeaderboardDataMap(leaderboard.LeaderboardData)
				initialData := testutils.ExtractLeaderboardDataMap(initialLeaderboard.LeaderboardData)

				// Verify the user's tag hasn't changed
				user1ID := sharedtypes.DiscordID(requestPayload.RequestorID)
				if actualData[user1ID] != initialData[user1ID] {
					t.Errorf("User tag should not have changed: expected %d, got %d",
						initialData[user1ID], actualData[user1ID])
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapFailed},
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

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for unexpected messages
				unexpectedSuccessTopic := leaderboardevents.TagSwapProcessed
				if len(receivedMsgs[unexpectedSuccessTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedSuccessTopic, len(receivedMsgs[unexpectedSuccessTopic]))
				}
				unexpectedFailureTopic := leaderboardevents.TagSwapFailed
				if len(receivedMsgs[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(receivedMsgs[unexpectedFailureTopic]))
				}

				// Validate leaderboard state in database (should be unchanged)
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
			name:  "Failure - No Active Leaderboard",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				// Create a leaderboard but set it to inactive
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 20},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, false, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				requestorID := sharedtypes.DiscordID(users[0].UserID)
				targetID := sharedtypes.DiscordID(users[1].UserID)

				msg, err := createTagSwapRequestMessage(t, requestorID, targetID)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for failure message
				expectedTopic := leaderboardevents.TagSwapFailed
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				// Parse and validate payloads
				requestPayload, err := testutils.ParsePayload[leaderboardevents.TagSwapRequestedPayload](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.TagSwapFailedPayload](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				validateTagSwapFailureResponse(t, requestPayload, responsePayload)

				// Verify the reason contains information about there being no active leaderboard
				if responsePayload.Reason != "no active leaderboard found" {
					t.Errorf("Expected specific failure reason, got: %s", responsePayload.Reason)
				}

				// Validate leaderboard state in database (should be unchanged)
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
				if leaderboard.IsActive != false {
					t.Error("Expected leaderboard to remain inactive")
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapFailed},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Same User Swap Request",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				// Same user for both requestor and target (should fail)
				sameUserID := sharedtypes.DiscordID(users[0].UserID)

				msg, err := createTagSwapRequestMessage(t, sameUserID, sameUserID)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// We expect the handler to process this rather than error
				// It should result in a failure message since swapping with self is not allowed

				// Check for failure message
				expectedTopic := leaderboardevents.TagSwapFailed
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				// Parse and validate payloads
				requestPayload, err := testutils.ParsePayload[leaderboardevents.TagSwapRequestedPayload](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.TagSwapFailedPayload](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				validateTagSwapFailureResponse(t, requestPayload, responsePayload)

				// Validate leaderboard state in database (should be unchanged)
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
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapFailed},
			expectHandlerError:     false,
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
