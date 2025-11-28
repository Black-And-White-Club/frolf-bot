package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	websocketmodule "github.com/Black-And-White-Club/frolf-bot/app/modules/websocket"
	"github.com/Black-And-White-Club/frolf-bot/config"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load config
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize observability
	obsConfig := observability.Config{
		ServiceName:    "frolf-bot-websocket",
		Environment:    "development", // TODO: from config
		Version:        "1.0.0",
		LokiURL:        cfg.Observability.LokiURL,
		MetricsAddress: ":9090",
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}

	logger := obs.Provider.Logger
	logger.Info("Starting WebSocket server")

	// Initialize event bus
	tracer := obs.Provider.TracerProvider.Tracer("websocket")
	eventBus, err := eventbus.NewEventBus(
		ctx,
		cfg.NATS.URL,
		logger,
		"websocket",
		obs.Registry.EventBusMetrics,
		tracer,
	)
	if err != nil {
		log.Fatalf("Failed to create event bus: %v", err)
	}
	defer eventBus.Close()

	// Create WebSocket module
	wsModule, err := websocketmodule.NewWebSocketModule(ctx, cfg, obs, eventBus)
	if err != nil {
		log.Fatalf("Failed to create WebSocket module: %v", err)
	}

	// Start module
	if err := wsModule.Start(ctx); err != nil {
		log.Fatalf("Failed to start WebSocket module: %v", err)
	}

	logger.Info("WebSocket server started successfully")

	// Wait for interrupt
	<-ctx.Done()
	logger.Info("Shutting down WebSocket server")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := wsModule.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}

	logger.Info("WebSocket server stopped")
}
