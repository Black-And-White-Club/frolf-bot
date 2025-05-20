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

	// Ensure both pointers are non-nil before dereferencing
	if request.GetTagNumber() != nil && response.GetTagNumber() != nil {
		if *response.GetTagNumber() != *request.GetTagNumber() {
			t.Errorf("Payload TagNumber mismatch: expected %d, got %d",
				*request.GetTagNumber(), *response.GetTagNumber())
		}
	} else if request.GetTagNumber() != response.GetTagNumber() {
		t.Errorf("Payload TagNumber mismatch: one is nil, the other isn't. Request: %v, Response: %v",
			request.GetTagNumber(), response.GetTagNumber())
	}
}

// TestHandleTagAssignmentRequested runs integration tests for the tag assignment handler
func TestHandleTagAssignmentRequested(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	updateUser := testutils.NewTestDataGenerator(time.Now().UnixNano()).GenerateUsers(1)[0]
	updateUserID := sharedtypes.DiscordID(updateUser.UserID)

	swapUsers := testutils.NewTestDataGenerator(time.Now().UnixNano() + 1).GenerateUsers(2)
	swapUserA := sharedtypes.DiscordID(swapUsers[0].UserID)
	swapUserB := sharedtypes.DiscordID(swapUsers[1].UserID)
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
			name:  "Success - Assign New Tag",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 1},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
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
			name:  "Success - Update Existing Tag",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{
					{UserID: updateUserID, TagNumber: 20},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
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
			name:  "Success - Add Brand New User",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
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
			name:  "Success - Tag Swap Triggered and Processed", // This is an odd one because it verifies the downstream handler
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				data := leaderboardtypes.LeaderboardData{
					{UserID: swapUserA, TagNumber: 10},
					{UserID: swapUserB, TagNumber: 20},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, data, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				// A requests tag 20 (currently held by User B)
				msg := createTagAssignmentRequestMessage(t, swapUserA, 20, "manual", "update")
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardTagAssignmentRequested, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				processedTopic := leaderboardevents.TagSwapProcessed
				msgs := received[processedTopic]
				if len(msgs) < 1 {
					t.Fatalf("Expected at least 1 message on %q, got %d", processedTopic, len(msgs))
				}
				t.Logf("Received downstream swap processed event: %s", msgs[0].UUID)
			},
			expectedOutgoingTopics: []string{leaderboardevents.TagSwapProcessed},
		},
		{
			name:  "Failure - Invalid JSON Message",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
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
