package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidRoundDeleteRequestPayload creates a valid RoundDeleteRequestPayload for testing
func createValidRoundDeleteRequestPayload(roundID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) roundevents.RoundDeleteRequestPayloadV1 {
	return roundevents.RoundDeleteRequestPayloadV1{
		GuildID:              "test-guild",
		RoundID:              roundID,
		RequestingUserUserID: requestingUserID,
	}
}

// createExistingRoundForDeletion creates and stores a round that can be deleted
func createExistingRoundForDeletion(t *testing.T, userID sharedtypes.DiscordID, db bun.IDB) sharedtypes.RoundID {
	t.Helper()
	helper := testutils.NewRoundTestHelper(nil, nil)
	return helper.CreateRoundInDB(t, db, userID)
}

// TestHandleRoundDeleteRequest tests the handler logic for RoundDeleteRequest
func TestHandleRoundDeleteRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Valid Delete Request",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				roundID := createExistingRoundForDeletion(t, data.UserID, deps.DB)
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				// For this test we need to get the roundID from setup, but since we create it here too,
				// we'll create another one. In production you'd pass it through initial state.
				roundID := createExistingRoundForDeletion(t, data.UserID, deps.DB)
				payload := createValidRoundDeleteRequestPayload(roundID, data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundDeleteRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundDeleteAuthorizedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundDeleteAuthorizedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				// Ensure no error messages
				errorMsgs := receivedMsgs[roundevents.RoundDeleteErrorV1]
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Nil Round ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				payload := createValidRoundDeleteRequestPayload(sharedtypes.RoundID(uuid.Nil), data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundDeleteRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{}, // No messages expected
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify no authorized or error messages
				authorizedMsgs := receivedMsgs[roundevents.RoundDeleteAuthorizedV1]
				errorMsgs := receivedMsgs[roundevents.RoundDeleteErrorV1]
				if len(authorizedMsgs) > 0 {
					t.Errorf("Expected no authorized messages, got %d", len(authorizedMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Non-Existent Round ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createValidRoundDeleteRequestPayload(nonExistentRoundID, data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundDeleteRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundDeleteErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundDeleteErrorV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected error message on topic %q", expectedTopic)
				}

				// Ensure no authorized messages
				authorizedMsgs := receivedMsgs[roundevents.RoundDeleteAuthorizedV1]
				if len(authorizedMsgs) > 0 {
					t.Errorf("Expected no authorized messages, got %d", len(authorizedMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Unauthorized User",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data1 := NewTestData()
				roundID := createExistingRoundForDeletion(t, data1.UserID, deps.DB)
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data1 := NewTestData()
				data2 := NewTestData()
				roundID := createExistingRoundForDeletion(t, data1.UserID, deps.DB)
				payload := createValidRoundDeleteRequestPayload(roundID, data2.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundDeleteRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundDeleteErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundDeleteErrorV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected error message on topic %q", expectedTopic)
				}

				// Ensure no authorized messages
				authorizedMsgs := receivedMsgs[roundevents.RoundDeleteAuthorizedV1]
				if len(authorizedMsgs) > 0 {
					t.Errorf("Expected no authorized messages, got %d", len(authorizedMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidJSON := []byte(`invalid json`)
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundDeleteRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{}, // No valid messages expected
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify no authorized or error messages
				authorizedMsgs := receivedMsgs[roundevents.RoundDeleteAuthorizedV1]
				errorMsgs := receivedMsgs[roundevents.RoundDeleteErrorV1]
				if len(authorizedMsgs) > 0 {
					t.Errorf("Expected no authorized messages, got %d", len(authorizedMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, env)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, env, triggerMsg, receivedMsgs, initialState)
				},
				ExpectError:    false,
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
