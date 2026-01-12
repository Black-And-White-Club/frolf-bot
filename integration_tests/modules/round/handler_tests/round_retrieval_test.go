package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleGetRoundRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Retrieve Existing Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				return struct{ RoundID sharedtypes.RoundID; UserID sharedtypes.DiscordID }{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				init := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, init.UserID, roundtypes.RoundStateUpcoming)

				payload := createGetRoundRequestPayload(roundID, init.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.GetRoundRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundRetrievedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundRetrievedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundRetrievedV1)
				}
				var payload roundtypes.Round
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.ID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
				if payload.CreatedBy == "" {
					t.Error("Expected CreatedBy to be set")
				}
				if payload.Title == "" {
					t.Error("Expected Title to be set")
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Retrieve Round with Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, response, &tagNumber)
				return struct{ RoundID sharedtypes.RoundID; UserID sharedtypes.DiscordID }{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, response, &tagNumber)

				payload := createGetRoundRequestPayload(roundID, data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.GetRoundRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundRetrievedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundRetrievedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundRetrievedV1)
				}
				var payload roundtypes.Round
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.ID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
				if len(payload.Participants) == 0 {
					t.Error("Expected round to have participants")
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Retrieve In-Progress Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				return struct{ RoundID sharedtypes.RoundID; UserID sharedtypes.DiscordID }{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				payload := createGetRoundRequestPayload(roundID, data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.GetRoundRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundRetrievedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundRetrievedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundRetrievedV1)
				}
				var payload roundtypes.Round
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.State != roundtypes.RoundStateInProgress {
					t.Errorf("Expected State %s, got %s", roundtypes.RoundStateInProgress, payload.State)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Retrieve Completed Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateFinalized)
				return struct{ RoundID sharedtypes.RoundID; UserID sharedtypes.DiscordID }{RoundID: roundID, UserID: data.UserID}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateFinalized)
				payload := createGetRoundRequestPayload(roundID, data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.GetRoundRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundRetrievedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundRetrievedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundRetrievedV1)
				}
				var payload roundtypes.Round
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.State != roundtypes.RoundStateFinalized {
					t.Errorf("Expected State %s, got %s", roundtypes.RoundStateFinalized, payload.State)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Retrieve Non-Existent Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createGetRoundRequestPayload(nonExistentRoundID, data.UserID)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.GetRoundRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundRetrievalFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundRetrievalFailedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one error message on topic %q", roundevents.RoundRetrievalFailedV1)
				}
				var payload roundevents.RoundRetrievalFailedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set in error payload")
				}
				if payload.Error == "" {
					t.Error("Expected error message to be populated")
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
				invalidJSON := []byte("invalid json")
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.GetRoundRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				validatedMsgs := receivedMsgs[roundevents.RoundRetrievedV1]
				errorMsgs := receivedMsgs[roundevents.RoundRetrievalFailedV1]
				if len(validatedMsgs) > 0 || len(errorMsgs) > 0 {
					t.Errorf("Expected no messages for invalid JSON, got validated=%d, error=%d", len(validatedMsgs), len(errorMsgs))
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
				MessageTimeout: 5 * time.Second,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}

// Helper functions for creating payloads - UPDATED TO MATCH HANDLER
func createGetRoundRequestPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) roundevents.GetRoundRequestPayloadV1 {
	return roundevents.GetRoundRequestPayloadV1{
		GuildID:        "test-guild",
		RoundID:        roundID,
		EventMessageID: "test-event-message-id",
		UserID:         userID,
	}
}
