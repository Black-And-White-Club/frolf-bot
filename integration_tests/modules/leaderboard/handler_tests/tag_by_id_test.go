package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

const testGuildID = sharedtypes.GuildID("test_guild")

// Helper function to create a tag lookup request message
func createTagLookupRequestMessage(t *testing.T, userID sharedtypes.DiscordID) (*message.Message, error) {
	t.Helper()
	payload := sharedevents.DiscordTagLookupRequestedPayloadV1{
		ScopedGuildID:     sharedevents.ScopedGuildID{GuildID: testGuildID},
		RequestingUserID: userID,
		UserID:           userID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg, nil
}

// Helper to validate the leaderboard state in database
func assertLeaderboardState(t *testing.T, deps LeaderboardHandlerTestDeps, expectedLeaderboard *leaderboarddb.Leaderboard, expectedCount int, expectedActive bool) {
	t.Helper()
	testutils.AssertLeaderboardState(t, deps.DB, expectedLeaderboard, expectedCount, expectedActive)
}

func TestHandleGetTagByUserIDRequest(t *testing.T) {
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
			name:  "Success - Tag found for user on active leaderboard",
			users: generator.GenerateUsers(3),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 42},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 1},
					{UserID: sharedtypes.DiscordID(users[2].UserID), TagNumber: 23},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				userID := sharedtypes.DiscordID(users[0].UserID)
				msg, err := createTagLookupRequestMessage(t, userID)
				if err != nil {
					t.Fatalf("Failed to create tag lookup request message: %v", err)
				}
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := sharedevents.DiscordTagLookupSucceededV1
				unexpectedTopic := sharedevents.DiscordTagLookupNotFoundV1
				unexpectedFailTopic := sharedevents.DiscordTagLookupFailedV1

				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
				if len(receivedMsgs[unexpectedFailTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailTopic, len(receivedMsgs[unexpectedFailTopic]))
				}

				receivedMsg := msgs[0]
				var successPayload sharedevents.DiscordTagLookupResultPayloadV1
				if err := json.Unmarshal(receivedMsg.Payload, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal DiscordTagLookupResultPayload: %v", err)
				}

				var requestPayload sharedevents.DiscordTagLookupRequestedPayloadV1
				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
					t.Fatalf("Failed to unmarshal incoming request payload: %v", err)
				}

				if successPayload.UserID != requestPayload.UserID {
					t.Errorf("UserID mismatch: expected %q, got %q", requestPayload.UserID, successPayload.UserID)
				}
				if !successPayload.Found {
					t.Error("Expected Found to be true in success payload")
				}
				if successPayload.TagNumber == nil {
					t.Error("Expected TagNumber to be non-nil in success payload")
				}

				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				assertLeaderboardState(t, deps, initialLeaderboard, 1, true)
			},
			expectedOutgoingTopics: []string{sharedevents.DiscordTagLookupSucceededV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name:  "Success - User not found on active leaderboard",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 2},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				// Generate a user NOT on the leaderboard
				notFoundUser := generator.GenerateUsers(1)[0]
				userID := sharedtypes.DiscordID(notFoundUser.UserID)
				msg, err := createTagLookupRequestMessage(t, userID)
				if err != nil {
					t.Fatalf("Failed to create tag lookup request message: %v", err)
				}
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := sharedevents.DiscordTagLookupNotFoundV1
				unexpectedTopic := sharedevents.DiscordTagLookupSucceededV1
				unexpectedFailTopic := sharedevents.DiscordTagLookupFailedV1

				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
				if len(receivedMsgs[unexpectedFailTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailTopic, len(receivedMsgs[unexpectedFailTopic]))
				}

				receivedMsg := msgs[0]
				var notFoundPayload sharedevents.DiscordTagLookupResultPayloadV1
				if err := json.Unmarshal(receivedMsg.Payload, &notFoundPayload); err != nil {
					t.Fatalf("Failed to unmarshal DiscordTagLookupResultPayload: %v", err)
				}

				var requestPayload sharedevents.DiscordTagLookupRequestedPayloadV1
				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
					t.Fatalf("Failed to unmarshal incoming request payload: %v", err)
				}

				if notFoundPayload.UserID != requestPayload.UserID {
					t.Errorf("UserID mismatch: expected %q, got %q", requestPayload.UserID, notFoundPayload.UserID)
				}
				if notFoundPayload.Found {
					t.Error("Expected Found to be false in not-found payload")
				}
				if notFoundPayload.TagNumber != nil {
					t.Errorf("Expected nil TagNumber in not-found payload, got %v", notFoundPayload.TagNumber)
				}

				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				assertLeaderboardState(t, deps, initialLeaderboard, 1, true)
			},
			expectedOutgoingTopics: []string{sharedevents.DiscordTagLookupNotFoundV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - No active leaderboard exists",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, false, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				userID := sharedtypes.DiscordID(users[0].UserID)
				msg, err := createTagLookupRequestMessage(t, userID)
				if err != nil {
					t.Fatalf("Failed to create tag lookup request message: %v", err)
				}
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				expectedTopic := sharedevents.DiscordTagLookupFailedV1
				unexpectedTopic1 := sharedevents.DiscordTagLookupSucceededV1
				unexpectedTopic2 := sharedevents.DiscordTagLookupNotFoundV1

				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				if len(receivedMsgs[unexpectedTopic1]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic1, len(receivedMsgs[unexpectedTopic1]))
				}
				if len(receivedMsgs[unexpectedTopic2]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic2, len(receivedMsgs[unexpectedTopic2]))
				}

				receivedFailedMsg := msgs[0]
				var failurePayload sharedevents.DiscordTagLookupFailedPayloadV1
				if err := json.Unmarshal(receivedFailedMsg.Payload, &failurePayload); err != nil {
					t.Fatalf("Failed to unmarshal DiscordTagLookupByUserIDFailedPayload: %v", err)
				}

				var requestPayload sharedevents.DiscordTagLookupRequestedPayloadV1
				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
					t.Fatalf("Failed to unmarshal incoming request payload: %v", err)
				}

				if failurePayload.UserID != requestPayload.UserID {
					t.Errorf("UserID mismatch: expected %q, got %q", requestPayload.UserID, failurePayload.UserID)
				}

				if receivedFailedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						receivedFailedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				assertLeaderboardState(t, deps, initialLeaderboard, 0, false)
			},
			expectedOutgoingTopics: []string{sharedevents.DiscordTagLookupFailedV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - Invalid incoming message payload (Unmarshal error)",
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
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				unexpectedTopic1 := sharedevents.DiscordTagLookupSucceededV1
				unexpectedTopic2 := sharedevents.DiscordTagLookupNotFoundV1
				unexpectedTopic3 := sharedevents.DiscordTagLookupFailedV1

				if len(receivedMsgs[unexpectedTopic1]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic1, len(receivedMsgs[unexpectedTopic1]))
				}
				if len(receivedMsgs[unexpectedTopic2]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic2, len(receivedMsgs[unexpectedTopic2]))
				}
				if len(receivedMsgs[unexpectedTopic3]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic3, len(receivedMsgs[unexpectedTopic3]))
				}

				assertLeaderboardState(t, deps, initialLeaderboard, 1, true)
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
			timeout:                2 * time.Second,
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
