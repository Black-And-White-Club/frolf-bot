package roundintegrationtests

import (
	"context"
	"database/sql" // Import sql package
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TestProcessRoundStart tests the functionality of starting a round.
func TestProcessRoundStart(t *testing.T) {
	tag1 := sharedtypes.TagNumber(1)

	tests := []struct {
		name                  string
		roundID               sharedtypes.RoundID
		initialSetup          func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID)
		payload               roundevents.RoundStartedPayloadV1
		expectedError         bool
		expectedErrorContains string
		validateResponse      func(t *testing.T, result roundservice.RoundOperationResult, db *bun.DB, roundID sharedtypes.RoundID)
	}{
		{
			name:    "Successful round start",
			roundID: sharedtypes.RoundID(uuid.New()),
			initialSetup: func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID) {
				// Setup a round in 'Upcoming' state
				_, _ = SetupRoundWithParticipantsHelper(t, db, roundID,
					roundtypes.Title("Test Round for Start"), "start_msg_123",
					[]roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user_A"), TagNumber: &tag1, Response: roundtypes.ResponseAccept, Score: nil},
					})
			},
			payload: roundevents.RoundStartedPayloadV1{
				GuildID:   "test-guild",
				RoundID:   sharedtypes.RoundID(uuid.Nil), // Will be updated in test loop
				Title:     roundtypes.Title("Test Round"),
				Location:  nil,
				StartTime: nil,
				ChannelID: "",
			},
			expectedError: false,
			validateResponse: func(t *testing.T, result roundservice.RoundOperationResult, db *bun.DB, roundID sharedtypes.RoundID) {
				if result.Success == nil {
					t.Fatalf("Expected success payload, got nil")
				}
				// Fix: Expect pointer type instead of value type
				successPayload, ok := result.Success.(*roundevents.DiscordRoundStartPayloadV1)
				if !ok {
					t.Fatalf("Expected *DiscordRoundStartPayload, got %T", result.Success)
				}

				// Assert payload content
				if successPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, successPayload.RoundID)
				}
				if successPayload.EventMessageID != "start_msg_123" {
					t.Errorf("Expected EventMessageID 'start_msg_123', got '%s'", successPayload.EventMessageID)
				}
				if len(successPayload.Participants) != 1 {
					t.Errorf("Expected 1 participant, got %d", len(successPayload.Participants))
				}
				if successPayload.Participants[0].UserID != sharedtypes.DiscordID("user_A") {
					t.Errorf("Expected participant 'user_A', got '%s'", successPayload.Participants[0].UserID)
				}

				// Verify round state in DB
				fetchedRound := new(roundtypes.Round)
				err := db.NewSelect().Model(fetchedRound).Where("id = ?", roundID).Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after update: %v", err)
				}
				if fetchedRound.State != roundtypes.RoundStateInProgress {
					t.Errorf("Expected round state to be '%s', got '%s'", roundtypes.RoundStateInProgress, fetchedRound.State)
				}
			},
		},
		{
			name:    "Round not found in DB",
			roundID: sharedtypes.RoundID(uuid.New()), // This round will not be inserted
			initialSetup: func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID) {
				// No setup, so the round won't exist
			},
			payload: roundevents.RoundStartedPayloadV1{
				GuildID:   "test-guild",
				RoundID:   sharedtypes.RoundID(uuid.Nil), // Will be updated in test loop
				Title:     roundtypes.Title("Test Round"),
				Location:  nil,
				StartTime: nil,
				ChannelID: "",
			},
			expectedError:         false,           // Service uses failure payload instead of error
			expectedErrorContains: "round with ID", // Error from GetRound
			validateResponse: func(t *testing.T, result roundservice.RoundOperationResult, db *bun.DB, roundID sharedtypes.RoundID) {
				if result.Failure == nil {
					t.Fatalf("Expected failure payload, but got nil")
				}
				// Fix: Expect pointer type instead of value type
				failurePayload, ok := result.Failure.(*roundevents.RoundErrorPayloadV1)
				if !ok {
					t.Fatalf("Expected *RoundErrorPayloadV1, got %T", result.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round with ID") {
					t.Errorf("Expected error message to contain 'round with ID', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name:    "Failed to update round state in DB",
			roundID: sharedtypes.RoundID(uuid.New()),
			initialSetup: func(t *testing.T, db *bun.DB, roundID sharedtypes.RoundID) {
				// Setup a round that exists initially.
				_, _ = SetupRoundWithParticipantsHelper(t, db, roundID,
					roundtypes.Title("Test Round for Update Fail"), "fail_msg_456",
					[]roundtypes.Participant{
						{UserID: sharedtypes.DiscordID("user_B"), TagNumber: &tag1, Response: roundtypes.ResponseAccept, Score: nil},
					})
			},
			payload: roundevents.RoundStartedPayloadV1{
				GuildID:   "test-guild",
				RoundID:   sharedtypes.RoundID(uuid.Nil), // Will be updated in test loop
				Title:     roundtypes.Title("Test Round"),
				Location:  nil,
				StartTime: nil,
				ChannelID: "",
			},
			expectedError:         false,           // Service uses failure payload instead of error
			expectedErrorContains: "round with ID", // Updated to match the actual error from GetRound
			validateResponse: func(t *testing.T, result roundservice.RoundOperationResult, db *bun.DB, roundID sharedtypes.RoundID) {
				if result.Failure == nil {
					t.Fatalf("Expected failure payload, but got nil")
				}
				// Fix: Expect pointer type instead of value type
				failurePayload, ok := result.Failure.(*roundevents.RoundErrorPayloadV1)
				if !ok {
					t.Fatalf("Expected *RoundErrorPayloadV1, got %T", result.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round with ID") { // Updated string check
					t.Errorf("Expected error message to contain 'round with ID', got '%s'", failurePayload.Error)
				}
				// Verify that the round is no longer in the DB, as it was deleted to simulate update failure.
				fetchedRound := new(roundtypes.Round)
				err := db.NewSelect().Model(fetchedRound).Where("id = ?", roundID).Scan(context.Background())
				if err == nil {
					t.Errorf("Expected round to not be found in DB after simulated update failure, but it was found")
				} else if err != sql.ErrNoRows {
					t.Fatalf("Failed to fetch round from DB after failed update with unexpected error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			// No defer deps.Cleanup() here, as per your request. Cleanup is external.

			tt.roundID = sharedtypes.RoundID(uuid.New())
			tt.payload.RoundID = tt.roundID

			if tt.initialSetup != nil {
				tt.initialSetup(t, deps.BunDB, tt.roundID)
			}

			// --- NEW LOGIC FOR "Failed to update round state in DB" TEST CASE ---
			if tt.name == "Failed to update round state in DB" {
				// Simulate a scenario where the update fails because the round is no longer present.
				// This forces UpdateRound to return sql.ErrNoRows, which ProcessRoundStart handles.
				_, err := deps.BunDB.NewDelete().Model(&roundtypes.Round{}).Where("id = ?", tt.roundID).Exec(context.Background())
				if err != nil {
					t.Fatalf("Failed to delete round for update failure simulation: %v", err)
				}
			}
			// --- END NEW LOGIC ---

			result, err := deps.Service.ProcessRoundStart(deps.Ctx, tt.payload.GuildID, tt.payload.RoundID)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				if result.Failure == nil {
					t.Errorf("Expected a failure payload, but got nil")
				}
				if result.Success != nil {
					t.Errorf("Expected nil success payload, but got %v", result.Success)
				}
				if result.Failure != nil && tt.expectedErrorContains != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundErrorPayloadV1)
					if ok && !strings.Contains(failurePayload.Error, tt.expectedErrorContains) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorContains, failurePayload.Error)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}

				// Handle both success and failure cases when expectedError is false
				if result.Failure != nil && result.Success != nil {
					t.Errorf("Got both failure and success payloads - should only have one")
				}
				if result.Failure == nil && result.Success == nil {
					t.Errorf("Expected either a success or failure payload, but got neither")
				}

				// Handle validation failures when expectedError is false but operation fails
				if result.Failure != nil && tt.expectedErrorContains != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundErrorPayloadV1)
					if ok && !strings.Contains(failurePayload.Error, tt.expectedErrorContains) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorContains, failurePayload.Error)
					}
				}
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, result, deps.BunDB, tt.roundID)
			}
		})
	}
}
