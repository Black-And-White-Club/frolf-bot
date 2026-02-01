package roundintegrationtests

import (
	"context"
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

func TestScheduleRoundEvents(t *testing.T) {
	tests := []struct {
		name           string
		setupTestEnv   func(ctx context.Context, deps RoundTestDeps) *roundtypes.ScheduleRoundEventsRequest
		validateResult func(t *testing.T, res results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error], roundID sharedtypes.RoundID)
	}{
		{
			name: "Successfully schedule round events",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.ScheduleRoundEventsRequest {
				roundID := sharedtypes.RoundID(uuid.New())
				startTime := sharedtypes.StartTime(time.Now().Add(24 * time.Hour).UTC())

				return &roundtypes.ScheduleRoundEventsRequest{
					GuildID:        "test-guild",
					RoundID:        roundID,
					Title:          "Scheduled Round",
					StartTime:      startTime,
					Location:       "Test Location",
					EventMessageID: "msg123",
				}
			},
			validateResult: func(t *testing.T, res results.OperationResult[*roundtypes.ScheduleRoundEventsResult, error], roundID sharedtypes.RoundID) {
				if res.Success == nil {
					t.Fatalf("Expected success, got failure: %+v", res.Failure)
				}
				success := *res.Success
				if success.RoundID != roundID {
					t.Errorf("Expected round ID %s, got %s", roundID, success.RoundID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.ScheduleRoundEvents(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, req.RoundID)
			}
		})
	}
}
