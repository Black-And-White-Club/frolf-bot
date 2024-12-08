package app

import (
	"context"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	eventbus "github.com/Black-And-White-Club/tcr-bot/eventbus"
	"github.com/Black-And-White-Club/tcr-bot/nats"
	roundcommands "github.com/Black-And-White-Club/tcr-bot/round/commands"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/round/queries"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type App struct {
	Cfg                *config.Config
	db                 *bundb.DBService
	NatsConnectionPool *nats.NatsConnectionPool
	// LeaderboardService *leaderboard.LeaderboardService
	// UserService        *user.UserService
	RoundService      roundcommands.CommandService
	RoundQueryService roundqueries.RoundQueryService
	// ScoreService       *score.ScoreService
	messagePublisher message.Publisher
}

// NewApp initializes the application with the necessary services and configuration.
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.NewConfig(ctx)
	dsn := cfg.DSN
	natsURL := cfg.NATS.URL

	dbService, err := bundb.NewBunDBService(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	natsConnectionPool, err := nats.NewNatsConnectionPool(natsURL, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NATS connection pool: %w", err)
	}

	log.Printf("NATS connection pool initialized with URL: %s", natsURL)

	publisher, err := eventbus.NewPublisher(natsURL, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	eventbus.InitPublisher(publisher)

	// leaderboardService := leaderboard.NewLeaderboardService(dbService.Leaderboard, natsConnectionPool, publisher)
	// userService := user.NewUserService(dbService.User, natsConnectionPool, publisher)
	roundService := roundcommands.NewRoundCommandService(dbService.Round, publisher, eventHandler) // Pass the event handler to NewRoundCommandService
	roundQueryService := roundqueries.NewRoundQueryService(dbService.Round)

	// scoreService := score.NewScoreService(dbService.Score, natsConnectionPool, publisher)

	return &App{
		Cfg:                cfg,
		db:                 dbService,
		NatsConnectionPool: natsConnectionPool,
		// LeaderboardService: leaderboardService,
		// UserService:        userService,
		RoundService:      roundService,
		RoundQueryService: roundQueryService,
		// ScoreService:       scoreService,
		messagePublisher: publisher,
	}, nil
}

// DB returns the database service.
func (app *App) DB() *bundb.DBService {
	return app.db
}

// Publisher returns the publisher.
func (app *App) Publisher() message.Publisher {
	return app.messagePublisher
}
