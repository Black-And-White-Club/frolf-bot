package bettinghandlers

import (
	"context"

	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

type Handlers interface {
	HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundDeleted(ctx context.Context, payload *roundevents.RoundDeletedPayloadV1) ([]handlerwrapper.Result, error)
	HandleBettingSnapshotRequest(ctx context.Context, payload *bettingevents.BettingSnapshotRequestPayloadV1) ([]handlerwrapper.Result, error)
	// HandleFeatureAccessUpdated suspends open markets when a club's betting
	// entitlement transitions to frozen or disabled.
	HandleFeatureAccessUpdated(ctx context.Context, payload *guildevents.GuildFeatureAccessUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	// HandlePointsAwarded mirrors season-point deltas into the betting wallet
	// journal for all awarded players. Idempotent — duplicate round events are
	// no-ops at the DB layer.
	HandlePointsAwarded(ctx context.Context, payload *sharedevents.PointsAwardedPayloadV1) ([]handlerwrapper.Result, error)
}
