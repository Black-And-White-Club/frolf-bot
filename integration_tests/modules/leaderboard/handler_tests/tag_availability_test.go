package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
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

// Helper function to create a tag availability check request message
func createTagAvailabilityCheckRequestMessage(
	t *testing.T,
	userID sharedtypes.DiscordID,
	tagNumber sharedtypes.TagNumber,
) (*message.Message, error) {
	t.Helper() // Mark this as a helper function

	tagPtr := tagPtr(tagNumber)
	payload := leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
		GuildID:   sharedtypes.GuildID("test_guild"),
		UserID:    userID,
		TagNumber: tagPtr,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	return msg, nil
}

// Helper to validate response properties for tag available
func validateTagAvailableResponse(
	t *testing.T,
	requestPayload *leaderboardevents.TagAvailabilityCheckRequestedPayloadV1,
	availablePayload *leaderboardevents.LeaderboardTagAvailablePayloadV1,
) {
	t.Helper() // Mark this as a helper function

	if availablePayload.GuildID != requestPayload.GuildID {
		t.Errorf("GuildID mismatch in available payload: expected %q, got %q",
			requestPayload.GuildID, availablePayload.GuildID)
	}

	if availablePayload.UserID != requestPayload.UserID {
		t.Errorf("UserID mismatch in available payload: expected %q, got %q",
			requestPayload.UserID, availablePayload.UserID)
	}

	if *availablePayload.TagNumber != *requestPayload.TagNumber {
		t.Errorf("TagNumber mismatch in available payload: expected %d, got %d",
			*requestPayload.TagNumber, *availablePayload.TagNumber)
	}
}

// Helper to validate response properties for tag unavailable
func validateTagUnavailableResponse(
	t *testing.T,
	requestPayload *leaderboardevents.TagAvailabilityCheckRequestedPayloadV1,
	unavailablePayload *leaderboardevents.LeaderboardTagUnavailablePayloadV1,
) {
	t.Helper() // Mark this as a helper function

	if unavailablePayload.GuildID != requestPayload.GuildID {
		t.Errorf("GuildID mismatch in unavailable payload: expected %q, got %q",
			requestPayload.GuildID, unavailablePayload.GuildID)
	}

	if unavailablePayload.UserID != requestPayload.UserID {
		t.Errorf("UserID mismatch in unavailable payload: expected %q, got %q",
			requestPayload.UserID, unavailablePayload.UserID)
	}

	if *unavailablePayload.TagNumber != *requestPayload.TagNumber {
		t.Errorf("TagNumber mismatch in unavailable payload: expected %d, got %d",
			*requestPayload.TagNumber, *unavailablePayload.TagNumber)
	}
}

// TestHandleTagAvailabilityCheckRequested runs integration tests for the tag availability handler
func TestHandleTagAvailabilityCheckRequested(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
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
			name:  "Success - Tag Available",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 1},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				newUser := generator.GenerateUsers(1)[0]

				msg, err := createTagAvailabilityCheckRequestMessage(
					t,
					sharedtypes.DiscordID(newUser.UserID),
					10, // Tag number that should be available
				)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagAvailabilityCheckRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check TagAvailable response
				availableTopic := leaderboardevents.LeaderboardTagAvailableV1
				msgs := receivedMsgs[availableTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", availableTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", availableTopic, len(msgs))
				}

				// Parse payloads
				requestPayload, err := testutils.ParsePayload[leaderboardevents.TagAvailabilityCheckRequestedPayloadV1](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				availablePayload, err := testutils.ParsePayload[leaderboardevents.LeaderboardTagAvailablePayloadV1](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				// Validate available payload
				validateTagAvailableResponse(t, requestPayload, availablePayload)

				// Validate correlation ID
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// Check for unexpected messages
				unexpectedTopic := leaderboardevents.LeaderboardTagUnavailableV1
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
				unexpectedFailureTopic := leaderboardevents.TagAvailabilityCheckFailedV1
				if len(receivedMsgs[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(receivedMsgs[unexpectedFailureTopic]))
				}
			},
			expectedOutgoingTopics: []string{
				leaderboardevents.LeaderboardTagAvailableV1,
			},
			expectHandlerError: false,
		},
		{
			name:  "Success - Tag Unavailable (Already Taken)",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initialData := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 42},
				}

				return testutils.SetupLeaderboardWithEntries(t, deps.DB, initialData, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				newUser := generator.GenerateUsers(1)[0]

				msg, err := createTagAvailabilityCheckRequestMessage(
					t,
					sharedtypes.DiscordID(newUser.UserID),
					42, // Tag number that is already taken
				)
				if err != nil {
					t.Fatalf("%v", err)
				}

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagAvailabilityCheckRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				unavailableTopic := leaderboardevents.LeaderboardTagUnavailableV1
				msgs := receivedMsgs[unavailableTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", unavailableTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", unavailableTopic, len(msgs))
				}

				// Parse payloads
				requestPayload, err := testutils.ParsePayload[leaderboardevents.TagAvailabilityCheckRequestedPayloadV1](incomingMsg)
				if err != nil {
					t.Fatalf("%v", err)
				}

				unavailablePayload, err := testutils.ParsePayload[leaderboardevents.LeaderboardTagUnavailablePayloadV1](msgs[0])
				if err != nil {
					t.Fatalf("%v", err)
				}

				// Validate payload
				validateTagUnavailableResponse(t, requestPayload, unavailablePayload)

				// Validate correlation ID
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// Check for unexpected messages
				availableTopic := leaderboardevents.LeaderboardTagAvailableV1
				assignTopic := leaderboardevents.LeaderboardTagAssignmentRequestedV1
				failureTopic := leaderboardevents.TagAvailabilityCheckFailedV1

				if len(receivedMsgs[availableTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						availableTopic, len(receivedMsgs[availableTopic]))
				}
				if len(receivedMsgs[assignTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						assignTopic, len(receivedMsgs[assignTopic]))
				}
				if len(receivedMsgs[failureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						failureTopic, len(receivedMsgs[failureTopic]))
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardTagUnavailableV1},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Invalid Message Payload",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid json payload"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.TagAvailabilityCheckRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Check for unexpected messages
				availableTopic := leaderboardevents.LeaderboardTagAvailableV1
				unavailableTopic := leaderboardevents.LeaderboardTagUnavailableV1
				assignTopic := leaderboardevents.LeaderboardTagAssignmentRequestedV1
				failureTopic := leaderboardevents.TagAvailabilityCheckFailedV1

				if len(receivedMsgs[availableTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						availableTopic, len(receivedMsgs[availableTopic]))
				}
				if len(receivedMsgs[unavailableTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						unavailableTopic, len(receivedMsgs[unavailableTopic]))
				}
				if len(receivedMsgs[assignTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						assignTopic, len(receivedMsgs[assignTopic]))
				}
				if len(receivedMsgs[failureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d",
						failureTopic, len(receivedMsgs[failureTopic]))
				}

				// Validate leaderboard state in database
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("%v", err)
				}
				if len(leaderboards) != 1 {
					t.Fatalf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
				}
				leaderboard := leaderboards[0]
				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
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
