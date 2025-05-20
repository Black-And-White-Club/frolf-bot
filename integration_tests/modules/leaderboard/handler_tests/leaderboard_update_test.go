package leaderboardhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

// Helper: Parse "tag:user" string into LeaderboardEntry
func parseTagUserStrings(tagUsers []string) leaderboardtypes.LeaderboardData {
	data := leaderboardtypes.LeaderboardData{}
	for _, entry := range tagUsers {
		parts := strings.Split(entry, ":")
		if len(parts) == 2 {
			tag, _ := strconv.Atoi(parts[0])
			data = append(data, leaderboardtypes.LeaderboardEntry{
				UserID:    sharedtypes.DiscordID(parts[1]),
				TagNumber: sharedtypes.TagNumber(tag),
			})
		}
	}
	return data
}

// Helper: Validate success response for leaderboard update
func validateLeaderboardUpdatedPayload(t *testing.T, req leaderboardevents.LeaderboardUpdateRequestedPayload, msg *message.Message) leaderboardevents.LeaderboardUpdatedPayload {
	t.Helper()
	var res leaderboardevents.LeaderboardUpdatedPayload
	if err := json.Unmarshal(msg.Payload, &res); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if res.RoundID != req.RoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", req.RoundID, res.RoundID)
	}

	if msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != "" &&
		msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != msg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		t.Errorf("Correlation ID mismatch")
	}
	return res
}

// Integration test for LeaderboardUpdateRequested handler
func TestHandleLeaderboardUpdateRequested(t *testing.T) {
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
			name:  "Success - Sorted participant tags apply to new leaderboard",
			users: generator.GenerateUsers(3),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				initial := leaderboardtypes.LeaderboardData{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), TagNumber: 2},
					{UserID: sharedtypes.DiscordID(users[2].UserID), TagNumber: 3},
				}
				// SetupLeaderboardWithEntries returns the created leaderboard which is initially active
				// We store this initial leaderboard object to reference its state before the update.
				// Assuming SetupLeaderboardWithEntries sets both UpdateID and RoundID to the provided roundID.
				initialLeaderboard := testutils.SetupLeaderboardWithEntries(t, deps.DB, initial, true, sharedtypes.RoundID(uuid.New()))
				// Corrected logging to use UpdateID
				t.Logf("SetupFn: Initial Leaderboard ID: %d, UpdateID: %s, IsActive: %t",
					initialLeaderboard.ID, initialLeaderboard.UpdateID, initialLeaderboard.IsActive)
				return initialLeaderboard
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				sorted := []string{
					fmt.Sprintf("3:%s", users[2].UserID), // User 2 should get tag 3
					fmt.Sprintf("1:%s", users[0].UserID), // User 0 should get tag 1
					fmt.Sprintf("2:%s", users[1].UserID), // User 1 should get tag 2
				}
				// The RoundID in the payload should be the identifier for the NEW leaderboard state.
				payload := leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               sharedtypes.RoundID(uuid.New()),
					SortedParticipantTags: sorted,
					// Assuming UpdateID in payload is not directly used to identify the new LB in DB
					// but rather RoundID is used as the UpdateID in the DB.
					// If UpdateID in payload is used, adjust logic below.
					// UpdateID:              uuid.New().String(),
				}
				payloadBytes, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				t.Logf("PublishMsgFn: Publishing message with RoundID: %s, SortedParticipantTags: %v",
					payload.RoundID, payload.SortedParticipantTags)
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
					t.Fatalf("failed publishing message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				msgs := received[leaderboardevents.LeaderboardUpdated]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 success message, got %d", len(msgs))
				}
				var req leaderboardevents.LeaderboardUpdateRequestedPayload
				if err := json.Unmarshal(incoming.Payload, &req); err != nil {
					t.Fatalf("Invalid request payload: %v", err)
				}
				// Validate the payload of the success message
				_ = validateLeaderboardUpdatedPayload(t, req, msgs[0])

				// DB assertions
				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), deps.DB)
				if err != nil {
					t.Fatalf("DB query failed: %v", err)
				}
				t.Logf("ValidateFn: Retrieved %d leaderboards from DB", len(leaderboards))
				for i, lb := range leaderboards {
					// Corrected logging to use UpdateID
					t.Logf("ValidateFn: DB Leaderboard %d: ID: %d, UpdateID: %s, IsActive: %t, Data: %+v",
						i, lb.ID, lb.UpdateID, lb.IsActive, lb.LeaderboardData)
				}

				// We expect two leaderboards: the initial one (now inactive) and the new one (active).
				if len(leaderboards) != 2 {
					t.Fatalf("Expected 2 leaderboards (1 old + 1 new), got %d", len(leaderboards))
				}

				// Explicitly find old and new leaderboards using UpdateID and IsActive
				var oldLB, newLB *leaderboarddb.Leaderboard
				for i := range leaderboards {
					lb := &leaderboards[i] // Use pointer to avoid copying
					if lb.IsActive {
						// The new active leaderboard should have the UpdateID matching the RoundID from the incoming request
						if lb.UpdateID == req.RoundID {
							newLB = lb
						}
					} else {
						// The old inactive leaderboard should have the UpdateID of the initial leaderboard
						if lb.UpdateID == initial.UpdateID {
							oldLB = lb
						}
					}
				}

				if oldLB == nil {
					// Corrected logging to use UpdateID
					t.Fatalf("Could not find the old inactive leaderboard with UpdateID %s", initial.UpdateID)
				}
				if newLB == nil {
					// Corrected logging to use UpdateID
					t.Fatalf("Could not find the new active leaderboard with UpdateID %s", req.RoundID)
				}

				// Corrected logging to use UpdateID
				t.Logf("ValidateFn: Identified Old Leaderboard ID: %d, UpdateID: %s, IsActive: %t", oldLB.ID, oldLB.UpdateID, oldLB.IsActive)
				// Corrected logging to use UpdateID
				t.Logf("ValidateFn: Identified New Leaderboard ID: %d, UpdateID: %s, IsActive: %t", newLB.ID, newLB.UpdateID, newLB.IsActive)

				// --- Add logging for raw LeaderboardData from the new leaderboard ---
				t.Logf("ValidateFn: Raw New Leaderboard Data (newLB.LeaderboardData): %+v", newLB.LeaderboardData)
				// --- End added logging ---

				// Assert that the old leaderboard is indeed inactive
				if oldLB.IsActive {
					t.Errorf("Old leaderboard should be inactive")
				}
				// Assert that the new leaderboard is indeed active
				if !newLB.IsActive {
					t.Errorf("New leaderboard should be active")
				}

				// Compare the data of the NEW leaderboard using maps
				expectedDataMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
				// --- UPDATED LOGIC HERE ---
				for _, entryStr := range req.SortedParticipantTags {
					parts := strings.Split(entryStr, ":")
					if len(parts) == 2 {
						tag, err := strconv.Atoi(parts[0])
						if err != nil {
							t.Fatalf("Failed to parse tag number from sorted participant tag string '%s': %v", entryStr, err)
						}
						userID := sharedtypes.DiscordID(parts[1])
						expectedDataMap[userID] = sharedtypes.TagNumber(tag)
					} else {
						t.Fatalf("Invalid format in sorted participant tags: '%s'", entryStr)
					}
				}
				// --- END UPDATED LOGIC ---

				t.Logf("ValidateFn: Expected Data Map: %+v", expectedDataMap)

				actualDataMap := testutils.ExtractLeaderboardDataMap(newLB.LeaderboardData)
				t.Logf("ValidateFn: Actual Data Map (from newLB): %+v", actualDataMap)

				// Check if the number of entries match
				if len(expectedDataMap) != len(actualDataMap) {
					t.Errorf("Leaderboard entry count mismatch: expected %d, got %d", len(expectedDataMap), len(actualDataMap))
				}

				// Check if each user has the expected tag in the actual data
				for userID, expectedTag := range expectedDataMap {
					actualTag, ok := actualDataMap[userID]
					if !ok {
						t.Errorf("User %s not found in new leaderboard data", userID)
					} else if actualTag != expectedTag {
						t.Errorf("Tag mismatch for user %s: expected %d, got %d", userID, expectedTag, actualTag)
					}
				}
			},
			expectedOutgoingTopics: []string{leaderboardevents.LeaderboardUpdated},
			expectHandlerError:     false,
		},
		{
			name:  "Failure - Invalid JSON Payload",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				entries := []leaderboardtypes.LeaderboardEntry{
					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: 99},
				}
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, entries, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("not-a-json"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received[leaderboardevents.LeaderboardUpdated]) > 0 {
					t.Errorf("Unexpected success message published")
				}
				if len(received[leaderboardevents.LeaderboardUpdateFailed]) > 0 {
					t.Errorf("Unexpected failure message published")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
		{
			name:  "Failure - Empty SortedParticipantTags",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				payload := leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               sharedtypes.RoundID(uuid.New()),
					SortedParticipantTags: []string{},
				}
				b, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received[leaderboardevents.LeaderboardUpdated]) > 0 {
					t.Errorf("Expected no success messages, but found some")
				}
				if len(received[leaderboardevents.LeaderboardUpdateFailed]) > 0 {
					t.Errorf("Unexpected failure message published")
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
		},
		{
			name:  "Failure - Incorrect Tag Format",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *leaderboarddb.Leaderboard {
				return testutils.SetupLeaderboardWithEntries(t, deps.DB, []leaderboardtypes.LeaderboardEntry{}, true, sharedtypes.RoundID(uuid.New()))
			},
			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, users []testutils.User) *message.Message {
				payload := leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID: sharedtypes.RoundID(uuid.New()),
					SortedParticipantTags: []string{
						"user_no_tag",
					},
				}
				b, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), leaderboardevents.LeaderboardUpdateRequested, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incoming *message.Message, received map[string][]*message.Message, initial *leaderboarddb.Leaderboard) {
				if len(received[leaderboardevents.LeaderboardUpdated]) > 0 {
					t.Errorf("Unexpected success message")
				}
				if len(received[leaderboardevents.LeaderboardUpdateFailed]) > 0 {
					t.Errorf("Unexpected failure message")
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
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, incoming, received, initialState.(*leaderboarddb.Leaderboard))
				},
				ExpectError: tc.expectHandlerError,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
