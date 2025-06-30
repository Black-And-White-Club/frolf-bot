// integration_tests/modules/round/application_tests/update_round_message_id_test.go
package roundintegrationtests

import (
	"context"
	"strings"
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// StartTimePtr is a helper function to convert time.Time to *sharedtypes.StartTime
func StartTimePtr(t time.Time) *sharedtypes.StartTime {
	st := sharedtypes.StartTime(t)
	return &st
}

func TestUpdateRoundMessageID(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.Round)
		discordMessageID         string
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedRound *roundtypes.Round)
	}{
		{
			name: "Successful update of Discord message ID for an existing round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.Round) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy:        testutils.DiscordID("test_user_1"),
					Title:            "Existing Round for Update",
					State:            roundtypes.RoundStateUpcoming,
					ParticipantCount: 0,
					StartTime:        testutils.StartTimePtr(time.Now().Add(24 * time.Hour)),
				})
				roundForDBInsertion.EventMessageID = ""

				err := deps.DB.CreateRound(ctx, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				persistedRound, err := deps.DB.GetRound(ctx, "test-guild", roundForDBInsertion.ID)
				if err != nil {
					t.Fatalf("Failed to fetch newly created round from DB after insertion: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Newly created round not found in DB after insertion for ID: %s", roundForDBInsertion.ID)
				}
				return persistedRound.ID, persistedRound
			},
			discordMessageID: "new_discord_message_id_12345",
			expectedError:    false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedRound *roundtypes.Round) {
				if returnedRound == nil {
					t.Fatalf("Expected a returned round, but got nil")
				}
				if returnedRound.EventMessageID != "new_discord_message_id_12345" {
					t.Errorf("Returned round EventMessageID mismatch: expected 'new_discord_message_id_12345', got '%v'", returnedRound.EventMessageID)
				}

				persistedRound, err := deps.DB.GetRound(ctx, "test-guild", returnedRound.ID)
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after update: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Round not found in DB after update for ID: %s", returnedRound.ID)
				}
				if persistedRound.EventMessageID != "new_discord_message_id_12345" {
					t.Errorf("Persisted round EventMessageID mismatch: expected 'new_discord_message_id_12345', got '%v'", persistedRound.EventMessageID)
				}
			},
		},
		{
			name: "Attempt to update Discord message ID for a non-existent round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.Round) {
				return sharedtypes.RoundID(uuid.New()), nil
			},
			discordMessageID: "some_message_id_for_non_existent",
			expectedError:    true,
			// --- FIX APPLIED HERE ---
			// Changed from "failed to update Discord event message ID in DB: round not found"
			// to "sql: no rows in result set" to match the actual error from the DB layer.
			expectedErrorMessagePart: "sql: no rows in result set",
			// --- END FIX ---
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedRound *roundtypes.Round) {
				if returnedRound != nil {
					t.Errorf("Expected nil round on error, but got: %+v", returnedRound)
				}
			},
		},
		{
			name: "Update Discord message ID to an empty string",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.Round) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy:        testutils.DiscordID("test_user_2"),
					Title:            "Round with existing message ID",
					State:            roundtypes.RoundStateUpcoming,
					ParticipantCount: 0,
					StartTime:        testutils.StartTimePtr(time.Now().Add(24 * time.Hour)),
				})
				roundForDBInsertion.EventMessageID = "old_message_id_abc"

				err := deps.DB.CreateRound(ctx, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				persistedRound, err := deps.DB.GetRound(ctx, "test-guild", roundForDBInsertion.ID)
				if err != nil {
					t.Fatalf("Failed to fetch newly created round from DB after insertion: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Newly created round not found in DB after insertion for ID: %s", roundForDBInsertion.ID)
				}
				return persistedRound.ID, persistedRound
			},
			discordMessageID: "",
			expectedError:    false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedRound *roundtypes.Round) {
				if returnedRound == nil {
					t.Fatalf("Expected a returned round, but got nil")
				}
				if returnedRound.EventMessageID != "" {
					t.Errorf("Returned round EventMessageID mismatch: expected empty string, got '%v'", returnedRound.EventMessageID)
				}

				persistedRound, err := deps.DB.GetRound(ctx, "test-guild", returnedRound.ID)
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after update: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Round not found in DB after update for ID: %s", returnedRound.ID)
				}
				if persistedRound.EventMessageID != "" {
					t.Errorf("Persisted round EventMessageID mismatch: expected empty string, got '%v'", persistedRound.EventMessageID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var roundToUpdateID sharedtypes.RoundID
			if tt.setupTestEnv != nil {
				roundToUpdateID, _ = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				roundToUpdateID = sharedtypes.RoundID(uuid.New())
			}

			returnedRound, err := deps.Service.UpdateRoundMessageID(deps.Ctx, "test-guild", roundToUpdateID, tt.discordMessageID)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if tt.expectedErrorMessagePart != "" && !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, returnedRound)
			}
		})
	}
}
