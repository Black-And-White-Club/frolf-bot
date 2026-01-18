package testutils

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
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
// func sanitizeForNATS(name string) string {
// 	// Replace common test characters with underscore
// 	replacer := map[rune]rune{
// 		' ':  '_',
// 		'(':  '_',
// 		')':  '_',
// 		'/':  '_',
// 		'\\': '_',
// 		'.':  '_',
// 	}

// 	result := make([]rune, 0, len(name))
// 	for _, ch := range name {
// 		if r, ok := replacer[ch]; ok {
// 			result = append(result, r)
// 		} else if (ch >= 'a' && ch <= 'z') ||
// 			(ch >= 'A' && ch <= 'Z') ||
// 			(ch >= '0' && ch <= '9') ||
// 			ch == '_' || ch == '-' {
// 			result = append(result, ch)
// 		}
// 	}
// 	return string(result)
// }

// // deleteAllTestConsumers deletes all consumers with names starting with "test-" from known streams
// func deleteAllTestConsumers(env *TestEnvironment) error {
// 	ctx, cancel := context.WithTimeout(env.Ctx, 5*time.Second)
// 	defer cancel()

// 	streamNames := []string{"round", "user", "discord", "leaderboard", "score"}

// 	for _, streamName := range streamNames {
// 		stream, err := env.JetStream.Stream(ctx, streamName)
// 		if err != nil {
// 			// Stream doesn't exist, skip
// 			continue
// 		}

// 		consumers := stream.ListConsumers(ctx)
// 		for ci := range consumers.Info() {
// 			if ci == nil {
// 				continue
// 			}
// 			// Delete any consumer that starts with "test-"
// 			if strings.HasPrefix(ci.Name, "test-") {
// 				if err := env.JetStream.DeleteConsumer(ctx, streamName, ci.Name); err != nil {
// 					log.Printf("Warning: failed to delete test consumer %q from stream %q: %v", ci.Name, streamName, err)
// 				} else {
// 					log.Printf("Deleted leftover test consumer %q from stream %q", ci.Name, streamName)
// 				}
// 			}
// 		}
// 		if err := consumers.Err(); err != nil {
// 			log.Printf("Warning: error listing consumers for stream %q: %v", streamName, err)
// 		}
// 	}

// 	return nil
// }

// defaultDeleteConsumer attempts to delete a JetStream consumer from known streams
// func defaultDeleteConsumer(env *TestEnvironment, topic, consumerName string) error {
// 	ctx, cancel := context.WithTimeout(env.Ctx, 3*time.Second)
// 	defer cancel()

// 	// Try common stream names (map topics to their streams)
// 	streamNames := []string{"round", "user", "discord", "delayed", "leaderboard", "score"}

// 	for _, streamName := range streamNames {
// 		err := env.JetStream.DeleteConsumer(ctx, streamName, consumerName)
// 		if err == nil {
// 			log.Printf("Deleted consumer %q from stream %q", consumerName, streamName)
// 			return nil
// 		}

// 		errMsg := err.Error()
// 		// Consumer or stream not found in this stream, try next one (this is expected)
// 		if strings.Contains(errMsg, "consumer not found") ||
// 			strings.Contains(errMsg, "CONSUMER_NOT_FOUND") ||
// 			strings.Contains(errMsg, "stream not found") ||
// 			strings.Contains(errMsg, "err_code=10059") {
// 			continue
// 		}

// 		// Log unexpected errors but continue trying other streams
// 		log.Printf("Error deleting consumer %q from stream %q: %v", consumerName, streamName, err)
// 	}

// 	// Consumer not found is actually fine - might have been auto-cleaned
// 	return nil
// }

// RunTest executes a test case with smart polling to handle async event delays.
// RunTest executes a test case with smart polling to handle async event delays.
func RunTest(t *testing.T, tc TestCase, env *TestEnvironment) {
	t.Helper()

	if tc.MessageTimeout == 0 {
		tc.MessageTimeout = 10 * time.Second
	}

	// 1. Setup per-test state
	initialState := tc.SetupFn(t, env)

	receivedMsgs := make(map[string][]*message.Message)
	mu := &sync.Mutex{}
	subscriberWg := &sync.WaitGroup{}

	// Create a context that we can cancel to stop the test subscribers
	subCtx, cancelSub := context.WithCancel(context.Background())
	defer func() {
		cancelSub()
		subscriberWg.Wait()
	}()

	// 2. Subscribe to expected topics
	for _, topic := range tc.ExpectedTopics {
		msgCh, err := env.EventBus.SubscribeForTest(subCtx, topic)
		if err != nil {
			t.Fatalf("SubscribeForTest failed for topic %s: %v", topic, err)
		}

		subscriberWg.Add(1)
		go func(topic string, ch <-chan *message.Message) {
			defer subscriberWg.Done()
			for {
				select {
				case msg, ok := <-ch:
					if !ok {
						return
					}
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

	// No wait needed - SubscribeForTest creates ephemeral consumer synchronously

	// 3. Trigger the event
	triggerMsg := tc.PublishMsgFn(t, env)

	// 4. Polling Loop
	deadline := time.Now().Add(tc.MessageTimeout)

	for time.Now().Before(deadline) {
		mu.Lock()
		// Copy messages for validation to avoid race conditions
		snapshot := make(map[string][]*message.Message)
		for k, v := range receivedMsgs {
			snapshot[k] = v
		}
		mu.Unlock()

		// Check if we have at least one message per topic before validating
		allTopicsPopulated := true
		for _, topic := range tc.ExpectedTopics {
			if len(snapshot[topic]) == 0 {
				allTopicsPopulated = false
				break
			}
		}

		if allTopicsPopulated {
			// Try to validate. We use a separate goroutine to catch Fatalf (runtime.Goexit)
			if passed := internalValidate(tc.ValidateFn, env, triggerMsg, snapshot, initialState); passed {
				return // SUCCESS!
			}
		}
		// Polling interval
		time.Sleep(100 * time.Millisecond)
	}

	// 5. Final Attempt: If we timed out, run validation one last time with the REAL T
	// to report the specific failure reasons to the console.
	mu.Lock()
	tc.ValidateFn(t, env, triggerMsg, receivedMsgs, initialState)
	mu.Unlock()
}

// internalValidate runs the validation function in a protected goroutine.
// It returns true if the validation passed, and false if it failed or panicked.
func internalValidate(
	fn func(t *testing.T, env *TestEnvironment, tm *message.Message, rm map[string][]*message.Message, is interface{}),
	env *TestEnvironment, tm *message.Message, rm map[string][]*message.Message, is interface{},
) (passed bool) {
	tmpT := &testing.T{}
	done := make(chan struct{})

	go func() {
		defer func() {
			// Catch runtime.Goexit() (from t.Fatalf) or actual panics
			recover()
			close(done)
		}()
		fn(tmpT, env, tm, rm, is)
	}()

	<-done
	return !tmpT.Failed()
}

// func waitForJetStreamConsumer(
// 	t *testing.T,
// 	env *TestEnvironment,
// 	topic string,
// 	timeout time.Duration,
// ) {
// 	t.Helper()

// 	ctx, cancel := context.WithTimeout(env.Ctx, timeout)
// 	defer cancel()

// 	streamNames := []string{"round", "user", "discord", "leaderboard", "score"}

// 	ticker := time.NewTicker(50 * time.Millisecond)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			t.Fatalf("Timed out waiting for JetStream consumer for topic %q", topic)

// 		case <-ticker.C:
// 			for _, streamName := range streamNames {
// 				stream, err := env.JetStream.Stream(ctx, streamName)
// 				if err != nil {
// 					continue
// 				}

// 				consumers := stream.ListConsumers(ctx)
// 				for ci := range consumers.Info() {
// 					if ci == nil {
// 						continue
// 					}
// 					// FilterSubject is the key signal that the consumer is active
// 					if strings.Contains(ci.Config.FilterSubject, topic) {
// 						return
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// WaitForMessages waits for at least count messages on the specified topic
func WaitForMessages(
	t *testing.T,
	env *TestEnvironment,
	receivedMsgs map[string][]*message.Message,
	mu *sync.Mutex,
	topic string,
	count int,
	timeout time.Duration,
) []*message.Message {
	t.Helper()

	ctx, cancel := context.WithTimeout(env.Ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			msgs := receivedMsgs[topic]
			mu.Unlock()
			return msgs

		case <-ticker.C:
			mu.Lock()
			msgs := receivedMsgs[topic]
			currentCount := len(msgs)
			mu.Unlock()

			if count == 0 && currentCount > 0 {
				return msgs
			}

			if count > 0 && currentCount >= count {
				return msgs
			}
		}
	}
}
