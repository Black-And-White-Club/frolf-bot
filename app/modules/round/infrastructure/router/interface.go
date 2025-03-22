package roundrouter

import (
	"context"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Router interface for user routing.
type Router interface {
	Configure(router *message.Router, roundService roundservice.Service, subscriber eventbus.EventBus) error
	Run(ctx context.Context) error
	Close() error
}
