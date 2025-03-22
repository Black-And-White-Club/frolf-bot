package leaderboardrouter

import (
	"context"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Router interface for user routing.
type Router interface {
	Configure(router *message.Router, leaderboardService leaderboardservice.Service, subscriber eventbus.EventBus) error
	Run(ctx context.Context) error
	Close() error
}
