package modules

import (
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/score"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// ModuleRegistry holds all the modules for the application
type ModuleRegistry struct {
	UserModule        *user.UserModule
	RoundModule       *round.RoundModule
	LeaderboardModule *leaderboard.LeaderboardModule
	ScoreModule       *score.ScoreModule
	PubSub            watermillutil.PubSuber
}

// NewModuleRegistry initializes and returns a new ModuleRegistry with all modules registered
func NewModuleRegistry(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*ModuleRegistry, error) {
	userModule, err := user.NewUserModule(dbService, commandBus, pubsub)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user module: %w", err)
	}

	roundModule, err := round.NewRoundModule(dbService, commandBus, pubsub)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize round module: %w", err)
	}

	leaderboardModule, err := leaderboard.NewLeaderboardModule(dbService, commandBus, pubsub)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}

	scoreModule, err := score.NewScoreModule(dbService, commandBus, pubsub)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize score module: %w", err)
	}

	return &ModuleRegistry{
		UserModule:        userModule,
		RoundModule:       roundModule,
		LeaderboardModule: leaderboardModule,
		ScoreModule:       scoreModule,
		PubSub:            pubsub,
	}, nil
}
