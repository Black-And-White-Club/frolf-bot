package leaderboardhandler_integration_tests

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"testing"
// 	"time"

// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
// 	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
// 	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
// 	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
// 	"github.com/ThreeDotsLabs/watermill/message"
// 	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
// 	"github.com/google/uuid"
// )

// // Helper: Parse "tag:user" string into LeaderboardEntry
// func parseTagUserStrings(tagUsers []string) leaderboardtypes.LeaderboardData {
// 	data := leaderboardtypes.LeaderboardData{}
// 	for _, entry := range tagUsers {
// 		parts := strings.Split(entry, ":")
// 		if len(parts) == 2 {
// 			tag, _ := strconv.Atoi(parts[0])
// 			data = append(data, leaderboardtypes.LeaderboardEntry{
// 				UserID:    sharedtypes.DiscordID(parts[1]),
// 				TagNumber: tagPtr(sharedtypes.TagNumber(tag)),
// 			})
// 		}
// 	}
// 	return data
// }

// // Helper: Validate success response for leaderboard update
// func validateLeaderboardUpdatedPayload(t *testing.T, req leaderboardevents.LeaderboardUpdateRequestedPayload, msg *message.Message) leaderboardevents.LeaderboardUpdatedPayload {
// 	t.Helper()
// 	var res leaderboardevents.LeaderboardUpdatedPayload
// 	if err := json.Unmarshal(msg.Payload, &res); err != nil {
// 		t.Fatalf("Failed to parse payload: %v", err)
// 	}

// 	if res.RoundID != req.RoundID {
// 		t.Errorf("RoundID mismatch: expected %s, got %s", req.RoundID, res.RoundID)
// 	}

// 	if msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != "" &&
// 		msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != msg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
// 		t.Errorf("Correlation ID mismatch")
// 	}
// 	return res
// }

// // Integration test for LeaderboardUpdateRequested handler
// func TestHandleLeaderboardUpdateRequested(t *testing.T) {
// 	testCases := []struct {
// 		name                   string
// 		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard
// 		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message
// 		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard)
// 		expectedOutgoingTopics []string
// 		expectHandlerError     bool
// 	}{
// 		{
// 			name: "Success - Sorted participant tags apply to new leaderboard",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				users := generator.GenerateUsers(3)
// 				initial := leaderboardtypes.LeaderboardData{
// 					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: tagPtr(1)},
// 					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: tagPtr(2)},
// 					{UserID: sharedtypes.DiscordID(users[2].UserID), TagNumber: tagPtr(3)},
// 				}
// 				leaderboard, err := testutils.InsertLeaderboard(t, deps.DB, initial)
// 				if err != nil {
// 					t.Fatalf("failed inserting leaderboard: %v", err)
// 				}
// 				return leaderboard
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				users := generator.GenerateUsers(3)
// 				sorted := []string{
// 					fmt.Sprintf("3:%s", users[2].UserID),
// 					fmt.Sprintf("1:%s", users[0].UserID),
// 					fmt.Sprintf("2:%s", users[1].UserID),
// 				}
// 				payload := leaderboardevents.LeaderboardUpdateRequestedPayload{
// 					RoundID:               sharedtypes.RoundID(uuid.New()),
// 					SortedParticipantTags: sorted,
// 				}
// 				payloadBytes, _ := json.Marshal(payload)
// 				msg := message.NewMessage(uuid.New().String(), payloadBytes)
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
// 					t.Fatalf("failed publishing message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				msgs := received[leaderboardevents.LeaderboardUpdated]
// 				if len(msgs) != 1 {
// 					t.Fatalf("Expected 1 success message, got %d", len(msgs))
// 				}
// 				var req leaderboardevents.LeaderboardUpdateRequestedPayload
// 				if err := json.Unmarshal(incoming.Payload, &req); err != nil {
// 					t.Fatalf("Invalid request payload: %v", err)
// 				}
// 				_ = validateLeaderboardUpdatedPayload(t, req, msgs[0])

// 				// DB assertions
// 				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
// 				if err != nil {
// 					t.Fatalf("DB query failed: %v", err)
// 				}
// 				if len(leaderboards) != 2 {
// 					t.Fatalf("Expected 2 leaderboards (1 old + 1 new), got %d", len(leaderboards))
// 				}
// 				oldLB, newLB := leaderboards[0], leaderboards[1]
// 				if oldLB.IsActive {
// 					t.Errorf("Old leaderboard should be inactive")
// 				}
// 				if !newLB.IsActive {
// 					t.Errorf("New leaderboard should be active")
// 				}

// 				expectedData := parseTagUserStrings(req.SortedParticipantTags)
// 				actualData := testutils.ExtractLeaderboardDataMap(newLB.LeaderboardData)

// 				for _, e := range expectedData {
// 					tag, ok := actualData[e.UserID]
// 					if !ok {
// 						t.Errorf("User %s not in new leaderboard", e.UserID)
// 					} else if tag != *e.TagNumber {
// 						t.Errorf("Tag mismatch for %s: expected %d, got %d", e.UserID, *e.TagNumber, tag)
// 					}
// 				}
// 			},
// 			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardUpdated},
// 			expectHandlerError:     false,
// 		},
// 		{
// 			name: "Failure - Invalid JSON Payload",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, leaderboardtypes.LeaderboardData{})
// 				if err != nil {
// 					t.Fatalf("Failed to insert leaderboard: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				msg := message.NewMessage(uuid.New().String(), []byte("not-a-json"))
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				if len(received[leaderboardevents.LeaderboardUpdated]) > 0 {
// 					t.Errorf("Unexpected success message published")
// 				}
// 				if len(received[leaderboardevents.LeaderboardUpdateFailed]) > 0 {
// 					t.Errorf("Unexpected failure message published")
// 				}
// 			},
// 			expectedOutgoingTopics: []string{},
// 			expectHandlerError:     true,
// 		},
// 		{
// 			name: "Failure - Empty SortedParticipantTags",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, leaderboardtypes.LeaderboardData{})
// 				if err != nil {
// 					t.Fatalf("Failed to insert leaderboard: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				payload := leaderboardevents.LeaderboardUpdateRequestedPayload{
// 					RoundID:               sharedtypes.RoundID(uuid.New()),
// 					SortedParticipantTags: []string{},
// 				}
// 				b, _ := json.Marshal(payload)
// 				msg := message.NewMessage(uuid.New().String(), b)
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				if len(received[leaderboardevents.LeaderboardUpdated]) > 0 {
// 					t.Errorf("Expected no success messages, but found some")
// 				}
// 				if len(received[leaderboardevents.LeaderboardUpdateFailed]) > 0 {
// 					t.Errorf("Unexpected failure message published")
// 				}
// 			},
// 			expectedOutgoingTopics: []string{},
// 			expectHandlerError:     true,
// 		},
// 		{
// 			name: "Failure - Incorrect Tag Format",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				lb, err := testutils.InsertLeaderboard(t, deps.DB, leaderboardtypes.LeaderboardData{})
// 				if err != nil {
// 					t.Fatalf("Failed to insert leaderboard: %v", err)
// 				}
// 				return lb
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				payload := leaderboardevents.LeaderboardUpdateRequestedPayload{
// 					RoundID: sharedtypes.RoundID(uuid.New()),
// 					SortedParticipantTags: []string{
// 						"user_no_tag",
// 					},
// 				}
// 				b, _ := json.Marshal(payload)
// 				msg := message.NewMessage(uuid.New().String(), b)
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
// 				if len(received[leaderboardevents.LeaderboardUpdated]) > 0 {
// 					t.Errorf("Unexpected success message")
// 				}
// 				if len(received[leaderboardevents.LeaderboardUpdateFailed]) > 0 {
// 					t.Errorf("Unexpected failure message")
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
