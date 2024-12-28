package userinterfaces

import (
	"context"

	types "github.com/Black-And-White-Club/tcr-bot/app/types"
)

type Handlers interface {
	HandleUserSignupRequest(ctx context.Context, msg types.Message) error
	HandleUserRoleUpdateRequest(ctx context.Context, msg types.Message) error
}
