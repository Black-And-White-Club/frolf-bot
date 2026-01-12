package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleRoundStarted(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Start Round with Single Participant",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Move creation into publish step so we can control the round title (DB is source of truth)
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Ensure the round stored in DB has the title we expect the Discord payload to contain
				if _, err := (&rounddb.RoundDBImpl{DB: env.DB}).UpdateRound(env.Ctx, sharedtypes.GuildID("test-guild"), roundID, &roundtypes.Round{Title: roundtypes.Title("Test Round")}); err != nil {
					t.Fatalf("Failed to set round title in DB: %v", err)
				}

				// Publish identity-only start request (handler/service will read DB)
				payload := roundevents.RoundStartRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundStartRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundStartedDiscordV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundStartedDiscordV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var payload roundevents.DiscordRoundStartPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if payload.Title != roundtypes.Title("Test Round") {
					t.Errorf("Title mismatch: expected Test Round, got %s", payload.Title)
				}

				if len(payload.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(payload.Participants))
				} else {
					if payload.Participants[0].Response != roundtypes.ResponseAccept {
						t.Errorf("Expected response ACCEPT, got %s", payload.Participants[0].Response)
					}
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Start Round with Multiple Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				score1 := sharedtypes.Score(2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data4.UserID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				// Publish identity-only start request
				payload := roundevents.RoundStartRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundStartRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundStartedDiscordV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundStartedDiscordV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var payload roundevents.DiscordRoundStartPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if len(payload.Participants) != 3 {
					t.Errorf("Expected 3 participants, got %d", len(payload.Participants))
				}

				// Verify all response types are present
				for _, p := range payload.Participants {
					if p.Response != roundtypes.ResponseAccept && p.Response != roundtypes.ResponseTentative {
						t.Errorf("Unexpected response type: %s", p.Response)
					}
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Start Round with No Participants",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})

				// Publish identity-only start request
				payload := roundevents.RoundStartRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundStartRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundStartedDiscordV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundStartedDiscordV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var payload roundevents.DiscordRoundStartPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if len(payload.Participants) != 0 {
					t.Errorf("Expected 0 participants, got %d", len(payload.Participants))
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Start Round with Participant Tag Numbers",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				// Publish identity-only start request
				payload := roundevents.RoundStartRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundStartRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundStartedDiscordV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundStartedDiscordV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var payload roundevents.DiscordRoundStartPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if len(payload.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(payload.Participants))
				}

				for _, p := range payload.Participants {
					if p.TagNumber == nil {
						t.Errorf("Expected participant %s to have a tag number", p.UserID)
					}
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Round Not Found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())

				// Publish identity-only start request for a non-existent round
				payload := roundevents.RoundStartRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: nonExistentRoundID,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundStartRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundStartFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				expectedTopic := roundevents.RoundStartFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
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
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundStartRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify no messages were published for invalid JSON
				successMsgs := receivedMsgs[roundevents.RoundStartedDiscordV1]
				errorMsgs := receivedMsgs[roundevents.RoundStartFailedV1]
				if len(successMsgs) > 0 || len(errorMsgs) > 0 {
					t.Errorf("Expected no messages for invalid JSON, got success=%d, error=%d", len(successMsgs), len(errorMsgs))
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

// Helper functions for creating payloads - UNIQUE TO ROUND STARTED TESTS
func createRoundStartedPayload(roundID sharedtypes.RoundID, title string, startTime *time.Time, location *roundtypes.Location) roundevents.RoundStartedPayloadV1 {
	var sharedStartTime *sharedtypes.StartTime
	if startTime != nil {
		st := sharedtypes.StartTime(*startTime)
		sharedStartTime = &st
	}

	return roundevents.RoundStartedPayloadV1{
		RoundID:   roundID,
		Title:     roundtypes.Title(title),
		Location:  location,
		StartTime: sharedStartTime,
		ChannelID: "test-channel-id",
		GuildID:   "test-guild",
	}
}
