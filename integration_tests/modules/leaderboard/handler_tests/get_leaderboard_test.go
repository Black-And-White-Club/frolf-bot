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

// Helper for creating and publishing a get leaderboard request message
func createGetLeaderboardRequestMessage(t *testing.T) (*message.Message, error) {
	t.Helper() // Mark this as a helper function
	payload := leaderboardevents.GetLeaderboardRequestedPayloadV1{GuildID: sharedtypes.GuildID("test_guild")}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg, nil
}

// TestHandleGetLeaderboardRequested runs integration tests for the get leaderboard handler
func TestHandleGetLeaderboardRequested(t *testing.T) {
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
			name:  "Success - Get Leaderboard With Data",
			users: generator.GenerateUsers(3),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 2},
					{UserID: sharedtypes.DiscordID(users[2].UserID), TagNumber: 3},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, err := createGetLeaderboardRequestMessage(t)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.GetLeaderboardRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := leaderboardevents.GetLeaderboardResponseV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.GetLeaderboardResponsePayloadV1](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				// Use the helper to compare leaderboard data maps
				expectedMap := testutils.ExtractLeaderboardDataMap(initialLeaderboard.LeaderboardData)
				actualMap := testutils.ExtractLeaderboardDataMap(responsePayload.Leaderboard)
				testutils.ValidateLeaderboardData(t, expectedMap, actualMap)

				// Validate correlation ID
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// Check for error messages
				unexpectedTopic := leaderboardevents.GetLeaderboardFailedV1
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.GetLeaderboardResponseV1},
			expectHandlerError:     false,
		},
		{
			name: "Success - Get Empty Leaderboard",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				// Create an empty leaderboard
				initialData := leaderboardtypes.LeaderboardData{}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, err := createGetLeaderboardRequestMessage(t)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.GetLeaderboardRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := leaderboardevents.GetLeaderboardResponseV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				responsePayload, err := testutils.ParsePayload[leaderboardevents.GetLeaderboardResponsePayloadV1](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				// Validate that we got an empty data set back
				if len(responsePayload.Leaderboard) != 0 {
					t.Errorf("Expected empty leaderboard data, got %d entries", len(responsePayload.Leaderboard))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.GetLeaderboardResponseV1},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Invalid Message Payload",
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

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.GetLeaderboardRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for unexpected success messages
				unexpectedSuccessTopic := leaderboardevents.GetLeaderboardResponseV1
				if len(receivedMsgs[unexpectedSuccessTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedSuccessTopic, len(receivedMsgs[unexpectedSuccessTopic]))
				}

				// Check for unexpected error messages
				unexpectedFailureTopic := leaderboardevents.GetLeaderboardFailedV1
				if len(receivedMsgs[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(receivedMsgs[unexpectedFailureTopic]))
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
		{
			name:  "Failure - No Active Leaderboard",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, false, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, err := createGetLeaderboardRequestMessage(t)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.GetLeaderboardRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// No success messages should be sent
				unexpectedSuccessTopic := leaderboardevents.GetLeaderboardResponseV1
				if len(receivedMsgs[unexpectedSuccessTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedSuccessTopic, len(receivedMsgs[unexpectedSuccessTopic]))
				}

				// No explicit error messages should be sent either in this case
				unexpectedFailureTopic := leaderboardevents.GetLeaderboardFailedV1
				if len(receivedMsgs[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(receivedMsgs[unexpectedFailureTopic]))
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
