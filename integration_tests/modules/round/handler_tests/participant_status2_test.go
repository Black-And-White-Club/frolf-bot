package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleParticipantJoinValidationRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Accept Response for Scheduled Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				id := helper.CreateRoundInDB(t, deps.DB, data.UserID)
				return id
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundInDB(t, deps.DB, data.UserID)
				payload := createPayload(roundID, data.UserID, roundtypes.ResponseAccept)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantJoinValidationRequestedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{sharedevents.LeaderboardTagLookupRequestedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[sharedevents.LeaderboardTagLookupRequestedV1]
				if len(msgs) == 0 {
					t.Fatalf("expected leaderboard tag lookup request, got none")
				}
			},
			timeout: 1 * time.Second,
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

// Test-specific helper functions - only used in this file

func createPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response) roundevents.ParticipantJoinValidationRequestPayloadV1 {
	return roundevents.ParticipantJoinValidationRequestPayloadV1{
		RoundID:  roundID,
		UserID:   userID,
		Response: response,
		GuildID:  "test-guild",
	}
}

// Note: MessageCapture-dependent helpers removed during refactor. Tests use
// testutils.RunTest and validate via receivedMsgs and deps.TestHelpers.UnmarshalPayload.

func validateTagLookupRequest(t *testing.T, deps HandlerTestDeps, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.TagLookupRequestPayloadV1 {
	t.Helper()

	var result roundevents.TagLookupRequestPayloadV1
	if err := deps.TestHelpers.UnmarshalPayload(msg, &result); err != nil {
		t.Fatalf("Failed to unmarshal tag lookup request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Response != expectedResponse {
		t.Errorf("Response mismatch: expected %s, got %s", expectedResponse, result.Response)
	}

	return &result
}

func validateParticipantStatusUpdateRequest(t *testing.T, deps HandlerTestDeps, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.ParticipantJoinRequestPayloadV1 {
	t.Helper()

	var result roundevents.ParticipantJoinRequestPayloadV1
	if err := deps.TestHelpers.UnmarshalPayload(msg, &result); err != nil {
		t.Fatalf("Failed to unmarshal participant status update request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Response != expectedResponse {
		t.Errorf("Response mismatch: expected %s, got %s", expectedResponse, result.Response)
	}

	return &result
}

func validateParticipantJoinError(t *testing.T, deps HandlerTestDeps, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	t.Helper()

	var result roundevents.RoundParticipantJoinErrorPayloadV1
	if err := deps.TestHelpers.UnmarshalPayload(msg, &result); err != nil {
		t.Fatalf("Failed to unmarshal participant join error message: %v", err)
	}

	if result.ParticipantJoinRequest == nil {
		t.Error("Expected ParticipantJoinRequest to be populated in error payload")
		return
	}

	if result.ParticipantJoinRequest.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.ParticipantJoinRequest.RoundID)
	}

	if result.ParticipantJoinRequest.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.ParticipantJoinRequest.UserID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// MessageCapture-dependent publish/wait helpers removed; tests should use
// testutils.RunTest and validate via receivedMsgs in the ValidateFn.
