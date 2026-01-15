package userhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers interface defines the contract for user-related message handlers.
type Handlers interface {
	HandleUserSignupRequest(ctx context.Context, payload *userevents.UserSignupRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleUserRoleUpdateRequest(ctx context.Context, payload *userevents.UserRoleUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGetUserRequest(ctx context.Context, payload *userevents.GetUserRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGetUserRoleRequest(ctx context.Context, payload *userevents.GetUserRoleRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagUnavailable(ctx context.Context, payload *sharedevents.TagUnavailablePayloadV1) ([]handlerwrapper.Result, error)
	HandleTagAvailable(ctx context.Context, payload *sharedevents.TagAvailablePayloadV1) ([]handlerwrapper.Result, error)
	HandleUpdateUDiscIdentityRequest(ctx context.Context, payload *userevents.UpdateUDiscIdentityRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScorecardParsed(ctx context.Context, payload *roundevents.ParsedScorecardPayloadV1) ([]handlerwrapper.Result, error)
}
