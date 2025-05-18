package testutils

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/ThreeDotsLabs/watermill/message"
)

func WaitFor(timeout, interval time.Duration, check func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := check(); err == nil {
				return nil
			}
			return fmt.Errorf("timed out waiting: %w", ctx.Err())
		case <-ticker.C:
			if err := check(); err == nil {
				return nil
			}
		}
	}
}

// WaitForMessages waits for a specific number of messages on a topic
func (env *TestEnvironment) WaitForMessages(receivedMsgs map[string][]*message.Message, receivedMsgsMutex *sync.Mutex, topic string, expectedCount int, timeout time.Duration) []*message.Message {
	env.T.Helper() // Marks this function as a test helper

	// Create a context with timeout based on the environment's context
	ctx, cancel := context.WithTimeout(env.Ctx, timeout)
	defer cancel()

	env.T.Logf("Waiting for %d messages on topic %q with timeout %v", expectedCount, topic, timeout)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context timeout or cancellation
			receivedMsgsMutex.Lock()
			finalMsgs, exists := receivedMsgs[topic]
			finalCount := 0
			if exists {
				finalCount = len(finalMsgs)
			}
			receivedMsgsMutex.Unlock()

			env.T.Fatalf("Timeout waiting for %d messages on topic %q. Received %d. Error: %v",
				expectedCount, topic, finalCount, ctx.Err())
			return nil // unreachable due to t.Fatalf

		case <-ticker.C:
			receivedMsgsMutex.Lock()
			msgs, exists := receivedMsgs[topic]
			currentCount := 0
			if exists {
				currentCount = len(msgs)
			}

			if currentCount >= expectedCount {
				result := make([]*message.Message, len(msgs))
				copy(result, msgs)
				receivedMsgsMutex.Unlock()
				env.T.Logf("Successfully received %d messages on topic %q", currentCount, topic)
				return result
			}

			receivedMsgsMutex.Unlock()
			env.T.Logf("Current count for topic %q: %d (expected: %d)", topic, currentCount, expectedCount)
		}
	}
}

// PublishMessage sends a message to the event bus
func PublishMessage(t *testing.T, eventBus eventbus.EventBus, ctx context.Context, topic string, msg *message.Message) error {
	t.Helper() // Mark this as a helper function
	return eventBus.Publish(topic, msg)
}
