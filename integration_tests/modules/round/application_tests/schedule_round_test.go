package roundintegrationtests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// TestScheduleRoundEvents is the main integration test function for the ScheduleRoundEvents service method.
func TestScheduleRoundEvents_Behavioral(t *testing.T) {
	tests := []struct {
		name          string
		setupRound    func() (roundevents.RoundScheduledPayload, string)
		expectedError bool
		waitAndVerify func(t *testing.T, ctx context.Context, deps RoundTestDeps, payload roundevents.RoundScheduledPayload)
	}{
		{
			name: "Round scheduled 1 second in future - only start event expected (reminder skipped)",
			setupRound: func() (roundevents.RoundScheduledPayload, string) {
				// This case simulates a round starting very soon, so the 1-hour reminder should be skipped.
				// But the start time needs to be at least 30 seconds in the future to be scheduled
				futureTime := sharedtypes.StartTime(time.Now().UTC().Add(35 * time.Second))

				payload := roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:   sharedtypes.RoundID(uuid.New()),
						Title:     "Test Round - Near Future",
						StartTime: &futureTime,
					},
				}

				return payload, "discord_msg_456"
			},
			expectedError: false,
			waitAndVerify: func(t *testing.T, ctx context.Context, deps RoundTestDeps, payload roundevents.RoundScheduledPayload) {
				// Only expect the start event to fire, as the reminder time is in the past.
				startReceived := make(chan bool, 1)
				reminderNotReceived := make(chan bool, 1) // To verify reminder is NOT sent

				go func() {
					subStart, err := deps.EventBus.Subscribe(ctx, string(roundevents.RoundStarted))
					if err != nil {
						t.Errorf("Failed to subscribe to start events: %v", err)
						return
					}
					for msg := range subStart {
						var startPayload roundevents.DiscordRoundStartPayload
						if err := json.Unmarshal(msg.Payload, &startPayload); err == nil {
							if startPayload.RoundID == payload.RoundID {
								t.Logf("Start event received for RoundID: %s", payload.RoundID)
								startReceived <- true
								return
							}
						} else {
							t.Logf("Failed to unmarshal start payload: %v", err)
						}
					}
				}()

				go func() {
					// Listen for reminder, but expect NOT to receive it
					subReminder, err := deps.EventBus.Subscribe(ctx, string(roundevents.RoundReminder))
					if err != nil {
						t.Errorf("Failed to subscribe to reminder events (for negative check): %v", err)
						return
					}
					select {
					case msg := <-subReminder:
						var reminderPayload roundevents.DiscordReminderPayload
						if err := json.Unmarshal(msg.Payload, &reminderPayload); err == nil {
							if reminderPayload.RoundID == payload.RoundID {
								t.Errorf("UNEXPECTED: Reminder event received for RoundID: %s", payload.RoundID)
							}
						}
					case <-time.After(40 * time.Second): // Wait a bit longer than expected start time
						reminderNotReceived <- true // Confirmed no reminder received
					}
				}()

				timeout := 40 * time.Second // A bit longer than the scheduled start time

				select {
				case <-startReceived:
					t.Log("Start event received as expected")
				case <-time.After(timeout):
					t.Errorf("Start event not received within %v", timeout)
				}

				select {
				case <-reminderNotReceived:
					t.Log("Reminder event NOT received as expected (correct behavior)")
				case <-time.After(timeout): // Should ideally not hit this if reminderNotReceived is sent
					t.Log("Timeout waiting for reminder not to be received (might indicate a problem or test timing issue)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t) // Assume this sets up your NATS connection and service
			defer deps.Ctx.Done()            // Ensure context is cancelled for cleanup

			payload, discordMessageID := tt.setupRound()

			// Call the service method
			result, err := deps.Service.ScheduleRoundEvents(deps.Ctx, payload, discordMessageID)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
				return
			}

			if result.Success == nil {
				t.Fatalf("Expected success result, but got nil")
			}

			// Fix: Expect pointer type instead of value type
			scheduledPayload, ok := result.Success.(*roundevents.RoundScheduledPayload)
			if !ok {
				t.Errorf("Expected result to be *RoundScheduledPayload, got %T", result.Success)
				return
			}

			if scheduledPayload.RoundID != payload.RoundID {
				t.Errorf("RoundID mismatch: expected %s, got %s", payload.RoundID, scheduledPayload.RoundID)
			}

			// Run behavioral verification
			if tt.waitAndVerify != nil {
				tt.waitAndVerify(t, deps.Ctx, deps, payload)
			}
		})
	}
}
