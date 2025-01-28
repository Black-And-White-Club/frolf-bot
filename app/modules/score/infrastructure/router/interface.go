package scorerouter

import (
	"context"

	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Router interface for user routing.
type Router interface {
	Configure(router *message.Router, scoreService scoreservice.Service, subscriber message.Subscriber) error
	Run(ctx context.Context) error
	Close() error
}
