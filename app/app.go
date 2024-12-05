package app

import (
	"context"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app/config"
	"github.com/Black-And-White-Club/tcr-bot/app/services"
	"github.com/Black-And-White-Club/tcr-bot/internal/db/bundb"
	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
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

	log.Printf("NATS connection pool initialized with URL: %s", natsURL) // Log the NATS URL

	// Initialize services with the correct types
	leaderboardService := services.NewLeaderboardService(dbService.Leaderboard, natsConnectionPool)
	userService := services.NewUserService(dbService.User, natsConnectionPool)
	roundService := services.NewRoundService(dbService.Round, natsConnectionPool) // Removed leaderboardService
	scoreService := services.NewScoreService(dbService.Score, natsConnectionPool)

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
