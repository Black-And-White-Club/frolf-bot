package roundhandler_integration_tests

import (
	"context"
	"fmt"
	"io"
	"log"
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

var standardStreamNames = []string{"user", "discord", "leaderboard", "round", "score"}

// Default timeout for waiting for messages in tests - reduced for faster test failure
const defaultTimeout = 2 * time.Second

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
	testEnv        *testutils.TestEnvironment
	testEnvOnce    sync.Once
	testEnvErr     error
	sharedRouter   *message.Router
	sharedModule   *round.Module
	sharedInitOnce sync.Once
	sharedInitErr  error
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
		// NOTE: Cleanup is handled in TestMain, not per-test
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

// initSharedRouterAndModule initializes the shared router and round module once for all tests
func initSharedRouterAndModule(t *testing.T, env *testutils.TestEnvironment) error {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	watermillLogger := watermill.NopLogger{}

	// Ensure all required streams exist
	for _, streamName := range standardStreamNames {
		if err := env.EventBus.CreateStream(env.Ctx, streamName); err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "stream name already in use") && err != jetstream.ErrStreamNameAlreadyInUse {
				return fmt.Errorf("failed to create stream %s: %w", streamName, err)
			}
		}
	}

	// Create shared router
	routerConfig := message.RouterConfig{CloseTimeout: 500 * time.Millisecond}
	router, err := message.NewRouter(routerConfig, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}

	// Create shared module
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

	module, err := round.NewRoundModule(
		env.Ctx,
		env.Config,
		testObservability,
		roundDB,
		env.DBService.UserDB,
		env.EventBus,
		router,
		realHelpers,
		env.Ctx,
	)
	if err != nil {
		return fmt.Errorf("failed to create round module: %w", err)
	}

	// Start the router in a goroutine
	go func() {
		if runErr := router.Run(env.Ctx); runErr != nil && runErr != context.Canceled {
			if !strings.Contains(runErr.Error(), "context canceled") {
				log.Printf("Watermill router stopped with error: %v", runErr)
			}
		}
	}()

	// Wait for router to be ready with proper checking
	maxWait := 5 * time.Second
	checkInterval := 50 * time.Millisecond
	startTime := time.Now()

	for {
		if router.IsRunning() {
			log.Printf("Router is running after %v", time.Since(startTime))
			break
		}
		if time.Since(startTime) > maxWait {
			return fmt.Errorf("router failed to start within %v", maxWait)
		}
		time.Sleep(checkInterval)
	}

	// Additional small delay to ensure all handlers are registered
	time.Sleep(100 * time.Millisecond)

	sharedRouter = router
	sharedModule = module
	return nil
}

// SetupTestRoundHandler sets up the environment and dependencies for Round handler tests.
func SetupTestRoundHandler(t *testing.T) RoundHandlerTestDeps {
	t.Helper()

	env := GetTestEnv(t)

	// Initialize shared router and module once
	sharedInitOnce.Do(func() {
		sharedInitErr = initSharedRouterAndModule(t, env)
	})

	if sharedInitErr != nil {
		t.Fatalf("Failed to initialize shared router/module: %v", sharedInitErr)
	}

	oldEnv := os.Getenv("APP_ENV")
	os.Setenv("APP_ENV", "test")

	// Create per-test context that will be canceled when the test completes
	// This ensures consumer goroutines stop and don't interfere with other tests
	testCtx, testCancel := context.WithCancel(env.Ctx)

	// Create per-test message capture for test verification
	captureTopics := []string{
		roundevents.RoundEntityCreatedV1,
		roundevents.RoundValidationFailedV1,
		roundevents.RoundCreatedV1,
		roundevents.RoundCreationFailedV1,
		roundevents.RoundEventMessageIDUpdatedV1,
		roundevents.RoundDeleteAuthorizedV1,
		roundevents.RoundDeleteErrorV1,
		roundevents.RoundDeletedV1,
		roundevents.RoundAllScoresSubmittedV1,
		roundevents.RoundFinalizedV1,
		roundevents.RoundFinalizedDiscordV1,
		roundevents.ProcessRoundScoresRequestedV1,
		roundevents.RoundFinalizationErrorV1,
		roundevents.RoundParticipantJoinRequestedV1,
		roundevents.RoundParticipantRemovalRequestedV1,
		roundevents.RoundParticipantJoinValidationRequestedV1,
		roundevents.RoundParticipantStatusCheckErrorV1,
		roundevents.RoundParticipantJoinErrorV1,
		roundevents.RoundParticipantJoinedV1,
		roundevents.RoundParticipantRemovedV1,
		roundevents.RoundParticipantDeclinedV1,
		roundevents.LeaderboardGetTagNumberRequestedV1,
		roundevents.RoundParticipantStatusUpdateRequestedV1,
		roundevents.RoundParticipantRemovalErrorV1,
		sharedevents.RoundTagLookupFoundV1,
		sharedevents.RoundTagLookupNotFoundV1,
		sharedevents.TagUpdateForScheduledRoundsV1,
		roundevents.TagsUpdatedForScheduledRoundsV1,
		roundevents.RoundUpdateErrorV1,
		roundevents.RoundReminderSentV1,
		roundevents.RoundErrorV1,
		roundevents.RoundRetrievedV1,
		roundevents.RoundScoreUpdateErrorV1,
		roundevents.RoundScoreUpdateValidatedV1,
		roundevents.RoundParticipantScoreUpdatedV1,
		roundevents.RoundScoresPartiallySubmittedV1,
		roundevents.RoundStartedDiscordV1,
		roundevents.RoundUpdateValidatedV1,
		roundevents.RoundUpdatedV1,
		roundevents.RoundScheduleUpdatedV1,
	}

	messageCapture := createSilentMessageCapture(captureTopics...)

	// Generate unique test ID for consumer names to avoid conflicts
	testID := fmt.Sprintf("%s-%d",
		strings.ReplaceAll(strings.ReplaceAll(t.Name(), "/", "-"), " ", "-"),
		time.Now().UnixNano())

	// Start message capture consumers with per-test context
	consumerWg := startOptimizedMessageCapture(t, testCtx, env.EventBus, messageCapture, captureTopics, testID)

	// Simple cleanup function - cancel context, wait for consumers, then purge streams
	cleanup := func() {
		// Cancel per-test context to stop consumer goroutines
		testCancel()

		// Wait for consumers to finish fetching (with shorter timeout since context is canceled)
		consumerDone := make(chan struct{})
		go func() {
			consumerWg.Wait()
			close(consumerDone)
		}()

		select {
		case <-consumerDone:
			// Consumer goroutines finished
		case <-time.After(500 * time.Millisecond):
			t.Logf("Warning: Consumer goroutines cleanup timed out for %s", t.Name())
		}

		// Don't purge streams - that would break parallel test execution!
		// Tests should filter messages by their own test-specific IDs (guild ID, round ID, etc.)
		// Each test creates unique test data and only looks for messages related to that data.

		// Reset environment
		os.Setenv("APP_ENV", oldEnv)
	}

	t.Cleanup(cleanup)

	// Use shared observability and helpers
	// discardLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	testObservability := observability.Observability{
		Provider: &observability.Provider{
			Logger: testLogger,
		},
		Registry: &observability.Registry{
			RoundMetrics: roundmetrics.NewNoop(),
			Tracer:       noop.NewTracerProvider().Tracer("test"),
			Logger:       testLogger,
		},
	}
	realHelpers := utils.NewHelper(testLogger)

	return RoundHandlerTestDeps{
		TestEnvironment:   env,
		RoundModule:       sharedModule,
		Router:            sharedRouter,
		EventBus:          env.EventBus,
		MessageCapture:    messageCapture,
		TestObservability: testObservability,
		TestHelpers:       realHelpers,
		testID:            testID,
		cleanup:           cleanup,
	}
}

// PrepareSubTest clears the message capture buffer to ensure test isolation between sub-tests.
// Call this at the beginning of each sub-test (t.Run) to avoid seeing messages from previous sub-tests.
//
// Example usage:
//
//	t.Run("sub-test 1", func(t *testing.T) {
//	    PrepareSubTest(deps)
//	    // ... test logic ...
//	})
func PrepareSubTest(deps RoundHandlerTestDeps) {
	deps.MessageCapture.Clear()
	// Brief sleep to allow any in-flight messages to be captured and cleared
	time.Sleep(50 * time.Millisecond)
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
	readyWg := &sync.WaitGroup{}

	for i, topic := range captureTopics {
		wg.Add(1)
		readyWg.Add(1)

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
					DeliverPolicy: jetstream.DeliverNewPolicy, // Only receive NEW messages, not historical ones
				})
				if err == nil {
					break
				}

				// Don't log from goroutine - can panic if test completes
				// Just continue with retries

				if attempts < 2 {
					time.Sleep(time.Duration(attempts+1) * 50 * time.Millisecond)
				}
			}

			if err != nil {
				// Don't call t.Errorf from goroutine - it can panic if test is done
				// Just log and return silently
				readyWg.Done()
				return
			}

			readyWg.Done()

			// Optimized message processing loop
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					msgs, err := consumer.Fetch(10, jetstream.FetchMaxWait(100*time.Millisecond))
					if err != nil {
						if err == jetstream.ErrNoMessages {
							continue
						}
						// Don't log from goroutine - just continue
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

	readyWg.Wait()
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
