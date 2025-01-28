package roundrouter

import (
	"context"

	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Router interface for user routing.
type Router interface {
	Configure(router *message.Router, roundService roundservice.Service, subscriber message.Subscriber) error
	Run(ctx context.Context) error
	Close() error
}
