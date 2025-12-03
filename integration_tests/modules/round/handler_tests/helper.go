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
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

func init() {
	// Initialize the global test environment once
	testEnvOnce.Do(func() {
		env, err := testutils.NewTestEnvironment(&testing.T{}) // This will need adjustment
		if err != nil {
			testEnvErr = fmt.Errorf("failed to initialize global test environment: %w", err)
			return
		}
		testEnv = env
	})
}

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
	testID            string
	cleanup           func()
}

// GetTestEnv creates a new test environment for the test.
// GetTestEnv creates or returns the shared test environment for the test.
func GetTestEnv(t *testing.T) *testutils.TestEnvironment {
	t.Helper()

	// Initialize if not already done
	testEnvOnce.Do(func() {
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			testEnvErr = fmt.Errorf("failed to initialize global test environment: %w", err)
			return
		}
		testEnv = env

		// Set up cleanup for the entire test suite
		t.Cleanup(func() {
			if testEnv != nil {
				testEnv.Cleanup()
				testEnv = nil
			}
		})
	})

	if testEnvErr != nil {
		t.Fatalf("Test environment initialization failed: %v", testEnvErr)
	}

	if testEnv == nil {
		t.Fatalf("Test environment not initialized")
	}

	return testEnv
}

// streamTopicMap defines which stream each topic belongs to for faster lookup
var streamTopicMap = map[string]string{
	roundevents.RoundEntityCreated:                    "round",
	roundevents.RoundValidationFailed:                 "discord", // This goes to discord stream
	roundevents.RoundCreated:                          "discord", // This goes to discord stream
	roundevents.RoundCreationFailed:                   "discord", // This goes to discord stream
	roundevents.RoundEventMessageIDUpdated:            "round",   // This should go to round stream since it starts with "round."
	roundevents.RoundDeleteAuthorized:                 "round",   // ✅ Correct - "round.delete.authorized"
	roundevents.RoundDeleteError:                      "round",   // 🔧 FIX: Change from "discord" to "round" - because "round.delete.error"
	roundevents.RoundDeleted:                          "discord", // ✅ Correct - "discord.round.deleted"
	roundevents.RoundAllScoresSubmitted:               "round",
	roundevents.RoundFinalized:                        "round",
	roundevents.RoundFinalized:                        "discord",
	roundevents.ProcessRoundScoresRequest:             "score",
	roundevents.RoundFinalizationError:                "round",
	roundevents.RoundParticipantJoinRequest:           "round",
	roundevents.RoundParticipantRemovalRequest:        "round",
	roundevents.RoundParticipantJoinValidationRequest: "round",
	roundevents.RoundParticipantStatusCheckError:      "discord",
	roundevents.RoundParticipantJoinError:             "round",
	roundevents.RoundParticipantJoined:                "discord",
	roundevents.RoundParticipantRemoved:               "discord",
	roundevents.RoundParticipantDeclined:              "discord",
	roundevents.LeaderboardGetTagNumberRequest:        "leaderboard",
	roundevents.RoundParticipantStatusUpdateRequest:   "round",
	roundevents.RoundParticipantRemovalError:          "round",
	sharedevents.RoundTagLookupFound:                  "round",
	sharedevents.RoundTagLookupNotFound:               "round",
	roundevents.TagUpdateForScheduledRounds:           "round",
	roundevents.TagsUpdatedForScheduledRounds:         "discord",
	roundevents.RoundUpdateError:                      "round",
	roundevents.RoundReminder:                         "discord",
	roundevents.RoundError:                            "round",
	roundevents.RoundRetrieved:                        "discord",
	roundevents.RoundScoreUpdateError:                 "discord",
	roundevents.RoundScoreUpdateValidated:             "round",
	roundevents.RoundParticipantScoreUpdated:          "round",
	roundevents.RoundNotAllScoresSubmitted:            "discord",
	roundevents.RoundStarted:                          "discord",
	roundevents.RoundUpdateValidated:                  "round",
	roundevents.RoundUpdated:                          "round",
	roundevents.RoundScheduleUpdate:                   "discord",
}

// getStreamForTopic returns the appropriate standard stream name for a topic
func getStreamForTopic(topic string) string {
	if stream, exists := streamTopicMap[topic]; exists {
		return stream
	}

	// Fallback logic for dynamic topics - return standard stream names
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

	// Check if containers should be recreated for stability
	if err := env.MaybeRecreateContainers(context.Background()); err != nil {
		t.Fatalf("Failed to handle container recreation: %v", err)
	}

	// Perform deep cleanup between tests for better isolation
	if err := env.DeepCleanup(); err != nil {
		t.Fatalf("Failed to perform deep cleanup: %v", err)
	}

	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	testCtx, testCancel := context.WithCancel(env.Ctx)

	// Use standard stream names that the EventBus recognizes
	standardStreamNames := []string{"user", "discord", "leaderboard", "round", "score"}

	// Additional cleanup after DeepCleanup
	if err := env.ResetJetStreamState(testCtx, standardStreamNames...); err != nil {
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

	// Use standard "backend" app type - this will create the standard streams
	eventBusImpl, err := eventbus.NewEventBus(
		testCtx,
		env.Config.NATS.URL,
		discardLogger,
		"backend", // Use standard app type that EventBus recognizes
		eventbusmetrics.NewNoop(),
		noop.NewTracerProvider().Tracer("test"),
	)
	if err != nil {
		testCancel()
		t.Fatalf("Failed to create EventBus: %v", err)
	}

	// Ensure all required streams exist after EventBus creation
	// Assumes eventBusImpl.CreateStream is a synchronous operation.
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(testCtx, streamName); err != nil {
			eventBusImpl.Close()
			testCancel()
			t.Fatalf("Failed to create stream %s: %v", streamName, err)
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

	// Create message capture for test verification
	captureTopics := []string{
		roundevents.RoundEntityCreated,
		roundevents.RoundValidationFailed,
		roundevents.RoundCreated,
		roundevents.RoundCreationFailed,
		roundevents.RoundEventMessageIDUpdated,
		roundevents.RoundDeleteAuthorized,
		roundevents.RoundDeleteError,
		roundevents.RoundDeleted,
		roundevents.RoundAllScoresSubmitted,
		roundevents.RoundFinalized,
		roundevents.RoundFinalized,
		roundevents.ProcessRoundScoresRequest,
		roundevents.RoundFinalizationError,
		roundevents.RoundParticipantJoinRequest,
		roundevents.RoundParticipantRemovalRequest,
		roundevents.RoundParticipantJoinValidationRequest,
		roundevents.RoundParticipantStatusCheckError,
		roundevents.RoundParticipantJoinError,
		roundevents.RoundParticipantJoined,
		roundevents.RoundParticipantRemoved,
		roundevents.RoundParticipantDeclined,
		roundevents.LeaderboardGetTagNumberRequest,
		roundevents.RoundParticipantStatusUpdateRequest,
		roundevents.RoundParticipantRemovalError,
		sharedevents.RoundTagLookupFound,
		sharedevents.RoundTagLookupNotFound,
		roundevents.TagUpdateForScheduledRounds,
		roundevents.TagsUpdatedForScheduledRounds,
		roundevents.RoundUpdateError,
		roundevents.RoundReminder,
		roundevents.RoundError,
		roundevents.RoundRetrieved,
		roundevents.RoundScoreUpdateError,
		roundevents.RoundScoreUpdateValidated,
		roundevents.RoundParticipantScoreUpdated,
		roundevents.RoundAllScoresSubmitted,
		roundevents.RoundNotAllScoresSubmitted,
		roundevents.RoundStarted,
		roundevents.RoundUpdateValidated,
		roundevents.RoundUpdated,
		roundevents.RoundScheduleUpdate,
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
	// Assumes round.NewRoundModule synchronously sets up its handlers
	// with the watermillRouter.
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

	// Generate unique test ID for consumer names to avoid conflicts
	testID := fmt.Sprintf("%s-%d",
		strings.ReplaceAll(strings.ReplaceAll(t.Name(), "/", "-"), " ", "-"),
		time.Now().UnixNano())

	// Start message capture consumers with standard streams
	// startOptimizedMessageCapture includes its own logic for ensuring consumers are created.
	consumerWg := startOptimizedMessageCapture(t, testCtx, eventBusImpl, messageCapture, captureTopics, testID)

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

	// Wait for the router to create the consumer for the update topic to avoid race conditions
	// Consumer durable is built as "<appType>-consumer-<sanitized_topic>" with appType="backend"
	updateTopic := roundevents.RoundEventMessageIDUpdate // "round.discord.event.id.update"
	updateStream := getStreamForTopic(updateTopic)       // should resolve to "round"
	consumerName := "backend-consumer-" + strings.ReplaceAll(strings.ReplaceAll(updateTopic, ".", "-"), "_", "-")

	js := eventBusImpl.GetJetStream()
	readyCtx, readyCancel := context.WithTimeout(testCtx, 2*time.Second)
	defer readyCancel()
	for {
		select {
		case <-readyCtx.Done():
			// Proceed anyway; tests may still pass but this reduces flakiness when successful
			goto afterReadyWait
		default:
			if _, err := js.Consumer(testCtx, updateStream, consumerName); err == nil {
				goto afterReadyWait
			}
			time.Sleep(25 * time.Millisecond)
		}
	}

afterReadyWait:

	// Optimized cleanup function
	cleanup := func() {
		// 1. Cancel context first to stop all operations
		testCancel()

		// 2. Stop router to prevent new message processing
		if watermillRouter != nil {
			watermillRouter.Close()
		}

		// 3. Wait for consumers to finish with timeout
		done := make(chan struct{})
		go func() {
			consumerWg.Wait()
			routerWg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second): // Reduced timeout for faster feedback
			t.Logf("Warning: Test cleanup (consumers/router) timed out for %s", t.Name())
		}

		// 4. Close remaining resources in parallel
		var wg sync.WaitGroup
		if roundModule != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = roundModule.Close()
			}()
		}
		// Always close EventBus explicitly to stop JetStream consumers and unblock subscribers
		if eventBusImpl != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = eventBusImpl.Close()
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
		case <-time.After(2 * time.Second): // Reduced timeout
			t.Logf("Warning: Resource cleanup (module/eventbus) timed out for %s", t.Name())
		}

		// 5. Reset environment
		os.Setenv("APP_ENV", oldEnv)
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
		testID:            testID,
		cleanup:           cleanup,
	}
}

// ensureStreamExistsWithSubjects checks if a stream exists and creates it with proper subjects if not
// func ensureStreamExistsWithSubjects(ctx context.Context, eventBus eventbus.EventBus, streamName string, subjects []string) error {
// 	js := eventBus.GetJetStream()
// 	_, err := js.Stream(ctx, streamName)
// 	if err != nil && strings.Contains(err.Error(), "stream not found") {
// 		// Stream doesn't exist, create it with proper subjects
// 		cfg := jetstream.StreamConfig{
// 			Name:     streamName,
// 			Subjects: subjects,
// 		}
// 		_, err = js.CreateStream(ctx, cfg)
// 		return err
// 	}
// 	return err
// }

// createSilentMessageCapture creates MessageCapture without debug logging
func createSilentMessageCapture(topicsToCapture ...string) *testutils.MessageCapture {
	// We'll use the existing NewMessageCapture but need to remove debug logging
	return testutils.NewMessageCapture(topicsToCapture...)
}

// startOptimizedMessageCapture starts consumer goroutines with better error handling and test isolation
func startOptimizedMessageCapture(t *testing.T, ctx context.Context, eventBusImpl eventbus.EventBus, messageCapture *testutils.MessageCapture, captureTopics []string, testID string) *sync.WaitGroup {
	wg := &sync.WaitGroup{}

	for i, topic := range captureTopics {
		wg.Add(1)

		go func(topicName string, index int) {
			defer wg.Done()

			consumerName := fmt.Sprintf("test-capture-%s-%s-%d-%d",
				strings.ReplaceAll(topicName, ".", "-"),
				testID,
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
					t.Logf("Attempt %d: Failed to create consumer for %s on stream %s: %v", attempts+1, topicName, streamName, err)
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

					// Non-blocking drain of fetched messages to avoid hangs on shutdown
					for {
						select {
						case <-ctx.Done():
							return
						default:
							msg, ok := <-msgs.Messages()
							if !ok {
								// No more messages in this batch
								goto afterDrain
							}

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
				afterDrain:
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
