package roundhandler_integration_tests

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	eventbusmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/eventbus"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/trace/noop"
)

// Global variables for the test environment, initialized once.
var (
	testEnv     *testutils.TestEnvironment
	testEnvOnce sync.Once
	testEnvErr  error
)

// RoundHandlerTestDeps holds shared dependencies for round handler tests.
type RoundHandlerTestDeps struct {
	*testutils.TestEnvironment
	RoundModule       *round.Module
	Router            *message.Router
	EventBus          eventbus.EventBus
	MessageCapture    *testutils.MessageCapture
	TestObservability observability.Observability
	TestHelpers       utils.Helpers
	cleanup           func()
}

// GetTestEnv creates a new test environment for the test.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()
	if testEnv == nil {
		t.Fatalf("RoundHandlers Global test environment not initialized")
	}
	return testEnv
}

// streamTopicMap defines which stream each topic belongs to for faster lookup
var streamTopicMap = map[string]string{
	roundevents.RoundEntityCreated:    "round",
	roundevents.RoundValidationFailed: "discord", // This goes to discord stream
	roundevents.RoundCreated:          "discord", // This goes to discord stream
	roundevents.RoundCreationFailed:   "discord", // This goes to discord stream
}

// getStreamForTopic returns the appropriate stream name for a topic
func getStreamForTopic(topic string) string {
	if stream, exists := streamTopicMap[topic]; exists {
		return stream
	}

	// Fallback logic for dynamic topics
	switch {
	case strings.HasPrefix(topic, "discord."):
		return "discord"
	case strings.HasPrefix(topic, "leaderboard."):
		return "leaderboard"
	case strings.HasPrefix(topic, "score."):
		return "score"
	case strings.HasPrefix(topic, "user."):
		return "user"
	case strings.HasPrefix(topic, "delayed."):
		return "delayed"
	default:
		return "round"
	}
}

// SetupTestRoundHandler sets up the environment and dependencies for Round handler tests.
func SetupTestRoundHandler(t *testing.T) RoundHandlerTestDeps {
	t.Helper()

	env := GetTestEnv(t)
	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	testCtx, testCancel := context.WithCancel(env.Ctx)

	// Clean up NATS consumers for all streams before starting the test
	streamNames := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}
	if err := env.ResetJetStreamState(testCtx, streamNames...); err != nil {
		testCancel()
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}

	if err := testutils.TruncateTables(testCtx, env.DB, "users", "scores", "leaderboards", "rounds"); err != nil {
		testCancel()
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	// Create shared resources with minimal logging
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	watermillLogger := watermill.NopLogger{}

	// Create main EventBus for business logic
	eventBusImpl, err := eventbus.NewEventBus(
		testCtx,
		env.Config.NATS.URL,
		discardLogger,
		"backend",
		eventbusmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		testCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Ensure required streams exist with proper subjects
	streamSubjects := map[string][]string{
		"discord":     {"discord.>"},
		"round":       {"round.>"},
		"user":        {"user.>"},
		"leaderboard": {"leaderboard.>"},
		"score":       {"score.>"},
		"delayed":     {"delayed.>"},
	}

	for streamName, subjects := range streamSubjects {
		if err := ensureStreamExistsWithSubjects(testCtx, eventBusImpl, streamName, subjects); err != nil {
			eventBusImpl.Close()
			testCancel()
			t.Fatalf("Failed to ensure stream %q exists: %v", streamName, err)
		}
	}

	// Optimized router config
	routerConfig := message.RouterConfig{CloseTimeout: 250 * time.Millisecond}
	watermillRouter, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		eventBusImpl.Close()
		testCancel()
		t.Fatalf("Failed to create Watermill router: %v", err)
	}

	// Create message capture for test verification - Remove debug logging
	captureTopics := []string{
		roundevents.RoundEntityCreated,
		roundevents.RoundValidationFailed,
		roundevents.RoundCreated,
		roundevents.RoundCreationFailed,
	}
	messageCapture := createSilentMessageCapture(captureTopics...)

	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: discardLogger,
		},
		Registry: &observability.Registry{
			RoundMetrics: roundmetrics.NewNoop(),
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Logger:       discardLogger,
		},
	}
	realHelpers := utils.NewHelper(discardLogger)
	roundDB := &rounddb.RoundDBImpl{DB: env.DB}

	// Create round module
	roundModule, err := round.NewRoundModule(
		testCtx,
		env.Config,
		testObservability,
		roundDB,
		eventBusImpl,
		watermillRouter,
		realHelpers,
		testCtx,
	)
	if err != nil {
		eventBusImpl.Close()
		testCancel()
		t.Fatalf("Failed to create round module: %v", err)
	}

	// Start optimized message capture consumers
	consumerWg := startOptimizedMessageCapture(t, testCtx, eventBusImpl, messageCapture, captureTopics)

	// Start the router
	routerWg := &sync.WaitGroup{}
	routerWg.Add(1)
	go func() {
		defer routerWg.Done()
		if runErr := watermillRouter.Run(testCtx); runErr != nil && runErr != context.Canceled {
			// Only log non-cancellation errors
			if !strings.Contains(runErr.Error(), "context canceled") {
				t.Errorf("Watermill router stopped with error: %v", runErr)
			}
		}
	}()

	// Minimal startup time
	time.Sleep(50 * time.Millisecond)

	// Optimized cleanup function
	cleanup := func() {
		os.Setenv("APP_ENV", oldEnv)
		testCancel()

		// Fast cleanup with shorter timeout
		done := make(chan struct{})
		go func() {
			consumerWg.Wait()
			routerWg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond): // Reduced timeout
		}

		// Parallel resource cleanup
		var wg sync.WaitGroup
		if watermillRouter != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				watermillRouter.Close()
			}()
		}

		if roundModule != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				roundModule.Close()
			}()
		} else if eventBusImpl != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				eventBusImpl.Close()
			}()
		}

		// Wait for cleanup with timeout
		cleanupDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(cleanupDone)
		}()

		select {
		case <-cleanupDone:
		case <-time.After(1 * time.Second):
		}
	}

	t.Cleanup(cleanup)

	return RoundHandlerTestDeps{
		TestEnvironment:   env,
		RoundModule:       roundModule,
		Router:            watermillRouter,
		EventBus:          eventBusImpl,
		MessageCapture:    messageCapture,
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
		cleanup:           cleanup,
	}
}

// ensureStreamExistsWithSubjects checks if a stream exists and creates it with proper subjects if not
func ensureStreamExistsWithSubjects(ctx context.Context, eventBus eventbus.EventBus, streamName string, subjects []string) error {
	js := eventBus.GetJetStream()
	_, err := js.Stream(ctx, streamName)
	if err != nil && strings.Contains(err.Error(), "stream not found") {
		// Stream doesn't exist, create it with proper subjects
		cfg := jetstream.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
		}
		_, err = js.CreateStream(ctx, cfg)
		return err
	}
	return err
}

// createSilentMessageCapture creates MessageCapture without debug logging
func createSilentMessageCapture(topicsToCapture ...string) *testutils.MessageCapture {
	// We'll use the existing NewMessageCapture but need to remove debug logging
	return testutils.NewMessageCapture(topicsToCapture...)
}

// startOptimizedMessageCapture starts consumer goroutines with better error handling
func startOptimizedMessageCapture(t *testing.T, ctx context.Context, eventBusImpl eventbus.EventBus, messageCapture *testutils.MessageCapture, captureTopics []string) *sync.WaitGroup {
	wg := &sync.WaitGroup{}

	for i, topic := range captureTopics {
		wg.Add(1)

		go func(topicName string, index int) {
			defer wg.Done()

			consumerName := fmt.Sprintf("test-capture-%s-%d-%d",
				strings.ReplaceAll(topicName, ".", "-"),
				index,
				time.Now().UnixNano())

			js := eventBusImpl.GetJetStream()
			streamName := getStreamForTopic(topicName)

			// Retry consumer creation with exponential backoff
			var consumer jetstream.Consumer
			var err error
			for attempts := 0; attempts < 3; attempts++ {
				consumer, err = js.CreateConsumer(ctx, streamName, jetstream.ConsumerConfig{
					Name:          consumerName,
					FilterSubject: topicName,
					AckPolicy:     jetstream.AckExplicitPolicy,
				})
				if err == nil {
					break
				}

				// Only log non-cancellation errors
				if !strings.Contains(err.Error(), "context canceled") {
					t.Logf("Attempt %d: Failed to create consumer for %s: %v", attempts+1, topicName, err)
				}

				if attempts < 2 {
					time.Sleep(time.Duration(attempts+1) * 50 * time.Millisecond)
				}
			}

			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					t.Errorf("Failed to create consumer for %s after retries: %v", topicName, err)
				}
				return
			}

			// Optimized message processing loop
			ticker := time.NewTicker(25 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					msgs, err := consumer.Fetch(5, jetstream.FetchMaxWait(10*time.Millisecond))
					if err != nil {
						if err == jetstream.ErrNoMessages {
							continue
						}
						if !strings.Contains(err.Error(), "context canceled") {
							t.Logf("Error fetching messages for %s: %v", topicName, err)
						}
						continue
					}

					for msg := range msgs.Messages() {
						msgID := getMessageID(msg)
						watermillMsg := message.NewMessage(msgID, msg.Data())

						// Efficient header copying
						if headers := msg.Headers(); headers != nil {
							for k, v := range headers {
								if len(v) > 0 {
									watermillMsg.Metadata.Set(k, v[0])
								}
							}
						}

						// Call capture handler without debug logging
						captureHandler := messageCapture.CaptureHandler(topicName)
						if _, err := captureHandler(watermillMsg); err != nil {
							t.Logf("Capture handler error for %s: %v", topicName, err)
						}

						msg.Ack()
					}
				}
			}
		}(topic, i)
	}

	return wg
}

// getMessageID extracts message ID from NATS message headers
func getMessageID(msg jetstream.Msg) string {
	if headers := msg.Headers(); headers != nil {
		if id := headers.Get("Nats-Msg-Id"); id != "" {
			return id
		}
		if id := headers.Get("_watermill_message_uuid"); id != "" {
			return id
		}
	}
	return fmt.Sprintf("generated-%d", time.Now().UnixNano())
}

// Helper functions
func tagPtr(n sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &n
}

func boolPtr(b bool) *bool {
	return &b
}

// WaitForMessageProcessed waits for a signal on a channel indicating a message has been processed.
func WaitForMessageProcessed(msgProcessedChan <-chan struct{}, timeout time.Duration) error {
	select {
	case <-msgProcessedChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out waiting for message to be processed")
	}
}
