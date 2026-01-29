package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestValidateRoundUpdate(t *testing.T) {
	testUserID := sharedtypes.DiscordID("user123")

	tests := []struct {
		name                     string
		payload                  *roundtypes.UpdateRoundRequest
		expectedErrorMessagePart string
	}{
		{
			name: "Valid update request - Title only",
			payload: &roundtypes.UpdateRoundRequest{
				GuildID:  "test-guild",
				RoundID:  sharedtypes.RoundID(uuid.New()),
				UserID:   testUserID,
				Title:    stringPtr("New Title"),
				Timezone: stringPtr("America/New_York"),
			},
		},
		{
			name: "Invalid update request - Zero RoundID",
			payload: &roundtypes.UpdateRoundRequest{
				GuildID:  "test-guild",
				RoundID:  sharedtypes.RoundID(uuid.Nil),
				UserID:   testUserID,
				Title:    stringPtr("New Title"),
				Timezone: stringPtr("America/New_York"),
			},
			expectedErrorMessagePart: "round ID cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			result, err := deps.Service.ValidateRoundUpdate(deps.Ctx, tt.payload, roundtime.NewTimeParser())
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedErrorMessagePart != "" {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failureErr := *result.Failure
				if !strings.Contains(failureErr.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMessagePart, failureErr.Error())
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
			}
		})
	}
}

func TestUpdateRoundEntity(t *testing.T) {
	testUserID := sharedtypes.DiscordID("user123")
	testGuildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.UpdateRoundRequest
		expectedErrorMessagePart string
	}{
		{
			name: "Successful update of title",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.UpdateRoundRequest {
				roundID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Original Title",
					State: roundtypes.RoundStateUpcoming,
				})
				round.GuildID = testGuildID

				err := deps.DB.CreateRound(ctx, deps.BunDB, testGuildID, &round)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB: %v", err)
				}

				return &roundtypes.UpdateRoundRequest{
					GuildID: testGuildID,
					RoundID: roundID,
					Title:   stringPtr("Updated Title"),
					UserID:  testUserID,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			payload := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.UpdateRoundEntity(deps.Ctx, payload)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectedErrorMessagePart != "" {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got nil: %+v", result.Failure)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
