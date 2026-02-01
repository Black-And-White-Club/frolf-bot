package roundintegrationtests

import (
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestStoreRound(t *testing.T) {
	createValidRound := func() *roundtypes.Round {
		startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour).UTC())
		eventType := roundtypes.EventType("casual")

		return &roundtypes.Round{
			ID:          sharedtypes.RoundID(uuid.New()),
			Title:       "Test Round Title",
			Description: "Test Description",
			Location:    "Test Location",
			EventType:   &eventType,
			StartTime:   &startTime,
			CreatedBy:   "user_123",
			State:       roundtypes.RoundStateUpcoming,
			GuildID:     "test-guild",
		}
	}

	tests := []struct {
		name            string
		setupRound      func() *roundtypes.Round
		expectedFailure bool
	}{
		{
			name:            "Successful round storage",
			setupRound:      createValidRound,
			expectedFailure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			round := tt.setupRound()
			result, err := deps.Service.StoreRound(deps.Ctx, round, "test-guild")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.expectedFailure {
				if result.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
			} else {
				if result.Success == nil {
					t.Fatalf("Expected success result, but got nil: %+v", result.Failure)
				}
				res := *result.Success
				if res.Round.Title != round.Title {
					t.Errorf("Expected title %s, got %s", round.Title, res.Round.Title)
				}
			}
		})
	}
}
