package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
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

// MessagePayload interface for handling different message payloads
type MessagePayload interface {
	GetUserID() sharedtypes.DiscordID
	GetTagNumber() *sharedtypes.TagNumber
}

// TestCase represents a test case for the tag assignment handler
type tagAssignmentTestCase struct {
	name                   string
	setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard
	publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message
	validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard)
	expectedOutgoingTopics []string
	expectHandlerError     bool
}

// Helper function to create and publish a tag assignment request message
func createTagAssignmentRequestMessage(
	t *testing.T,
	userID sharedtypes.DiscordID,
	tagNumber sharedtypes.TagNumber,
	source string,
	updateType string,
) *message.Message {
	t.Helper()

	tagPtr := tagPtr(tagNumber)
	payload := leaderboardevents.TagAssignmentRequestedPayload{
		UserID:     userID,
		TagNumber:  tagPtr,
		Source:     source,
		UpdateType: updateType,
		UpdateID:   sharedtypes.RoundID(uuid.New()),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg
}

// Helper to validate basic payload properties
func validatePayloads(t *testing.T, request, response MessagePayload) {
	t.Helper()

	if response.GetUserID() != request.GetUserID() {
		t.Errorf("Payload UserID mismatch: expected %q, got %q",
			request.GetUserID(), response.GetUserID())
	}

	if *response.GetTagNumber() != *request.GetTagNumber() {
		t.Errorf("Payload TagNumber mismatch: expected %d, got %d",
			*request.GetTagNumber(), *response.GetTagNumber())
	}
}

// TestHandleTagAssignmentRequested runs integration tests for the tag assignment handler
func TestHandleTagAssignmentRequested(t *testing.T) {
	updateUser := testutils.NewTestDataGenerator(time.Now().UnixNano()).GenerateUsers(1)[0]
	updateUserID := sharedtypes.DiscordID(updateUser.UserID)

	swapUsers := testutils.NewTestDataGenerator(time.Now().UnixNano() + 1).GenerateUsers(2)
	swapUserA := sharedtypes.DiscordID(swapUsers[0].UserID)
	swapUserB := sharedtypes.DiscordID(swapUsers[1].UserID)
	testCases := []struct {
		name                   string
		setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard
		publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message
		validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard)
		expectedOutgoingTopics []string
		expectHandlerError     bool
	}{
		{
			name: "Success - Assign New Tag",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
				users := generator.GenerateUsers(1)
				data := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: tagPtr(1)},
				}
				lb, err := testutils.InsertLeaderboard(t, deps.DB, data)
				if err != nil {
					t.Fatalf("Insert failed: %v", err)
				}
				return lb
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
				newUser := generator.GenerateUsers(1)[0]
				requestingUser := generator.GenerateUsers(1)[0]

				msg := createTagAssignmentRequestMessage(t,
					sharedtypes.DiscordID(newUser.UserID),
					10,
					string(sharedtypes.DiscordID(requestingUser.UserID)),
					"manual",
				)
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardTagAssignmentRequested, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				expectedTopic := leaderboardevents.LeaderboardTagAssignmentSuccess
				msgs := received[expectedTopic]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on %q, got %d", expectedTopic, len(msgs))
				}
				req, _ := testutils.ParsePayload[leaderboardevents.TagAssignmentRequestedPayload](incomingMsg)
				res, _ := testutils.ParsePayload[leaderboardevents.TagAssignedPayload](msgs[0])
				validatePayloads(t, req, res)

				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardTagAssignmentSuccess},
			expectHandlerError:     false,
		},
		{
			name: "Success - Update Existing Tag",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{
					{UserID: updateUserID, TagNumber: tagPtr(20)},
				}
				lb, err := testutils.InsertLeaderboard(t, deps.DB, data)
				if err != nil {
					t.Fatalf("Insert failed: %v", err)
				}
				return lb
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
				msg := createTagAssignmentRequestMessage(t, updateUserID, 99, "admin", "update")
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardTagAssignmentRequested, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[leaderboardevents.LeaderboardTagAssignmentSuccess]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 success message, got %d", len(msgs))
				}
				req, _ := testutils.ParsePayload[leaderboardevents.TagAssignmentRequestedPayload](incomingMsg)
				res, _ := testutils.ParsePayload[leaderboardevents.TagAssignedPayload](msgs[0])
				validatePayloads(t, req, res)
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardTagAssignmentSuccess},
			expectHandlerError:     false,
		},
		{
			name: "Success - Add Brand New User",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
				lb, err := testutils.InsertLeaderboard(t, deps.DB, leaderboardtypes.LeaderboardData{})
				if err != nil {
					t.Fatalf("Insert failed: %v", err)
				}
				return lb
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
				newUser := generator.GenerateUsers(1)[0]
				msg := createTagAssignmentRequestMessage(t, sharedtypes.DiscordID(newUser.UserID), 7, "system", "new")
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardTagAssignmentRequested, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[leaderboardevents.LeaderboardTagAssignmentSuccess]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message, got %d", len(msgs))
				}
				req, _ := testutils.ParsePayload[leaderboardevents.TagAssignmentRequestedPayload](incomingMsg)
				res, _ := testutils.ParsePayload[leaderboardevents.TagAssignedPayload](msgs[0])
				validatePayloads(t, req, res)
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardTagAssignmentSuccess},
			expectHandlerError:     false,
		},
		{
			name: "Tag Swap Triggered",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{
					{UserID: swapUserA, TagNumber: tagPtr(10)},
					{UserID: swapUserB, TagNumber: tagPtr(20)},
				}
				lb, err := testutils.InsertLeaderboard(t, deps.DB, data)
				if err != nil {
					t.Fatalf("Insert failed: %v", err)
				}
				return lb
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
				msg := createTagAssignmentRequestMessage(t, swapUserA, 20, "manual", "update")
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardTagAssignmentRequested, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[leaderboardevents.TagSwapRequested]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 TagSwapRequested message, got %d", len(msgs))
				}
				swap, _ := testutils.ParsePayload[leaderboardevents.TagSwapRequestedPayload](msgs[0])
				req, _ := testutils.ParsePayload[leaderboardevents.TagAssignmentRequestedPayload](incomingMsg)
				if swap.RequestorID != req.UserID {
					t.Errorf("Expected RequestorID %q, got %q", req.UserID, swap.RequestorID)
				}
				if swap.TargetID == "" {
					t.Errorf("Expected non-empty TargetID")
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapRequested},
			expectHandlerError:     false,
		},
		{
			name: "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
				lb, err := testutils.InsertLeaderboard(t, deps.DB, leaderboardtypes.LeaderboardData{})
				if err != nil {
					t.Fatalf("Insert failed: %v", err)
				}
				return lb
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("not json"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardTagAssignmentRequested, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				for _, topic := range []string{
					leaderboardevents.LeaderboardTagAssignmentSuccess,
					leaderboardevents.LeaderboardTagAssignmentFailed,
				} {
					if len(received[topic]) > 0 {
						t.Errorf("Expected no messages on %s, got %d", topic, len(received[topic]))
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
			generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, generator)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, generator)
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
