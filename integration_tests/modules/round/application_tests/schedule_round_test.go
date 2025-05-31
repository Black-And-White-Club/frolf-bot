package roundintegrationtests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
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
				futureTime := sharedtypes.StartTime(time.Now().UTC().Add(1 * time.Second))

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
					case <-time.After(2 * time.Second): // Wait a bit longer than expected start time
						reminderNotReceived <- true // Confirmed no reminder received
					}
				}()

				timeout := 2 * time.Second // A bit longer than the scheduled start time

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
		{
			// This test case verifies that the reminder *does* fire when scheduled far enough in the future.
			// The start event is also scheduled, but its delivery is not waited for in this test's short timeout.
			name: "Verify delayed messages are published to stream with correct info",
			setupRound: func() (roundevents.RoundScheduledPayload, string) {
				// Set StartTime far enough in the future for the 1-hour reminder to be scheduled.
				// This makes the reminder fire in ~2 seconds, while the start event is ~1 hour later.
				futureTime := sharedtypes.StartTime(time.Now().UTC().Add(1 * time.Hour).Add(2 * time.Second))

				payload := roundevents.RoundScheduledPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     sharedtypes.RoundID(uuid.New()),
						Title:       "Test Round - Long Future",
						Description: testutils.RoundDescriptionPtr("Integration test - long duration"),
						Location:    testutils.RoundLocationPtr("Test Course"),
						StartTime:   &futureTime,
					},
				}

				return payload, "discord_msg_789"
			},
			expectedError: false,
			waitAndVerify: func(t *testing.T, ctx context.Context, deps RoundTestDeps, payload roundevents.RoundScheduledPayload) {
				// Use deps.JetStreamContext directly
				js := deps.JetStreamContext
				streamName := "delayed"
				consumerName := fmt.Sprintf("test-pull-consumer-%s", uuid.New().String()) // Unique consumer for this test run

				// Create a pull consumer for the delayed stream
				cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
					Durable:       consumerName,
					FilterSubject: fmt.Sprintf("delayed.message.%s", payload.RoundID), // Filter for messages related to this round
					AckPolicy:     jetstream.AckExplicitPolicy,
					DeliverPolicy: jetstream.DeliverAllPolicy, // Start from the beginning of the stream
				})
				if err != nil {
					t.Fatalf("Failed to create pull consumer: %v", err)
				}
				defer func() {
					// Clean up the consumer after the test
					if err := js.DeleteConsumer(ctx, streamName, consumerName); err != nil {
						t.Logf("Failed to delete consumer %s: %v", consumerName, err)
					}
				}()

				// Fetch messages from the stream
				// We expect two messages: one reminder, one start event
				expectedMessages := map[string]bool{
					string(roundevents.RoundReminder): false,
					string(roundevents.RoundStarted):  false,
				}

				messagesReceived := 0
				_, pullCancel := context.WithTimeout(ctx, 5*time.Second) // Short timeout for pulling
				defer pullCancel()

				iter, err := cons.Fetch(2, jetstream.FetchMaxWait(1*time.Second)) // Fetch up to 2 messages, wait up to 1 second
				if err != nil {
					t.Fatalf("Failed to fetch messages: %v", err)
				}

				for msg := range iter.Messages() {
					messagesReceived++
					headers := msg.Headers()

					originalSubject := headers.Get("Original-Subject")
					roundIDHeader := headers.Get("Round-ID")
					executeAtStr := headers.Get("Execute-At")

					if roundIDHeader != (payload.RoundID.String()) {
						t.Errorf("Message Round-ID mismatch for subject %s: expected %s, got %s", originalSubject, payload.RoundID, roundIDHeader)
					}

					// Verify Execute-At timestamp
					executeAt, err := time.Parse(time.RFC3339, executeAtStr)
					if err != nil {
						t.Errorf("Failed to parse Execute-At header '%s': %v", executeAtStr, err)
					}

					// Acknowledge the message so it's not redelivered to this consumer
					if err := msg.Ack(); err != nil {
						t.Logf("Failed to ack message: %v", err)
					}

					switch originalSubject {
					case string(roundevents.RoundReminder):
						expectedReminderTime := payload.StartTime.AsTime().Add(-1 * time.Hour)
						// Allow a small delta for time comparisons
						if !executeAt.Before(expectedReminderTime.Add(1*time.Second)) || !executeAt.After(expectedReminderTime.Add(-1*time.Second)) {
							t.Errorf("Reminder Execute-At mismatch: expected around %s, got %s", expectedReminderTime, executeAt)
						}
						var reminderPayload roundevents.DiscordReminderPayload
						// Corrected: Call msg.Data() to get the []byte slice
						if err := json.Unmarshal(msg.Data(), &reminderPayload); err != nil {
							t.Errorf("Failed to unmarshal reminder payload: %v", err)
						}
						if reminderPayload.RoundID != payload.RoundID {
							t.Errorf("Reminder payload RoundID mismatch: expected %s, got %s", payload.RoundID, reminderPayload.RoundID)
						}
						if reminderPayload.ReminderType != "1h" {
							t.Errorf("Reminder payload ReminderType mismatch: expected %q, got %q", "1h", reminderPayload.ReminderType)
						}
						expectedMessages[string(roundevents.RoundReminder)] = true

					case string(roundevents.RoundStarted):
						expectedStartTime := payload.StartTime.AsTime()
						if !executeAt.Before(expectedStartTime.Add(1*time.Second)) || !executeAt.After(expectedStartTime.Add(-1*time.Second)) {
							t.Errorf("Start event Execute-At mismatch: expected around %s, got %s", expectedStartTime, executeAt)
						}
						var startPayload roundevents.DiscordRoundStartPayload
						// Corrected: Call msg.Data() to get the []byte slice
						if err := json.Unmarshal(msg.Data(), &startPayload); err != nil {
							t.Errorf("Failed to unmarshal start payload: %v", err)
						}
						if startPayload.RoundID != payload.RoundID {
							t.Errorf("Start event payload RoundID mismatch: expected %s, got %s", payload.RoundID, startPayload.RoundID)
						}
						expectedMessages[string(roundevents.RoundStarted)] = true

					default:
						t.Errorf("Unexpected original subject in delayed message: %s", originalSubject)
					}
				}

				if err := iter.Error(); err != nil && err != context.DeadlineExceeded {
					t.Errorf("Error during message iteration: %v", err)
				}

				if messagesReceived != 2 {
					t.Errorf("Expected 2 delayed messages, but received %d", messagesReceived)
				}

				if !expectedMessages[string(roundevents.RoundReminder)] {
					t.Errorf("Did not receive expected delayed reminder message")
				}
				if !expectedMessages[string(roundevents.RoundStarted)] {
					t.Errorf("Did not receive expected delayed start message")
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

			// Verify the immediate result
			scheduledPayload, ok := result.Success.(*roundevents.RoundScheduledPayload)
			if !ok {
				t.Errorf("Expected result to be RoundScheduledPayload, got %T", result.Success)
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
