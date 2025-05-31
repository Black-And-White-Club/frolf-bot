package roundhandler_integration_tests

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
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

var (
	sharedEnv     *testutils.TestEnvironment
	sharedEnvOnce sync.Once
	envMutex      sync.RWMutex
)

// getSharedTestEnv returns a shared environment for all tests in this package
// getSharedTestEnv returns a shared environment for all tests in this package
func getSharedTestEnv(t *testing.T) *testutils.TestEnvironment {
	sharedEnvOnce.Do(func() {
		env, err := testutils.NewTestEnvironment(t)
		if err != nil {
			t.Fatalf("Failed to create shared test environment: %v", err)
		}
		sharedEnv = env

		// DON'T register cleanup here - it runs after each test
		// Instead, we'll use a different approach or register it only once at package level
	})

	envMutex.RLock()
	defer envMutex.RUnlock()

	if sharedEnv == nil {
		t.Fatalf("Shared environment is nil - this should not happen")
	}

	return sharedEnv
}

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

// streamTopicMap defines which stream each topic belongs to for faster lookup
var streamTopicMap = map[string]string{
	roundevents.RoundEntityCreated:                    "round",
	roundevents.RoundValidationFailed:                 "discord",
	roundevents.RoundCreated:                          "discord",
	roundevents.RoundCreationFailed:                   "discord",
	roundevents.RoundEventMessageIDUpdated:            "round",
	roundevents.RoundDeleteAuthorized:                 "round",
	roundevents.RoundDeleteError:                      "round",
	roundevents.RoundDeleted:                          "discord",
	roundevents.RoundAllScoresSubmitted:               "round",
	roundevents.RoundFinalized:                        "round",
	roundevents.DiscordRoundFinalized:                 "discord",
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
	roundevents.DiscordRoundReminder:                  "discord",
	roundevents.RoundError:                            "round",
	roundevents.RoundRetrieved:                        "discord",
	roundevents.RoundScoreUpdateError:                 "discord",
	roundevents.RoundScoreUpdateValidated:             "round",
	roundevents.RoundParticipantScoreUpdated:          "round",
	roundevents.RoundNotAllScoresSubmitted:            "discord",
	roundevents.DiscordRoundStarted:                   "discord",
	roundevents.RoundUpdateValidated:                  "round",
	roundevents.RoundUpdated:                          "round",
	roundevents.RoundScheduleUpdate:                   "discord",
}

// getStreamForTopic returns the appropriate standard stream name for a topic
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

	// Use shared environment instead of creating new one
	env := getSharedTestEnv(t)

	// Only do lightweight cleanup between tests, not container recreation
	testCtx, testCancel := context.WithCancel(env.Ctx)

	// Standard stream names
	standardStreamNames := []string{"user", "discord", "leaderboard", "round", "score", "delayed"}

	// Only clean JetStream state and tables, not containers
	if err := env.ResetJetStreamState(testCtx, standardStreamNames...); err != nil {
		testCancel()
		t.Fatalf("Failed to clean NATS JetStream state: %v", err)
	}

	// Only truncate tables, don't recreate containers
	if err := testutils.TruncateTables(testCtx, env.DB, "users", "scores", "leaderboards", "rounds"); err != nil {
		testCancel()
		t.Fatalf("Failed to truncate DB tables: %v", err)
	}

	// Create shared resources with minimal logging
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	watermillLogger := watermill.NopLogger{}

	// Create EventBus
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

	// Ensure all required streams exist
	for _, streamName := range standardStreamNames {
		if err := eventBusImpl.CreateStream(testCtx, streamName); err != nil {
			eventBusImpl.Close()
			testCancel()
			t.Fatalf("Failed to create stream %s: %v", streamName, err)
		}
	}

	// Create router with MUCH shorter timeouts for tests
	routerConfig := message.RouterConfig{
		CloseTimeout: 50 * time.Millisecond, // Very short for tests
	}
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
		roundevents.DiscordRoundFinalized,
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
		roundevents.DiscordRoundReminder,
		roundevents.RoundError,
		roundevents.RoundRetrieved,
		roundevents.RoundScoreUpdateError,
		roundevents.RoundScoreUpdateValidated,
		roundevents.RoundParticipantScoreUpdated,
		roundevents.RoundNotAllScoresSubmitted,
		roundevents.DiscordRoundStarted,
		roundevents.RoundUpdateValidated,
		roundevents.RoundUpdated,
		roundevents.RoundScheduleUpdate,
	}

	messageCapture := testutils.NewMessageCapture(captureTopics...)

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

	// Generate unique test ID for consumer names
	testID := fmt.Sprintf("%s-%d",
		strings.ReplaceAll(strings.ReplaceAll(t.Name(), "/", "-"), " ", "-"),
		time.Now().UnixNano())

	// Start message capture consumers
	consumerWg := startMessageCapture(t, testCtx, eventBusImpl, messageCapture, captureTopics, testID)

	// Start the router in background but with aggressive termination
	routerDone := make(chan struct{})
	go func() {
		defer close(routerDone)

		// Create a separate context for the router with even shorter timeout
		routerCtx, routerCancel := context.WithCancel(testCtx)
		defer routerCancel()

		// Run router but exit quickly on any error
		if runErr := watermillRouter.Run(routerCtx); runErr != nil {
			if runErr != context.Canceled && !strings.Contains(runErr.Error(), "context canceled") {
				log.Printf("Router error (ignoring for test): %v", runErr)
			}
		}
	}()

	// Wait a tiny bit for router to initialize
	time.Sleep(100 * time.Millisecond)

	// LIGHTWEIGHT cleanup function - only clean up test-specific resources
	cleanup := func() {
		log.Println("Starting lightweight round handler cleanup...")

		// Step 1: Cancel test context (not shared environment context)
		testCancel()

		// Step 2: Close test-specific resources only
		if watermillRouter != nil {
			watermillRouter.Close()
		}

		// Step 3: Wait only very briefly for cleanup
		cleanupDone := make(chan struct{})
		go func() {
			defer close(cleanupDone)

			// Wait for consumers first (they should exit quickly)
			if consumerWg != nil {
				consumerWg.Wait()
			}

			// Close module
			if roundModule != nil {
				roundModule.Close()
			}

			// Close event bus (test-specific instance)
			if eventBusImpl != nil {
				eventBusImpl.Close()
			}
		}()

		// VERY short timeout - if cleanup doesn't finish, just exit
		select {
		case <-cleanupDone:
			log.Println("Lightweight cleanup completed successfully")
		case <-time.After(200 * time.Millisecond):
			log.Println("Lightweight cleanup timeout - forcing exit (this is OK for tests)")
			// Don't wait any longer - just let the test exit
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
		testID:            testID,
		cleanup:           cleanup,
	}
}

// startMessageCapture starts consumer goroutines with better error handling
// Replace the startMessageCapture function with this improved version

// startMessageCapture starts consumer goroutines with proper cancellation handling
func startMessageCapture(t *testing.T, ctx context.Context, eventBusImpl eventbus.EventBus, messageCapture *testutils.MessageCapture, captureTopics []string, testID string) *sync.WaitGroup {
	wg := &sync.WaitGroup{}

	for i, topic := range captureTopics {
		wg.Add(1)

		go func(topicName string, index int) {
			defer wg.Done()

			// Check for early cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			consumerName := fmt.Sprintf("test-capture-%s-%s-%d-%d",
				strings.ReplaceAll(topicName, ".", "-"),
				testID,
				index,
				time.Now().UnixNano())

			js := eventBusImpl.GetJetStream()
			streamName := getStreamForTopic(topicName)

			// Create consumer with timeout and cancellation awareness
			var consumer jetstream.Consumer
			var err error

			createCtx, createCancel := context.WithTimeout(ctx, 5*time.Second)
			defer createCancel()

			for attempts := 0; attempts < 3; attempts++ {
				select {
				case <-ctx.Done():
					return
				default:
				}

				consumer, err = js.CreateConsumer(createCtx, streamName, jetstream.ConsumerConfig{
					Name:          consumerName,
					FilterSubject: topicName,
					AckPolicy:     jetstream.AckExplicitPolicy,
				})
				if err == nil {
					break
				}

				if !strings.Contains(err.Error(), "context canceled") {
					t.Logf("Attempt %d: Failed to create consumer for %s: %v", attempts+1, topicName, err)
				}

				if attempts < 2 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Duration(attempts+1) * 50 * time.Millisecond):
					}
				}
			}

			if err != nil {
				if !strings.Contains(err.Error(), "context canceled") {
					t.Logf("Failed to create consumer for %s: %v", topicName, err)
				}
				return
			}

			// Message processing loop with aggressive cancellation checking
			ticker := time.NewTicker(25 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					// Context canceled - exit immediately
					return
				case <-ticker.C:
					// Try to fetch messages with a very short timeout
					_, fetchCancel := context.WithTimeout(ctx, 10*time.Millisecond)
					msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(5*time.Millisecond))
					fetchCancel()

					if err != nil {
						if err == jetstream.ErrNoMessages {
							continue
						}
						if !strings.Contains(err.Error(), "context canceled") {
							t.Logf("Error fetching messages for %s: %v", topicName, err)
						}
						continue
					}

					// Process messages but check for cancellation frequently
					for msg := range msgs.Messages() {
						select {
						case <-ctx.Done():
							msg.Ack() // Ack the message before exiting
							return
						default:
						}

						msgID := getMessageID(msg)
						watermillMsg := message.NewMessage(msgID, msg.Data())

						// Copy headers
						if headers := msg.Headers(); headers != nil {
							for k, v := range headers {
								if len(v) > 0 {
									watermillMsg.Metadata.Set(k, v[0])
								}
							}
						}

						// Capture message (non-blocking)
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
