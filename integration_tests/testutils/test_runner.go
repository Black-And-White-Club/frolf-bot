package testutils

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// TestCase represents a generic test case structure for event-driven tests
type TestCase struct {
	// Name of the test case
	Name string
	// SetupFn prepares the test environment and returns any initial state needed
	SetupFn func(t *testing.T, env *TestEnvironment) interface{}
	// PublishMsgFn creates and publishes the message that triggers the test flow
	PublishMsgFn func(t *testing.T, env *TestEnvironment) *message.Message
	// ExpectedTopics lists all topics where messages are expected as a result
	ExpectedTopics []string
	// ValidateFn validates the results of the test
	ValidateFn func(t *testing.T, env *TestEnvironment, triggerMsg *message.Message,
		receivedMsgs map[string][]*message.Message, initialState interface{})
	// ExpectError indicates if the test expects an error
	ExpectError bool
	// MessageTimeout is the maximum time to wait for messages (defaults to 5s)
	MessageTimeout time.Duration
	// DeleteConsumerFn is a function that cleans up NATS consumers
	DeleteConsumerFn func(env *TestEnvironment, topic string, consumerName string) error
}

// sanitizeForNATS removes characters that aren't allowed in NATS subject names
func sanitizeForNATS(name string) string {
	// Replace common test characters with underscore
	replacer := map[rune]rune{
		' ':  '_',
		'(':  '_',
		')':  '_',
		'/':  '_',
		'\\': '_',
		'.':  '_',
	}

	result := make([]rune, 0, len(name))
	for _, ch := range name {
		if r, ok := replacer[ch]; ok {
			result = append(result, r)
		} else if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' || ch == '-' {
			result = append(result, ch)
		}
	}
	return string(result)
}

// RunTest executes a test case in the given test environment
func RunTest(t *testing.T, tc TestCase, env *TestEnvironment) {
	t.Helper()

	// Apply defaults
	if tc.MessageTimeout == 0 {
		tc.MessageTimeout = 5 * time.Second
	}

	// Generate a unique ID for this test case to isolate subscriptions
	testID := fmt.Sprintf("%s-%s", sanitizeForNATS(t.Name()), uuid.New().String()[:8])

	// Setup initial state
	initialState := tc.SetupFn(t, env)

	// Setup message tracking
	receivedMsgs := make(map[string][]*message.Message)
	mu := &sync.Mutex{}
	subscriberWg := &sync.WaitGroup{}

	// Use a cancelable context for subscriptions
	subCtx, cancelSub := context.WithCancel(env.Ctx)

	// Map to store consumer names by topic
	consumersByTopic := make(map[string]string)

	// Ensure proper cleanup
	t.Cleanup(func() {
		log.Printf("Test Case %q: Canceling subscription context.", t.Name())
		cancelSub()

		log.Printf("Test Case %q: Waiting for subscriber goroutines to finish.", t.Name())
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer waitCancel()

		waitCh := make(chan struct{})
		go func() {
			subscriberWg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			log.Printf("Test Case %q: Subscriber goroutines finished.", t.Name())
		case <-waitCtx.Done():
			log.Printf("Test Case %q: WARNING: Subscriber goroutines did not finish within timeout after context cancellation.", t.Name())
		}

		// Clean up NATS consumers if we have a method to do so
		if deleteConsumerFn := tc.DeleteConsumerFn; deleteConsumerFn != nil {
			for topic, consumerName := range consumersByTopic {
				if err := deleteConsumerFn(env, topic, consumerName); err != nil {
					t.Logf("Warning: failed to delete consumer %s for topic %s: %v",
						consumerName, topic, err)
				}
			}
		}
	})

	// Setup subscribers for all expected topics
	for _, topic := range tc.ExpectedTopics {
		// Generate consumer name for this topic and test
		consumerName := fmt.Sprintf("test-%s-%s", sanitizeForNATS(topic), testID)
		consumersByTopic[topic] = consumerName

		// Access EventBus through the environment
		msgCh, err := env.EventBus.Subscribe(subCtx, topic)
		if err != nil {
			t.Fatalf("Failed to subscribe to topic %q: %v", topic, err)
		}

		subscriberWg.Add(1)
		go func(topic string, messages <-chan *message.Message) {
			defer subscriberWg.Done()
			for {
				select {
				case msg, ok := <-messages:
					if !ok {
						return
					}
					log.Printf("Test received message %s on topic %q", msg.UUID, topic)
					mu.Lock()
					receivedMsgs[topic] = append(receivedMsgs[topic], msg)
					mu.Unlock()
					msg.Ack()
				case <-subCtx.Done():
					return
				}
			}
		}(topic, msgCh)
	}

	// Wait a moment to ensure all subscribers are ready
	time.Sleep(200 * time.Millisecond)

	// Publish the message to start the flow
	triggerMsg := tc.PublishMsgFn(t, env)

	// Wait for expected messages on all topics
	for _, topic := range tc.ExpectedTopics {
		msgs := WaitForMessages(t, env, receivedMsgs, mu, topic, 1, tc.MessageTimeout)

		if len(msgs) == 0 && !tc.ExpectError {
			t.Fatalf("Failed to receive message on topic %q within timeout", topic)
		}
	}

	// Final validation using test case's validate function
	tc.ValidateFn(t, env, triggerMsg, receivedMsgs, initialState)
}

// WaitForMessages waits for at least count messages on the specified topic
func WaitForMessages(t *testing.T, env *TestEnvironment, receivedMsgs map[string][]*message.Message, mu *sync.Mutex, topic string, count int, timeout time.Duration) []*message.Message {
	t.Helper()

	ctx, cancel := context.WithTimeout(env.Ctx, timeout)
	defer cancel()

	t.Logf("Waiting for %d messages on topic %q with timeout %v", count, topic, timeout)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout - collect what we have so far
			mu.Lock()
			msgs, exists := receivedMsgs[topic]
			currentCount := 0
			if exists {
				currentCount = len(msgs)
			}
			mu.Unlock()

			if currentCount >= count {
				return msgs
			}

			t.Logf("Timeout waiting for %d messages on topic %q. Received %d.",
				count, topic, currentCount)
			return nil

		case <-ticker.C:
			mu.Lock()
			msgs, exists := receivedMsgs[topic]
			currentCount := 0
			if exists {
				currentCount = len(msgs)
			}

			if currentCount >= count {
				result := make([]*message.Message, len(msgs))
				copy(result, msgs)
				mu.Unlock()
				t.Logf("Successfully received %d messages on topic %q", currentCount, topic)
				return result
			}

			mu.Unlock()
		}
	}
}
