package bettingservice

import (
	"log/slog"

	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace"
)

// BettingService is the primary application-layer service for the betting
// module. Its public methods implement the Service interface; its private
// helpers are split across focused files by domain concern.
type BettingService struct {
	repo            bettingRepository
	userRepo        userRepository
	guildRepo       guildRepository
	leaderboardRepo leaderboardRepository
	roundRepo       roundRepository
	metrics         bettingmetrics.BettingMetrics
	logger          *slog.Logger
	tracer          trace.Tracer
	db              *bun.DB
	oddsEngine      *oddsEngine
}

func NewService(
	repo bettingRepository,
	userRepo userRepository,
	guildRepo guildRepository,
	leaderboardRepo leaderboardRepository,
	roundRepo roundRepository,
	metrics bettingmetrics.BettingMetrics,
	logger *slog.Logger,
	tracer trace.Tracer,
	db *bun.DB,
) *BettingService {
	return &BettingService{
		repo:            repo,
		userRepo:        userRepo,
		guildRepo:       guildRepo,
		leaderboardRepo: leaderboardRepo,
		roundRepo:       roundRepo,
		metrics:         metrics,
		logger:          logger,
		tracer:          tracer,
		db:              db,
		oddsEngine:      newOddsEngine(roundRepo, leaderboardRepo),
	}
}

// compile-time interface check
var _ Service = (*BettingService)(nil)
