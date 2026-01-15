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

// createTagSwapRequestMessage helper matches the handler's expected payload
func createTagSwapRequestMessage(t *testing.T, requestorID, targetID sharedtypes.DiscordID) (*message.Message, error) {
	t.Helper()
	payload := leaderboardevents.TagSwapRequestedPayloadV1{
		GuildID:     sharedtypes.GuildID("test_guild"),
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

func TestHandleTagSwapRequested(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

	testCases := []struct {
		name                   string
		users                  []testutils.User
		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard
		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message
		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initial *leaderboarddb.Leaderboard)
		expectedOutgoingTopics []string
	}{
		{
			name:  "Success - Immediate Swap (Both users on leaderboard)",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 20},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, _ := createTagSwapRequestMessage(t, sharedtypes.DiscordID(users[0].UserID), sharedtypes.DiscordID(users[1].UserID))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				// We expect TagSwapProcessedV1 as returned by Step 4 or 5 of the handler
				msgs := receivedMsgs[leaderboardevents.TagSwapProcessedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected message on topic %s", leaderboardevents.TagSwapProcessedV1)
				}

				var res leaderboardevents.TagSwapProcessedPayloadV1
				json.Unmarshal(msgs[0].Payload, &res)

				if res.RequestorID == "" || res.TargetID == "" {
					t.Errorf("Response payload missing IDs: %+v", res)
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapProcessedV1},
		},
		{
			name:  "Failure - Target user has no tag",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				// Only Requestor has a tag
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 10},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg, _ := createTagSwapRequestMessage(t, sharedtypes.DiscordID(users[0].UserID), sharedtypes.DiscordID(users[1].UserID))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagSwapRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := receivedMsgs[leaderboardevents.TagSwapFailedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected message on topic %s", leaderboardevents.TagSwapFailedV1)
				}

				var res leaderboardevents.TagSwapFailedPayloadV1
				json.Unmarshal(msgs[0].Payload, &res)

				// Based on Step 1 of your handler, the Reason should be "target_user_has_no_tag"
				if res.Reason != "target_user_has_no_tag" {
					t.Errorf("Expected reason 'target_user_has_no_tag', got %s", res.Reason)
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapFailedV1},
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
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, incoming, received, initialState.(*leaderboarddb.Leaderboard))
				},
			}, deps.TestEnvironment)
		})
	}
}
