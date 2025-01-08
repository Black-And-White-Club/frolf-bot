package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// App holds the application components.
type App struct {
	Config     *config.Config
	Logger     watermill.LoggerAdapter
	Router     *message.Router
	UserModule *user.Module
	DB         *bundb.DBService
}

// Initialize initializes the application.
func (app *App) Initialize(ctx context.Context) error {
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	fmt.Printf("Loaded Config: %+v\n", cfg)
	app.Config = cfg

	app.Logger = watermill.NewStdLogger(false, false)

	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	streamCreator, err := adapters.NewStreamCreator(cfg.NATS.URL, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to create stream creator: %w", err)
	}
	defer streamCreator.Close()

	streams := map[string][]struct {
		ConsumerName string
		Subject      string
	}{
		"user": {
			{
				ConsumerName: "user_signup_consumer",
				Subject:      userevents.UserSignupRequestSubject,
			},
			{
				ConsumerName: "user_role_update_consumer",
				Subject:      userevents.UserRoleUpdateRequestSubject,
			},
		},
	}

	for streamName, consumers := range streams {
		if err := streamCreator.CreateStream(streamName); err != nil {
			return fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}

		for _, consumer := range consumers {
			if err := streamCreator.CreateConsumer(streamName, consumer.ConsumerName, consumer.Subject); err != nil {
				return fmt.Errorf("failed to create consumer %s for stream %s and subject %s: %w", consumer.ConsumerName, streamName, consumer.Subject, err)
			}
		}
	}

	router, err := message.NewRouter(message.RouterConfig{}, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}

	// Add retry middleware with backoff for resilience
	router.AddMiddleware(middleware.Retry{
		MaxRetries:      3,
		InitialInterval: time.Millisecond * 100,
		Multiplier:      2, // Exponential backoff
		MaxInterval:     time.Second * 5,
	}.Middleware)

	app.Router = router

	userModule, err := user.NewUserModule(ctx, cfg, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	app.UserModule = userModule

	return nil
}

// Run starts the application.
func (app *App) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	// Create a new context for the router that is derived from the main context
	routerCtx, routerCancel := context.WithCancel(ctx)

	go func() {
		if err := app.Router.Run(routerCtx); err != nil {
			log.Printf("Error running Watermill router: %v", err)
			routerCancel() // Crucial: Cancel the router context on error
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timeoutCtx, timeoutCancel := context.WithTimeout(routerCtx, 10*time.Second)
		defer timeoutCancel()

		for {
			select {
			case <-timeoutCtx.Done():
				log.Fatalf("Timeout waiting for subscribers to initialize")
				return
			case <-time.After(100 * time.Millisecond):
				// Check if both user and leaderboard modules are initialized
				if app.UserModule != nil && app.UserModule.IsInitialized() {
					// && app.LeaderboardModule != nil && app.LeaderboardModule.IsInitialized() {
					fmt.Println("User module initialized")
					return
				}
			}
		}
	}()

	wg.Wait()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-interrupt:
		fmt.Println("Shutting down gracefully (interrupt signal received)...")
	case <-routerCtx.Done(): // Router context cancelled (due to error or shutdown)
		fmt.Println("Shutting down gracefully (router context done)...")
	}

	if err := app.Router.Close(); err != nil {
		log.Printf("Error closing Watermill router: %v", err)
	}

	app.Close()

	fmt.Println("Graceful shutdown complete.")

	return nil
}

// Close gracefully shuts down the application.
func (app *App) Close() {
	if app.Router != nil {
		if err := app.Router.Close(); err != nil {
			log.Printf("Error closing Watermill router: %v", err)
		}
	}

	if app.UserModule != nil {
		if err := app.UserModule.Publisher.Close(); err != nil {
			log.Printf("Error closing user module publisher: %v", err)
		}
		if err := app.UserModule.Subscriber.Close(); err != nil {
			log.Printf("Error closing user module subscriber: %v", err)
		}
	}

	// if app.RoundModule != nil {
	// 	if err := app.RoundModule.Publisher.Close(); err != nil {
	// 		log.Printf("Error closing round module publisher: %v", err)
	// 	}
	// 	if err := app.RoundModule.Subscriber.Close(); err != nil {
	// 		log.Printf("Error closing round module subscriber: %v", err)
	// 	}
	// }

	// if app.ScoreModule != nil {
	// 	if err := app.ScoreModule.Publisher.Close(); err != nil {
	// 		log.Printf("Error closing score module publisher: %v", err)
	// 	}
	// 	if err := app.ScoreModule.Subscriber.Close(); err != nil {
	// 		log.Printf("Error closing score module subscriber: %v", err)
	// 	}
	// }

	// if app.LeaderboardModule != nil { // Close connections for the leaderboard module
	// 	if err := app.LeaderboardModule.Publisher.Close(); err != nil {
	// 		log.Printf("Error closing leaderboard module publisher: %v", err)
	// 	}
	// 	if err := app.LeaderboardModule.Subscriber.Close(); err != nil {
	// 		log.Printf("Error closing leaderboard module subscriber: %v", err)
	// 	}
	// }
}
