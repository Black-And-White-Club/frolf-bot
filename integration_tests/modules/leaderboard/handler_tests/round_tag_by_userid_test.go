package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func publishTagLookupRequest(t *testing.T, deps LeaderboardHandlerTestDeps, payload sharedevents.RoundTagLookupRequestedPayloadV1) *message.Message {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal request payload: %v", err)
	}
	msg := message.NewMessage(uuid.New().String(), b)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.RoundTagLookupRequestedV1, msg); err != nil {
		t.Fatalf("Failed to publish tag lookup request: %v", err)
	}
	return msg
}

func validateTagLookupResponse(t *testing.T, requestPayload *sharedevents.RoundTagLookupRequestedPayloadV1, responseMsg *message.Message, expectedTag sharedtypes.TagNumber) {
	t.Helper()
	var response sharedevents.RoundTagLookupResultPayloadV1
	if err := json.Unmarshal(responseMsg.Payload, &response); err != nil {
		t.Fatalf("Failed to unmarshal result payload: %v", err)
	}

	if response.UserID != requestPayload.UserID {
		t.Errorf("UserID mismatch: expected %s, got %s", requestPayload.UserID, response.UserID)
	}
	if response.Found != true {
		t.Error("Expected Found=true")
	}
	if response.TagNumber == nil || *response.TagNumber != expectedTag {
		t.Errorf("Expected TagNumber %d, got %v", expectedTag, response.TagNumber)
	}
}

func TestHandleRoundGetTagRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

	testCases := []struct {
		name                   string
		users                  []testutils.User
		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard
		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message
		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard)
		expectedOutgoingTopics []string
		expectHandlerError     bool
	}{
		{
			name:  "Success - Tag found for user on active leaderboard",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99}}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				return publishTagLookupRequest(t, deps, sharedevents.RoundTagLookupRequestedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: "test_guild"},
					UserID:        sharedtypes.DiscordID(users[0].UserID),
					RoundID:       sharedtypes.RoundID(uuid.New()),
					Response:      roundtypes.ResponseAccept,
				})
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[sharedevents.RoundTagLookupFoundV1]
				if len(msgs) == 0 {
					t.Fatal("Expected RoundTagLookupFoundV1 message")
				}
				var req sharedevents.RoundTagLookupRequestedPayloadV1
				json.Unmarshal(incoming.Payload, &req)
				validateTagLookupResponse(t, &req, msgs[0], 99)
			},
			expectedOutgoingTopics: []string{sharedevents.RoundTagLookupFoundV1},
		},
		{
			name:  "Success - Tag not found (user not on leaderboard)",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{{UserID: "someone_else", TagNumber: 77}}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				return publishTagLookupRequest(t, deps, sharedevents.RoundTagLookupRequestedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: "test_guild"},
					UserID:        sharedtypes.DiscordID(users[0].UserID),
					RoundID:       sharedtypes.RoundID(uuid.New()),
				})
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				// Service now returns a failure result for missing tag lookups; handler maps that to RoundTagLookupNotFoundV1
				if len(received[sharedevents.RoundTagLookupNotFoundV1]) != 1 {
					t.Fatalf("Expected 1 RoundTagLookupNotFound message, got %d", len(received[sharedevents.RoundTagLookupNotFoundV1]))
				}
			},
			expectedOutgoingTopics: []string{sharedevents.RoundTagLookupNotFoundV1},
		},
		{
			name:  "Failure - No active leaderboard present",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, nil, false, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				return publishTagLookupRequest(t, deps, sharedevents.RoundTagLookupRequestedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: "test_guild"},
					UserID:        sharedtypes.DiscordID(users[0].UserID),
					RoundID:       sharedtypes.RoundID(uuid.New()),
				})
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				// If the handler returns (nil, err), the wrapper might not publish to a topic.
				// But based on your specific requirements, if you expect a fail topic, ensure it's mapped in the router.
				// For now, we verify that no "Found" or "NotFound" messages were sent erroneously.
				if len(received[sharedevents.RoundTagLookupFoundV1]) > 0 || len(received[sharedevents.RoundTagLookupNotFoundV1]) > 0 {
					t.Error("Should not have sent Found/NotFound on missing leaderboard")
				}
			},
			expectedOutgoingTopics: []string{}, // Change to empty if handler simply returns error
			expectHandlerError:     true,       // The router should see this as a handler error
		},
		{
			name:  "Failure - Invalid payload (unmarshal error)",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid"))
				testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.RoundTagLookupRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received) > 0 {
					t.Errorf("Expected no messages, got %d", len(received))
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
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					var initial *leaderboarddb.Leaderboard
					if initialState != nil {
						initial = initialState.(*leaderboarddb.Leaderboard)
					}
					tc.validateFn(t, deps, incoming, received, initial)
				},
				ExpectError: tc.expectHandlerError,
			}, deps.TestEnvironment)
		})
	}
}
