package app

import (
	"context"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/api/services"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	"github.com/Black-And-White-Club/tcr-bot/nats"
)

type App struct {
	Cfg                *config.Config
	db                 *bundb.DBService
	NatsConnectionPool *nats.NatsConnectionPool
	LeaderboardService *services.LeaderboardService
	UserService        *services.UserService
	RoundService       *services.RoundService
	ScoreService       *services.ScoreService
}

// NewApp initializes the application with the necessary services and configuration.
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.NewConfig(ctx)
	dsn := cfg.DSN
	natsURL := cfg.NATS.URL

	// Initialize the database service
	dbService, err := bundb.NewBunDBService(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	// Initialize NATS connection pool
	natsConnectionPool, err := nats.NewNatsConnectionPool(natsURL, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NATS connection pool: %w", err)
	}

	log.Printf("NATS connection pool initialized with URL: %s", natsURL)

	// Initialize services with the correct types
	leaderboardService := services.NewLeaderboardService(dbService.Leaderboard, natsConnectionPool)
	userService := services.NewUserService(dbService.User, natsConnectionPool)
	roundService := services.NewRoundService(dbService.Round, natsConnectionPool)
	scoreService := services.NewScoreService(dbService.Score, natsConnectionPool)

	// Start the NATS subscribers for each service
	if err := leaderboardService.StartNATSSubscribers(ctx); err != nil {
		return nil, fmt.Errorf("failed to start NATS subscribers for LeaderboardService: %w", err)
	}
	if err := userService.StartNATSSubscribers(ctx); err != nil {
		return nil, fmt.Errorf("failed to start NATS subscribers for UserService: %w", err)
	}
	// if err := roundService.StartNATSSubscribers(ctx); err != nil {
	//   return nil, fmt.Errorf("failed to start NATS subscribers for RoundService: %w", err)
	// }
	// if err := scoreService.StartNATSSubscribers(ctx); err != nil {
	//   return nil, fmt.Errorf("failed to start NATS subscribers for ScoreService: %w", err)
	// }

	return &App{
		Cfg:                cfg,
		db:                 dbService,
		NatsConnectionPool: natsConnectionPool,
		LeaderboardService: leaderboardService,
		UserService:        userService,
		RoundService:       roundService,
		ScoreService:       scoreService,
	}, nil
}

// DB returns the database service.
func (app *App) DB() *bundb.DBService {
	return app.db
}
