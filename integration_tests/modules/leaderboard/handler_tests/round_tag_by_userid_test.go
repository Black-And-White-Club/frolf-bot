package leaderboardhandler_integration_tests

// import (
// 	"context"
// 	"encoding/json"
// 	"testing"
// 	"time"

// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
// 	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
// 	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
// 	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
// 	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
// 	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
// 	"github.com/ThreeDotsLabs/watermill/message"
// 	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
// 	"github.com/google/uuid"
// )

// // Helper to publish tag lookup request
// func publishTagLookupRequest(t *testing.T, deps LeaderboardHandlerTestDeps, payload sharedevents.RoundTagLookupRequestPayload) *message.Message {
// 	t.Helper()
// 	b, err := json.Marshal(payload)
// 	if err != nil {
// 		t.Fatalf("Failed to marshal request payload: %v", err)
// 	}
// 	msg := message.NewMessage(uuid.New().String(), b)
// 	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.RoundGetTagByUserIDRequest, msg); err != nil {
// 		t.Fatalf("Failed to publish tag lookup request: %v", err)
// 	}
// 	return msg
// }

// // Helper to validate tag lookup result
// func validateTagLookupResponse(t *testing.T, requestPayload *sharedevents.RoundTagLookupRequestPayload, responseMsg *message.Message, expectedTag sharedtypes.TagNumber) {
// 	t.Helper()
// 	var response sharedevents.RoundTagLookupResultPayload
// 	if err := json.Unmarshal(responseMsg.Payload, &response); err != nil {
// 		t.Fatalf("Failed to unmarshal result payload: %v", err)
// 	}

// 	if response.UserID != requestPayload.UserID {
// 		t.Errorf("UserID mismatch: expected %s, got %s", requestPayload.UserID, response.UserID)
// 	}
// 	if response.RoundID != requestPayload.RoundID {
// 		t.Errorf("RoundID mismatch: expected %s, got %s", requestPayload.RoundID, response.RoundID)
// 	}
// 	if !response.Found {
// 		t.Error("Expected Found=true, got false")
// 	}
// 	if response.TagNumber == nil || *response.TagNumber != expectedTag {
// 		t.Errorf("Expected TagNumber %d, got %v", expectedTag, response.TagNumber)
// 	}
// 	if response.OriginalResponse != requestPayload.Response {
// 		t.Errorf("OriginalResponse mismatch: expected %q, got %q", requestPayload.Response, response.OriginalResponse)
// 	}
// }

// func TestHandleRoundGetTagRequest(t *testing.T) {
// 	var testUserID sharedtypes.DiscordID
// 	testCases := []struct {
// 		name                   string
// 		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard
// 		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message
// 		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard)
// 		expectedOutgoingTopics []string
// 		expectHandlerError     bool
// 	}{
// 		{
// 			name: "Success - Tag found for user on active leaderboard",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				users := generator.GenerateUsers(1)
// 				user := users[0]
// 				userID := sharedtypes.DiscordID(user.UserID)
// 				// Save userID for use in publishMsgFn
// 				testUserID = userID
// 				data := leaderboardtypes.LeaderboardData{
// 					{UserID: userID, TagNumber: tagPtr(99)},
// 				}
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, data)
// 				if err != nil {
// 					t.Fatalf("Failed to insert leaderboard: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				payload := sharedevents.RoundTagLookupRequestPayload{
// 					UserID:     testUserID, // Use the same userID as in setupFn
// 					RoundID:    sharedtypes.RoundID(uuid.New()),
// 					Response:   roundtypes.ResponseAccept,
// 					JoinedLate: boolPtr(false),
// 				}
// 				return publishTagLookupRequest(t, deps, payload)
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				expectedTopic := sharedevents.RoundTagLookupFound
// 				msgs := received[expectedTopic]
// 				if len(msgs) != 1 {
// 					t.Fatalf("Expected 1 message on %q, got %d", expectedTopic, len(msgs))
// 				}
// 				var req sharedevents.RoundTagLookupRequestPayload
// 				if err := json.Unmarshal(incoming.Payload, &req); err != nil {
// 					t.Fatalf("Failed to parse request: %v", err)
// 				}
// 				validateTagLookupResponse(t, &req, msgs[0], 99)
// 			},
// 			expectedOutgoingTopics: []string{sharedevents.RoundTagLookupFound},
// 			expectHandlerError:     false,
// 		},
// 		{
// 			name: "Success - Tag not found (user not on leaderboard)",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				existing := generator.GenerateUsers(1)[0]
// 				data := leaderboardtypes.LeaderboardData{
// 					{UserID: sharedtypes.DiscordID(existing.UserID), TagNumber: tagPtr(77)},
// 				}
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, data)
// 				if err != nil {
// 					t.Fatalf("InsertLeaderboard error: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				nonMember := generator.GenerateUsers(1)[0]
// 				payload := sharedevents.RoundTagLookupRequestPayload{
// 					UserID:     sharedtypes.DiscordID(nonMember.UserID),
// 					RoundID:    sharedtypes.RoundID(uuid.New()),
// 					Response:   roundtypes.ResponseTentative,
// 					JoinedLate: boolPtr(true),
// 				}
// 				return publishTagLookupRequest(t, deps, payload)
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				msgs := received[sharedevents.RoundTagLookupNotFound]
// 				if len(msgs) != 1 {
// 					t.Fatalf("Expected 1 RoundTagLookupNotFound message, got %d", len(msgs))
// 				}
// 			},
// 			expectedOutgoingTopics: []string{sharedevents.RoundTagLookupNotFound},
// 			expectHandlerError:     false,
// 		},
// 		{
// 			name: "Failure - No active leaderboard present",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				data := leaderboardtypes.LeaderboardData{
// 					{UserID: "ghost_user", TagNumber: tagPtr(13)},
// 				}
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, data)
// 				if err != nil {
// 					t.Fatalf("InsertLeaderboard failed: %v", err)
// 				}
// 				// manually deactivate leaderboard
// 				if _, err := deps.DB.NewUpdate().Model(lb).Set("is_active = ?", false).WherePK().Exec(context.Background()); err != nil {
// 					t.Fatalf("Failed to deactivate leaderboard: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				u := generator.GenerateUsers(1)[0]
// 				payload := sharedevents.RoundTagLookupRequestPayload{
// 					UserID:     sharedtypes.DiscordID(u.UserID),
// 					RoundID:    sharedtypes.RoundID(uuid.New()),
// 					Response:   roundtypes.ResponseAccept,
// 					JoinedLate: boolPtr(false),
// 				}
// 				return publishTagLookupRequest(t, deps, payload)
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				msgs := received[leaderboardevents.GetTagNumberFailed]
// 				if len(msgs) != 1 {
// 					t.Fatalf("Expected 1 GetTagNumberFailed message, got %d", len(msgs))
// 				}
// 			},
// 			expectedOutgoingTopics: []string{leaderboardevents.GetTagNumberFailed},
// 			expectHandlerError:     false,
// 		},
// 		{
// 			name: "Failure - Invalid payload (unmarshal error)",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, leaderboardtypes.LeaderboardData{})
// 				if err != nil {
// 					t.Fatalf("InsertLeaderboard error: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				msg := message.NewMessage(uuid.New().String(), []byte("invalid JSON"))
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.RoundGetTagByUserIDRequest, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				for _, topic := range []string{
// 					sharedevents.RoundTagLookupFound,
// 					sharedevents.RoundTagLookupNotFound,
// 					leaderboardevents.GetTagNumberFailed,
// 				} {
// 					if len(received[topic]) > 0 {
// 						t.Errorf("Expected no messages on topic %q, but got %d", topic, len(received[topic]))
// 					}
// 				}
// 			},
// 			expectedOutgoingTopics: []string{},
// 			expectHandlerError:     true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			deps := SetupTestLeaderboardHandler(t)
// 			generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

// 			genericCase := testutils.TestCase{
// 				Name: tc.name,
// 				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
// 					return tc.setupFn(t, deps, generator)
// 				},
// 				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
// 					return tc.publishMsgFn(t, deps, generator)
// 				},
// 				ExpectedTopics: tc.expectedOutgoingTopics,
// 				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
// 					tc.validateFn(t, deps, incoming, received, initialState.(*leaderboarddb.Leaderboard))
// 				},
// 				ExpectError: tc.expectHandlerError,
// 			}
// 			testutils.RunTest(t, genericCase, deps.TestEnvironment)
// 		})
// 	}
// }
