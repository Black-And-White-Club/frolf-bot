package user

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
)

// Module represents the user module.
type Module struct {
	EventBus           eventbus.EventBus
	UserService        userservice.Service
	config             *config.Config
	UserRouter         *userrouter.UserRouter
	cancelFunc         context.CancelFunc // Stored cancel function for the module's Run context
	Helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry // Prometheus registry for module-specific metrics
}

// NewUserModule creates a new instance of the User module.
// It initializes the user service, router, and configures the router.
// It now accepts a routerCtx to be used for the router's run context.
func NewUserModule(
	ctx context.Context, // Context for the module's lifecycle (e.g., from main)
	cfg *config.Config,
	obs observability.Observability,
	userDB userdb.UserDB,
	eventBus eventbus.EventBus,
	router *message.Router, // The shared Watermill router instance
	helpers utils.Helpers,
	routerCtx context.Context, // Context specifically for the router's Run method
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.UserMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "user.NewUserModule called")

	// Initialize user service
	userService := userservice.NewUserService(userDB, eventBus, logger, metrics, tracer)

	// Create a new Prometheus Registry for this module's router and metrics.
	// This registry will be used by the router's metrics builder.
	prometheusRegistry := prometheus.NewRegistry()

	// Initialize user router with observability and the Prometheus Registry.
	// The router will conditionally add metrics based on environment.
	userRouter := userrouter.NewUserRouter(
		logger,
		router,
		eventBus, // Subscriber
		eventBus, // Publisher
		cfg,
		helpers,
		tracer,
		prometheusRegistry, // Pass the module's Prometheus registry
	)

	// Configure the router with the user service, passing the routerCtx.
	// The router will use this context for its internal operations, including AddHandler contexts.
	if err := userRouter.Configure(routerCtx, userService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}

	// Create the module instance
	module := &Module{
		EventBus:           eventBus,
		UserService:        userService,
		config:             cfg,
		UserRouter:         userRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry, // Assign the created registry
	}

	return module, nil
}

// Run starts the user module.
// It creates a cancellable context for the module's operations.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting user module")

	// Create a context that can be canceled to signal the module to stop
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel // Store the cancel function
	defer cancel()        // Ensure cancel is called when Run exits

	// If a wait group is provided, signal that this goroutine is done when it exits
	if wg != nil {
		defer wg.Done()
	}

	// Keep this goroutine alive until the context is canceled
	<-ctx.Done()
	logger.InfoContext(ctx, "User module goroutine stopped")
}

// Close stops the user module and cleans up resources.
// It cancels the module's context and explicitly closes the router.
func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping user module")

	// Cancel the module's context to signal running goroutines to stop
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// Explicitly close the user router. This will also close its associated subscribers/publishers.
	if m.UserRouter != nil {
		logger.Info("Closing UserRouter from module")
		if err := m.UserRouter.Close(); err != nil {
			logger.Error("Error closing UserRouter from module", "error", err)
			return fmt.Errorf("error closing UserRouter: %w", err)
		}
		logger.Info("UserRouter closed successfully")
	}

	logger.Info("User module stopped")
	return nil
}
