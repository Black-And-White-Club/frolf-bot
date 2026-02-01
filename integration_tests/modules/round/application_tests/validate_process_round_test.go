package roundintegrationtests

import (
	"strings"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

func TestValidateRoundCreation(t *testing.T) {
	tests := []struct {
		name                     string
		payload                  *roundtypes.CreateRoundInput
		expectedFailure          bool
		expectedErrorMessagePart string
	}{
		{
			name: "Valid round creation request",
			payload: &roundtypes.CreateRoundInput{
				Title:       "Test Round",
				Description: "Test Description",
				Location:    "Test Location",
				StartTime:   "tomorrow at 12:00",
				UserID:      "user_123",
				GuildID:     "guild_123",
				Timezone:    "UTC",
				ChannelID:   "channel_123",
			},
			expectedFailure: false,
		},
		{
			name: "Missing title",
			payload: &roundtypes.CreateRoundInput{
				Title:     "",
				StartTime: "tomorrow at 12:00",
				UserID:    "user_123",
				GuildID:   "guild_123",
				Timezone:  "UTC",
				ChannelID: "channel_123",
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			timeParser := roundtime.NewTimeParser()
			clock := &roundutil.RealClock{}

			result, err := deps.Service.ValidateRoundCreationWithClock(deps.Ctx, tt.payload, timeParser, clock)
			if err != nil {
				t.Fatalf("Unexpected error from service: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failureErr := *result.Failure
				if !strings.Contains(failureErr.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMessagePart, failureErr.Error())
				}
			} else {
				if result.Success == nil {
					if result.Failure != nil {
						t.Fatalf("Expected success result, but got failure: %v", *result.Failure)
					}
					t.Fatalf("Expected success result, but got nil Success and nil Failure")
				}
			}
		})
	}
}
