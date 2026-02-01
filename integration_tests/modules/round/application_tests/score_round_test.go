package roundintegrationtests

import (
	"strings"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestValidateScoreUpdateRequest(t *testing.T) {
	tests := []struct {
		name                  string
		payload               *roundtypes.ScoreUpdateRequest
		expectedErrorContains string
	}{
		{
			name: "Valid score update request",
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: "test-guild",
				RoundID: sharedtypes.RoundID(uuid.New()),
				UserID:  "123456789",
				Score:   func() *sharedtypes.Score { s := sharedtypes.Score(72); return &s }(),
			},
			expectedErrorContains: "",
		},
		{
			name: "Invalid request - zero round ID",
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: "test-guild",
				RoundID: sharedtypes.RoundID(uuid.Nil),
				UserID:  "123456789",
				Score:   func() *sharedtypes.Score { s := sharedtypes.Score(72); return &s }(),
			},
			expectedErrorContains: "round ID cannot be zero",
		},
		{
			name: "Invalid request - empty participant",
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: "test-guild",
				RoundID: sharedtypes.RoundID(uuid.New()),
				UserID:  "",
				Score:   func() *sharedtypes.Score { s := sharedtypes.Score(72); return &s }(),
			},
			expectedErrorContains: "participant Discord ID cannot be empty",
		},
		{
			name: "Invalid request - nil score",
			payload: &roundtypes.ScoreUpdateRequest{
				GuildID: "test-guild",
				RoundID: sharedtypes.RoundID(uuid.New()),
				UserID:  "123456789",
				Score:   nil,
			},
			expectedErrorContains: "score cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			result, err := deps.Service.ValidateScoreUpdateRequest(deps.Ctx, tt.payload)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedErrorContains != "" {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failureErr := *result.Failure
				if !strings.Contains(failureErr.Error(), tt.expectedErrorContains) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorContains, failureErr.Error())
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got nil: %+v", result.Failure)
				}
				success := *result.Success
				if success.UserID != tt.payload.UserID {
					t.Errorf("Expected UserID %s, got %s", tt.payload.UserID, success.UserID)
				}
			}
		})
	}
}

func TestUpdateParticipantScore(t *testing.T) {
	score72 := sharedtypes.Score(72)

	tests := []struct {
		name             string
		initialSetup     func(t *testing.T, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.ScoreUpdateRequest)
		expectedFailure  bool
		validateResponse func(t *testing.T, result roundservice.ScoreUpdateResult, roundID sharedtypes.RoundID)
	}{
		{
			name: "Successful score update",
			initialSetup: func(t *testing.T, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.ScoreUpdateRequest) {
				roundID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Test Round",
					State: roundtypes.RoundStateUpcoming,
				})
				round.Participants = []roundtypes.Participant{
					{UserID: "123456789", Response: roundtypes.ResponseAccept},
				}
				round.GuildID = "test-guild"
				round.EventMessageID = "msg123"

				err := deps.DB.CreateRound(deps.Ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to setup round: %v", err)
				}

				return roundID, &roundtypes.ScoreUpdateRequest{
					GuildID: "test-guild",
					RoundID: roundID,
					UserID:  "123456789",
					Score:   &score72,
				}
			},
			expectedFailure: false,
			validateResponse: func(t *testing.T, result roundservice.ScoreUpdateResult, roundID sharedtypes.RoundID) {
				if result.Success == nil {
					t.Fatalf("Expected success payload, got failure: %+v", result.Failure)
				}
				success := *result.Success
				if success.RoundID != roundID {
					t.Errorf("Expected roundID %s, got %s", roundID, success.RoundID)
				}
				found := false
				for _, p := range success.UpdatedParticipants {
					if p.UserID == "123456789" {
						found = true
						if p.Score == nil || *p.Score != score72 {
							t.Errorf("Expected score 72, got %v", p.Score)
						}
					}
				}
				if !found {
					t.Errorf("Updated participant not found in result")
				}
			},
		},
		{
			name: "Failure update for non-existent round",
			initialSetup: func(t *testing.T, deps RoundTestDeps) (sharedtypes.RoundID, *roundtypes.ScoreUpdateRequest) {
				roundID := sharedtypes.RoundID(uuid.New())
				return roundID, &roundtypes.ScoreUpdateRequest{
					GuildID: "test-guild",
					RoundID: roundID,
					UserID:  "nonexistent",
					Score:   &score72,
				}
			},
			expectedFailure: true,
			validateResponse: func(t *testing.T, result roundservice.ScoreUpdateResult, roundID sharedtypes.RoundID) {
				if result.Failure == nil {
					t.Fatalf("Expected failure payload, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			roundID, payload := tt.initialSetup(t, deps)

			result, err := deps.Service.UpdateParticipantScore(deps.Ctx, payload)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.validateResponse != nil {
				tt.validateResponse(t, result, roundID)
			}
		})
	}
}
