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
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		UserID:        userID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg, nil
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
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 42},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, _ := createTagLookupRequestMessage(t, sharedtypes.DiscordID(users[0].UserID))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				msgs := receivedMsgs[sharedevents.LeaderboardTagLookupSucceededV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on Success topic, got %d", len(msgs))
				}

				var res sharedevents.DiscordTagLookupResultPayloadV1
				json.Unmarshal(msgs[0].Payload, &res)

				if !res.Found || res.TagNumber == nil || *res.TagNumber != 42 {
					t.Errorf("Result mismatch: Found=%v, Tag=%v", res.Found, res.TagNumber)
				}
			},
			expectedOutgoingTopics: []string{sharedevents.LeaderboardTagLookupSucceededV1},
		},
		{
			name:  "Success - User not found on active leaderboard",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				// Setup leaderboard with someone else
				entries := []leaderboardtypes.LeaderboardEntry{{UserID: "other_user", TagNumber: 1}}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, _ := createTagLookupRequestMessage(t, sharedtypes.DiscordID(users[0].UserID))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				msgs := receivedMsgs[sharedevents.LeaderboardTagLookupNotFoundV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on NotFound topic, got %d", len(msgs))
				}

				var res sharedevents.DiscordTagLookupResultPayloadV1
				json.Unmarshal(msgs[0].Payload, &res)
				if res.Found {
					t.Error("Expected Found=false")
				}
			},
			expectedOutgoingTopics: []string{sharedevents.LeaderboardTagLookupNotFoundV1},
		},
		{
			name:  "Success - Treated as NotFound when no active leaderboard exists",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, nil, false, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, _ := createTagLookupRequestMessage(t, sharedtypes.DiscordID(users[0].UserID))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Per handler code: found := err == nil. If leaderboard doesn't exist, err is returned, found is false.
				msgs := receivedMsgs[sharedevents.LeaderboardTagLookupNotFoundV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on NotFound topic due to missing leaderboard, got %d", len(msgs))
				}
			},
			expectedOutgoingTopics: []string{sharedevents.LeaderboardTagLookupNotFoundV1},
		},
		{
			name:  "Failure - Invalid JSON",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid"))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.DiscordTagLookupRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				if len(receivedMsgs) > 0 {
					t.Error("Expected no outgoing messages on unmarshal failure")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestLeaderboardHandler(t)
			testutils.RunTest(t, testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, tc.users)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, tc.users)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					var initial *leaderboarddb.Leaderboard
					if initialState != nil {
						initial = initialState.(*leaderboarddb.Leaderboard)
					}
					tc.validateFn(t, deps, incomingMsg, receivedMsgs, initial)
				},
				ExpectError: tc.expectHandlerError,
			}, deps.TestEnvironment)
		})
	}
}
