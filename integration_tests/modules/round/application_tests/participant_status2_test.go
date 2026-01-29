package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestParticipantRemoval(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest)
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.Round, error])
	}{
		{
			name: "Valid removal of existing participant - Expecting Round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_removal_1"),
					Title:     "Round for participant removal",
					State:     roundtypes.RoundStateUpcoming,
				})

				scoreZero := sharedtypes.Score(0)
				tagNum1 := sharedtypes.TagNumber(1)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("participant_to_remove"),
						TagNumber: &tagNum1,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
					{
						UserID:   sharedtypes.DiscordID("participant_staying"),
						Response: roundtypes.ResponseTentative,
						Score:    &scoreZero,
					},
				}

				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					RoundID: roundForDBInsertion.ID,
					UserID:  sharedtypes.DiscordID("participant_to_remove"),
					GuildID: "test-guild",
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.Round, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				round := *returnedResult.Success

				// Verify participant is removed
				for _, p := range round.Participants {
					if p.UserID == sharedtypes.DiscordID("participant_to_remove") {
						t.Errorf("Participant 'participant_to_remove' should have been removed")
					}
				}
			},
		},
		{
			name: "Attempt to remove non-existent participant - Should succeed with no changes",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_removal_2"),
					Title:     "Round for non-existent participant removal",
					State:     roundtypes.RoundStateInProgress,
				})

				participant := roundtypes.Participant{
					UserID:   sharedtypes.DiscordID("existing_participant"),
					Response: roundtypes.ResponseAccept,
				}
				roundForDBInsertion.Participants = []roundtypes.Participant{participant}

				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					RoundID: roundForDBInsertion.ID,
					UserID:  sharedtypes.DiscordID("non_existent_participant"),
					GuildID: "test-guild",
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.Round, error]) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				round := *returnedResult.Success
				found := false
				for _, p := range round.Participants {
					if p.UserID == sharedtypes.DiscordID("existing_participant") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Existing participant should still be present")
				}
			},
		},
		{
			name: "Attempt to remove participant from non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return nonExistentID, &roundtypes.JoinRoundRequest{
					RoundID: nonExistentID,
					UserID:  sharedtypes.DiscordID("some_user"),
					GuildID: "test-guild",
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to remove participant",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.Round, error]) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload *roundtypes.JoinRoundRequest
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				payload = &roundtypes.JoinRoundRequest{
					RoundID: sharedtypes.RoundID(uuid.New()),
					UserID:  sharedtypes.DiscordID("default_user"),
				}
			}

			result, err := deps.Service.ParticipantRemoval(deps.Ctx, payload)
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					err := *result.Failure
					if !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
					}
				}
			} else {
				if result.Success == nil {
					t.Errorf("Expected success result, but got none")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
