package roundintegrationtests

import (
	"context"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestValidateRoundDeletion(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.DeleteRoundInput
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, returnedResult results.OperationResult[*roundtypes.Round, error])
	}{
		{
			name: "Valid deletion request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.DeleteRoundInput {
				generator := testutils.NewTestDataGenerator()
				creatorID := testutils.DiscordID("test-creator-123")
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: creatorID,
					Title:     "Round to delete",
					State:     roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}
				return &roundtypes.DeleteRoundInput{
					RoundID: round.ID,
					GuildID: "test-guild",
					UserID:  round.CreatedBy,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, returnedResult results.OperationResult[*roundtypes.Round, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				round := *returnedResult.Success
				if round.ID == (sharedtypes.RoundID{}) {
					t.Errorf("Expected valid Round in success result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			payload := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.ValidateRoundDeletion(deps.Ctx, payload)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				}
			} else {
				if result.Success == nil {
					t.Errorf("Expected success result, but got none")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestDeleteRound(t *testing.T) {
	tests := []struct {
		name           string
		setupTestEnv   func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.DeleteRoundInput)
		validateResult func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[bool, error], roundID sharedtypes.RoundID)
	}{
		{
			name: "Successfully delete a round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.DeleteRoundInput) {
				generator := testutils.NewTestDataGenerator()
				creatorID := testutils.DiscordID("test-creator-456")
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: creatorID,
					Title:     "Round to delete",
					State:     roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}
				return round.ID, &roundtypes.DeleteRoundInput{
					RoundID: round.ID,
					GuildID: "test-guild",
					UserID:  round.CreatedBy,
				}
			},
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[bool, error], roundID sharedtypes.RoundID) {
				if returnedResult.Success == nil || !*returnedResult.Success {
					t.Errorf("Expected success to be true")
				}
				// Verify DB state - should be soft deleted (state = DELETED)
				round, err := deps.DB.GetRound(ctx, deps.BunDB, "test-guild", roundID)
				if err != nil {
					t.Errorf("Expected round to still exist (soft delete), got error: %v", err)
				} else if round.State != roundtypes.RoundStateDeleted {
					t.Errorf("Expected round state to be DELETED, got: %v", round.State)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			roundID, payload := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.DeleteRound(deps.Ctx, payload)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result, roundID)
			}
		})
	}
}
