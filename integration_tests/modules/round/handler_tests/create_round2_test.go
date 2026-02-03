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

func ensureStreams(t *testing.T, env *testutils.TestEnvironment) {
	// Ensure 'discord' stream exists as it's often missing in fresh envs
	err := env.EventBus.CreateStream(env.Ctx, "discord")
	if err != nil {
		t.Logf("CreateStream 'discord' result: %v", err)
	}
	// Ensure 'round' stream exists
	err = env.EventBus.CreateStream(env.Ctx, "round")
	if err != nil {
		t.Logf("CreateStream 'round' result: %v", err)
	}
}

// createValidRoundEntityCreatedPayload creates a valid RoundEntityCreatedPayload for testing
func createValidRoundEntityCreatedPayload(userID sharedtypes.DiscordID) roundevents.RoundEntityCreatedPayloadV1 {
	now := time.Now()
	startTime := sharedtypes.StartTime(now.Add(24 * time.Hour))
	description := roundtypes.Description("Test round for deletion")
	location := roundtypes.Location("Test Course")

	return roundevents.RoundEntityCreatedPayloadV1{
		GuildID: "test-guild",
		Round: roundtypes.Round{
			ID:          sharedtypes.RoundID(uuid.New()),
			Title:       "Test Round",
			Description: description,
			Location:    location,
			StartTime:   &startTime,
			CreatedBy:   userID,
			State:       roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{
				{UserID: userID, Response: roundtypes.ResponseAccept},
			},
			Finalized: roundtypes.Finalized(false),
		},
		DiscordChannelID: "test-channel-123",
		DiscordGuildID:   "test-guild",
	}
}

// createMinimalRoundEntityCreatedPayload creates a minimal but valid payload
func createMinimalRoundEntityCreatedPayload(userID sharedtypes.DiscordID) roundevents.RoundEntityCreatedPayloadV1 {
	roundID := sharedtypes.RoundID(uuid.New())
	now := time.Now()
	startTime := sharedtypes.StartTime(now.Add(24 * time.Hour))

	description := roundtypes.Description("Quick round")
	location := roundtypes.Location("Local Course")

	return roundevents.RoundEntityCreatedPayloadV1{
		GuildID: "test-guild",
		Round: roundtypes.Round{
			ID:          roundID,
			Title:       "Quick Round",
			Description: description,
			Location:    location,
			StartTime:   &startTime,
			CreatedBy:   userID,
			State:       roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{
				{UserID: userID, Response: roundtypes.ResponseAccept},
			},
			Finalized: roundtypes.Finalized(false),
		},
		DiscordChannelID: "test-channel-456",
		DiscordGuildID:   "test-guild",
	}
}

// Note: helper functions that relied on MessageCapture/RoundTestHelper have been
// removed during refactor; payload creation helpers remain below.

// TestHandleRoundEntityCreated runs integration tests for the round entity created handler
func TestHandleRoundEntityCreated(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Handler Processes Valid Message and Publishes Success Event",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				return payload
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := t.Cleanup // noop to satisfy lint when not used
				_ = payload
				init := NewTestData()
				p := createValidRoundEntityCreatedPayload(init.UserID)
				payloadBytes, err := json.Marshal(p)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundCreatedV1)
				}
				var payload roundevents.RoundCreatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name: "Success - Handler Processes Minimal Valid Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				payload := createMinimalRoundEntityCreatedPayload(data.UserID)
				return payload
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				init := NewTestData()
				p := createMinimalRoundEntityCreatedPayload(init.UserID)
				payloadBytes, err := json.Marshal(p)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundCreatedV1)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name: "Success - Handler Processes Message with Complex Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				payload.Round.Participants = []roundtypes.Participant{
					{UserID: data.UserID, Response: roundtypes.ResponseAccept},
					{UserID: sharedtypes.DiscordID("user123"), Response: roundtypes.ResponseTentative},
					{UserID: sharedtypes.DiscordID("user456"), Response: roundtypes.ResponseDecline},
				}
				return payload
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				init := NewTestData()
				p := createValidRoundEntityCreatedPayload(init.UserID)
				p.Round.Participants = []roundtypes.Participant{
					{UserID: init.UserID, Response: roundtypes.ResponseAccept},
					{UserID: sharedtypes.DiscordID("user123"), Response: roundtypes.ResponseTentative},
					{UserID: sharedtypes.DiscordID("user456"), Response: roundtypes.ResponseDecline},
				}
				payloadBytes, err := json.Marshal(p)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundCreatedV1)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name: "Success - Handler Processes Different Round States",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				payload.Round.State = roundtypes.RoundStateInProgress
				return payload
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				init := NewTestData()
				p := createValidRoundEntityCreatedPayload(init.UserID)
				p.Round.State = roundtypes.RoundStateInProgress
				payloadBytes, err := json.Marshal(p)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundCreatedV1)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name: "Success - Handler Processes Multiple Messages Concurrently",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// no initial state
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				// publish two messages quickly; RunTest will collect both
				d1 := NewTestData()
				d2 := NewTestData()
				p1 := createValidRoundEntityCreatedPayload(d1.UserID)
				p2 := createValidRoundEntityCreatedPayload(d2.UserID)
				b1, err := json.Marshal(p1)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				b2, err := json.Marshal(p2)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg1 := message.NewMessage(uuid.New().String(), b1)
				msg1.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg2 := message.NewMessage(uuid.New().String(), b2)
				msg2.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg1); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg2); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg1
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) < 2 {
					t.Fatalf("Expected at least 2 messages on topic %q, got %d", roundevents.RoundCreatedV1, len(msgs))
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name:    "Failure - Handler Rejects Invalid JSON and Doesn't Publish Events",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidJSON := []byte("invalid json")
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				created := receivedMsgs[roundevents.RoundCreatedV1]
				failed := receivedMsgs[roundevents.RoundCreationFailedV1]
				if len(created) > 0 || len(failed) > 0 {
					t.Errorf("Expected no messages for invalid JSON, got created=%d, failed=%d", len(created), len(failed))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Handler Preserves Message Correlation ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				return payload
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				p := createValidRoundEntityCreatedPayload(NewTestData().UserID)
				payloadBytes, err := json.Marshal(p)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one round.created message")
				}
				var found *message.Message
				var payload roundevents.RoundCreatedPayloadV1
				for _, m := range msgs {
					if err := deps.TestHelpers.UnmarshalPayload(m, &payload); err == nil && payload.RoundID != sharedtypes.RoundID(uuid.Nil) {
						found = m
						break
					}
				}
				if found == nil {
					t.Fatalf("No round.created message found matching criteria")
				}
				origCID := triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey)
				resCID := found.Metadata.Get(middleware.CorrelationIDMetadataKey)
				if origCID == "" {
					t.Errorf("Original message correlation ID was empty")
				}
				if origCID != resCID {
					t.Errorf("Correlation ID not preserved: original=%s, result=%s", origCID, resCID)
				}
			},
			timeout: 10 * time.Second,
		},
		{
			name: "Success - Handler Publishes Correct Event Topic",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				payload := createValidRoundEntityCreatedPayload(data.UserID)
				return payload
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				p := createValidRoundEntityCreatedPayload(NewTestData().UserID)
				payloadBytes, err := json.Marshal(p)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEntityCreatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundCreatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected round.created message")
				}
			},
			timeout: 10 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			ensureStreams(t, deps.TestEnvironment)
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
