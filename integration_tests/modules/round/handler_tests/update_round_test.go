package roundhandler_integration_tests

import (
	"encoding/json"
	"strings"
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

func TestHandleRoundUpdateRequest(t *testing.T) {
	deps := SetupTestRoundHandler(t)

	// Ensure NATS/Watermill router is fully wired up
	time.Sleep(2 * time.Second)

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, roundID sharedtypes.RoundID) *message.Message
		validateFn             func(t *testing.T, msg *message.Message)
		expectedOutgoingTopics []string
	}{
		{
			name: "Success - Valid Title Update",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				return testutils.NewRoundTestHelper(env.EventBus, nil).CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, roundID sharedtypes.RoundID) *message.Message {
				title := roundtypes.Title("Updated Round Title")
				return publishUpdateMsg(t, env, createRoundUpdateRequestPayload(roundID, "user-1", &title, nil, nil, nil))
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, msg *message.Message) {
				var p roundevents.RoundEntityUpdatedPayloadV1
				_ = json.Unmarshal(msg.Payload, &p)
				if p.Round.Title != "Updated Round Title" {
					t.Errorf("Expected Title 'Updated Round Title', got '%s'", p.Round.Title)
				}
			},
		},
		{
			name: "Success - Valid Description Update",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				return testutils.NewRoundTestHelper(env.EventBus, nil).CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, roundID sharedtypes.RoundID) *message.Message {
				desc := roundtypes.Description("Updated description")
				return publishUpdateMsg(t, env, createRoundUpdateRequestPayload(roundID, "user-1", nil, &desc, nil, nil))
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, msg *message.Message) {
				var p roundevents.RoundEntityUpdatedPayloadV1
				_ = json.Unmarshal(msg.Payload, &p)
				if p.Round.Description != "Updated description" {
					t.Errorf("Description mismatch: got %v", p.Round.Description)
				}
			},
		},
		{
			name: "Success - Valid StartTime Update",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				return testutils.NewRoundTestHelper(env.EventBus, nil).CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, roundID sharedtypes.RoundID) *message.Message {
				payload := createRoundUpdateRequestPayload(roundID, "user-1", nil, nil, nil, nil)
				loc, err := time.LoadLocation("America/Chicago")
				if err != nil {
					t.Fatalf("Failed to load timezone: %v", err)
				}
				future := time.Now().In(loc).Add(48 * time.Hour)
				futureStr := future.Format("January 2, 2006 at 3:04pm")
				payload.StartTime = &futureStr
				return publishUpdateMsg(t, env, payload)
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdatedV1},
			validateFn: func(t *testing.T, msg *message.Message) {
				var p roundevents.RoundEntityUpdatedPayloadV1
				_ = json.Unmarshal(msg.Payload, &p)
				if p.Round.StartTime == nil {
					t.Error("Expected StartTime to be updated")
				}
			},
		},
		{
			name: "Failure - Zero Round ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return sharedtypes.RoundID(uuid.Nil)
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, roundID sharedtypes.RoundID) *message.Message {
				title := roundtypes.Title("Invalid ID")
				return publishUpdateMsg(t, env, createRoundUpdateRequestPayload(roundID, "user-1", &title, nil, nil, nil))
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdateErrorV1},
			validateFn: func(t *testing.T, msg *message.Message) {
				var p roundevents.RoundUpdateErrorPayloadV1
				_ = json.Unmarshal(msg.Payload, &p)
				if !strings.Contains(strings.ToLower(p.Error), "round id cannot be zero") {
					t.Errorf("Unexpected error msg: %s", p.Error)
				}
			},
		},
		{
			name: "Failure - No Fields to Update",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				return testutils.NewRoundTestHelper(env.EventBus, nil).CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{})
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, roundID sharedtypes.RoundID) *message.Message {
				return publishUpdateMsg(t, env, createRoundUpdateRequestPayload(roundID, "user-1", nil, nil, nil, nil))
			},
			expectedOutgoingTopics: []string{roundevents.RoundUpdateErrorV1},
			validateFn: func(t *testing.T, msg *message.Message) {
				var p roundevents.RoundUpdateErrorPayloadV1
				_ = json.Unmarshal(msg.Payload, &p)
				if !strings.Contains(strings.ToLower(p.Error), "at least one field") {
					t.Errorf("Expected field validation error, got: %s", p.Error)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedID sharedtypes.RoundID

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					state := tc.setupFn(t, deps, env)

					// Robust type handling for RoundID
					switch v := state.(type) {
					case sharedtypes.RoundID:
						capturedID = v
					case string:
						u, err := uuid.Parse(v)
						if err != nil {
							t.Fatalf("Setup returned invalid UUID string: %v", err)
						}
						capturedID = sharedtypes.RoundID(u)
					case uuid.UUID:
						capturedID = sharedtypes.RoundID(v)
					}
					return state
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					// With the new RunTest logic, we just trigger the message.
					// The infrastructure now handles the subscription 'settle' time.
					return tc.publishMsgFn(t, deps, env, capturedID)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, trigger *message.Message, received map[string][]*message.Message, state interface{}) {
					topic := tc.expectedOutgoingTopics[0]
					triggerCorrID := trigger.Metadata.Get(middleware.CorrelationIDMetadataKey)

					var target *message.Message
					msgs := received[topic]

					// 1. Primary Match: Correlation ID
					// (This ensures we aren't looking at messages from previous test runs)
					for _, m := range msgs {
						if m.Metadata.Get(middleware.CorrelationIDMetadataKey) == triggerCorrID {
							target = m
							break
						}
					}

					// 2. Fallback Match: Payload Content
					if target == nil {
						expectedUUIDStr := strings.ToLower(capturedID.String())
						for _, m := range msgs {
							payloadStr := strings.ToLower(string(m.Payload))
							if strings.Contains(payloadStr, expectedUUIDStr) {
								target = m
								break
							}
						}
					}

					// If we still don't have it, fail with a clear message
					if target == nil {
						t.Fatalf("[%s] Topic %s received %d msgs, but none matched this test. Expected CorrID: %s",
							tc.name, topic, len(msgs), triggerCorrID)
					}

					tc.validateFn(t, target)
				},
				MessageTimeout: 10 * time.Second,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}

// Helpers unchanged as they are already functioning correctly
func publishUpdateMsg(t *testing.T, env *testutils.TestEnvironment, payload interface{}) *message.Message {
	b, _ := json.Marshal(payload)
	m := message.NewMessage(uuid.New().String(), b)
	m.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	_ = testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundUpdateRequestedV1, m)
	return m
}

func createRoundUpdateRequestPayload(id sharedtypes.RoundID, uid sharedtypes.DiscordID, title *roundtypes.Title, desc *roundtypes.Description, loc *roundtypes.Location, st *sharedtypes.StartTime) roundevents.UpdateRoundRequestedPayloadV1 {
	tz := roundtypes.Timezone("America/Chicago")
	p := roundevents.UpdateRoundRequestedPayloadV1{
		RoundID: id, UserID: uid, GuildID: "test-guild", Timezone: &tz,
	}
	if title != nil {
		p.Title = title
	}
	if desc != nil {
		p.Description = desc
	}
	if loc != nil {
		p.Location = loc
	}
	if st != nil {
		s := time.Time(*st).Format("January 2, 2006 at 3:04pm")
		p.StartTime = &s
	}
	return p
}
