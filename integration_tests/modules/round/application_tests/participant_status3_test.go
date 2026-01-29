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

func TestUpdateParticipantStatus(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest)
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.Round, error])
	}{
		{
			name: "Status and tag number set after lookup",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_1"),
					Title:     "Round for direct update",
					State:     roundtypes.RoundStateUpcoming,
				})
				tagNum1 := sharedtypes.TagNumber(1)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("existing_participant"),
						TagNumber: &tagNum1,
						Response:  roundtypes.ResponseTentative,
					},
				}
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				tag := sharedtypes.TagNumber(123)
				joinedLate := false
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:    "test-guild",
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("existing_participant"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tag,
					JoinedLate: &joinedLate,
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
						if *p.TagNumber != 123 {
							t.Errorf("Expected TagNumber to be 123, got %d", *p.TagNumber)
						}
						if p.Response != roundtypes.ResponseAccept {
							t.Errorf("Expected Response to be Accept, got %s", p.Response)
						}
						break
					}
				}
				if !found {
					t.Errorf("Participant 'existing_participant' not found in returned round")
				}
			},
		},
		{
			name: "Participant accepts without TagNumber (adds participant with nil tag)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_3"),
					Title:     "Round for nil tag participant",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return roundForDBInsertion.ID, &roundtypes.JoinRoundRequest{
					GuildID:    "test-guild",
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("participant_needs_tag"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  nil,
					JoinedLate: &joinedLate,
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
					if p.UserID == sharedtypes.DiscordID("participant_needs_tag") {
						found = true
						if p.TagNumber != nil {
							t.Errorf("Expected TagNumber to be nil, got %v", p.TagNumber)
						}
						break
					}
				}
				if !found {
					t.Errorf("Participant 'participant_needs_tag' not found")
				}
			},
		},
		{
			name: "Attempt to update participant in non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.JoinRoundRequest) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return nonExistentID, &roundtypes.JoinRoundRequest{
					GuildID:  "test-guild",
					RoundID:  nonExistentID,
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "failed to update participant in DB",
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
					RoundID:  sharedtypes.RoundID(uuid.New()),
					UserID:   sharedtypes.DiscordID("default_user"),
					Response: roundtypes.ResponseAccept,
				}
			}

			result, err := deps.Service.UpdateParticipantStatus(deps.Ctx, payload)
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
