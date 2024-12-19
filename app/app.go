package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/round"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/nats-io/nats.go"
)

// App holds the application components.
type App struct {
	Config      *config.Config
	Logger      watermill.LoggerAdapter
	NATS        *nats.Conn
	JetStream   nats.JetStreamContext
	Router      *message.Router
	PubSub      *gochannel.GoChannel
	UserModule  *user.Module
	RoundModule *round.Module
	// ... other modules
}

// Initialize initializes the application.
func (app *App) Initialize(ctx context.Context) error {
	// 1. Load configuration
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	app.Config = cfg

	// 2. Initialize logger
	app.Logger = watermill.NewStdLogger(false, false)

	// 3. Initialize NATS connection
	natsConn, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	app.NATS = natsConn

	js, err := natsConn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	app.JetStream = js

	// 4. Initialize Watermill router and publisher
	router, err := message.NewRouter(message.RouterConfig{}, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}

	retryMiddleware := middleware.Retry{
		MaxRetries:      3,
		InitialInterval: time.Millisecond * 100,
		// ... other retry options if needed ...
	}
	router.AddMiddleware(
		middleware.CorrelationID,
		retryMiddleware.Middleware,
	)

	app.Router = router

	// 5. Initialize modules
	userModule, err := user.Init(ctx, js)
	if err != nil {
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	app.UserModule = userModule

	roundModule, err := round.Init(ctx, js)
	if err != nil {
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	app.RoundModule = roundModule

	// ... initialize other modules ...

	return nil
}

// Run starts the application.
func (app *App) Run(ctx context.Context) error {
	// Start the router
	go func() {
		if err := app.Router.Run(ctx); err != nil {
			log.Fatalf("Error running Watermill router: %v", err)
		}
	}()

	// ... other startup logic ...

	return nil
}

// Close gracefully shuts down the application.
func (app *App) Close() {
	// Close the Watermill router
	if err := app.Router.Close(); err != nil {
		log.Printf("Error closing Watermill router: %v", err)
	}

	// Close the GoChannel pub/sub
	if err := app.PubSub.Close(); err != nil {
		log.Printf("Error closing GoChannel pub/sub: %v", err)
	}

	// Close the NATS connection
	app.NATS.Close()

	// ... other cleanup logic ...
}
