package userrouter

import (
	"context"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Router interface for user routing.
type Router interface {
	Configure(router *message.Router, userService userservice.Service, subscriber message.Subscriber) error
	Run(ctx context.Context) error
	Close() error
}
