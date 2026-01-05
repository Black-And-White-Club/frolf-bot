package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
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

// Helper to publish tag lookup request
func publishTagLookupRequest(t *testing.T, deps LeaderboardHandlerTestDeps, payload sharedevents.RoundTagLookupRequestedPayloadV1) *message.Message {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal request payload: %v", err)
	}
	msg := message.NewMessage(uuid.New().String(), b)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.RoundGetTagByUserIDRequestedV1, msg); err != nil {
		t.Fatalf("Failed to publish tag lookup request: %v", err)
	}
	return msg
}

// Helper to validate tag lookup result
func validateTagLookupResponse(t *testing.T, requestPayload *sharedevents.RoundTagLookupRequestedPayloadV1, responseMsg *message.Message, expectedTag sharedtypes.TagNumber) {
	t.Helper()
	var response sharedevents.RoundTagLookupResultPayloadV1
	if err := json.Unmarshal(responseMsg.Payload, &response); err != nil {
		t.Fatalf("Failed to unmarshal result payload: %v", err)
	}

	if response.UserID != requestPayload.UserID {
		t.Errorf("UserID mismatch: expected %s, got %s", requestPayload.UserID, response.UserID)
	}
	if response.RoundID != requestPayload.RoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", requestPayload.RoundID, response.RoundID)
	}
	if !response.Found {
		t.Error("Expected Found=true, got false")
	}
	if response.TagNumber == nil || *response.TagNumber != expectedTag {
		t.Errorf("Expected TagNumber %d, got %v", expectedTag, response.TagNumber)
	}
	if response.OriginalResponse != requestPayload.Response {
		t.Errorf("OriginalResponse mismatch: expected %q, got %q", requestPayload.Response, response.OriginalResponse)
	}
}

func TestHandleRoundGetTagRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	var testUserID sharedtypes.DiscordID
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
				user := users[0]
				userID := sharedtypes.DiscordID(user.UserID)
				// Save userID for use in publishMsgFn
				testUserID = userID
				data := leaderboardtypes.LeaderboardData{
					{UserID: userID, TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				payload := sharedevents.RoundTagLookupRequestedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: sharedtypes.GuildID("test_guild")},
					UserID:     testUserID, // Use the same userID as in setupFn
					RoundID:    sharedtypes.RoundID(uuid.New()),
					Response:   roundtypes.ResponseAccept,
					JoinedLate: boolPtr(false),
				}
				return publishTagLookupRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				expectedTopic := sharedevents.RoundTagLookupFoundV1
				msgs := received[expectedTopic]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on %q, got %d", expectedTopic, len(msgs))
				}
				var req sharedevents.RoundTagLookupRequestedPayloadV1
				if err := json.Unmarshal(incoming.Payload, &req); err != nil {
					t.Fatalf("Failed to parse request: %v", err)
				}
				validateTagLookupResponse(t, &req, msgs[0], 99)
			},
			expectedOutgoingTopics: []string{sharedevents.RoundTagLookupFoundV1},
			expectHandlerError:     false,
		},
		{
			name:  "Success - Tag not found (user not on leaderboard)",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				existing := generator.GenerateUsers(1)[0]
				data := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(existing.UserID), TagNumber: 77},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				nonMember := generator.GenerateUsers(1)[0]
				payload := sharedevents.RoundTagLookupRequestedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: sharedtypes.GuildID("test_guild")},
					UserID:     sharedtypes.DiscordID(nonMember.UserID),
					RoundID:    sharedtypes.RoundID(uuid.New()),
					Response:   roundtypes.ResponseTentative,
					JoinedLate: boolPtr(true),
				}
				return publishTagLookupRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[sharedevents.RoundTagLookupNotFoundV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 RoundTagLookupNotFound message, got %d", len(msgs))
				}
			},
			expectedOutgoingTopics: []string{sharedevents.RoundTagLookupNotFoundV1},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - No active leaderboard present",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{
					{UserID: "ghost_user", TagNumber: 13},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, false, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				u := generator.GenerateUsers(1)[0]
				payload := sharedevents.RoundTagLookupRequestedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: sharedtypes.GuildID("test_guild")},
					UserID:     sharedtypes.DiscordID(u.UserID),
					RoundID:    sharedtypes.RoundID(uuid.New()),
					Response:   roundtypes.ResponseAccept,
					JoinedLate: boolPtr(false),
				}
				return publishTagLookupRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[leaderboardevents.GetTagNumberFailedV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetTagNumberFailed message, got %d", len(msgs))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.GetTagNumberFailedV1},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Invalid payload (unmarshal error)",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid JSON"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.RoundGetTagByUserIDRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				for _, topic := range []string{
					sharedevents.RoundTagLookupFoundV1,
					sharedevents.RoundTagLookupNotFoundV1,
					leaderboardevents.GetTagNumberFailedV1,
				} {
					if len(received[topic]) > 0 {
						t.Errorf("Expected no messages on topic %q, but got %d", topic, len(received[topic]))
					}
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
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, incoming, received, initialState.(*leaderboarddb.Leaderboard))
				},
				ExpectError: tc.expectHandlerError,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
