package userrouter

import (
	"context"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Router interface {
	Configure(router *message.Router, userService userservice.Service) error
	Run(ctx context.Context) error
	Close() error
}
