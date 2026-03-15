package bettinghandlers

import (
	"context"
	"fmt"
	"time"

	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
)

type EventHandlers struct {
	service bettingservice.Service
	metrics bettingmetrics.BettingMetrics
}

func NewEventHandlers(service bettingservice.Service, metrics bettingmetrics.BettingMetrics) *EventHandlers {
	return &EventHandlers{service: service, metrics: metrics}
}

func (h *EventHandlers) HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedPayloadV1) ([]handlerwrapper.Result, error) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(ctx, "HandleRoundFinalized")

	results, err := h.service.SettleRound(
		ctx,
		payload.GuildID,
		&bettingservice.BettingSettlementRound{
			ID:           payload.RoundID,
			Title:        payload.RoundData.Title.String(),
			GuildID:      payload.GuildID,
			Finalized:    bool(payload.RoundData.Finalized) || payload.RoundData.State == roundtypes.RoundStateFinalized,
			Participants: toSettlementParticipants(payload.RoundData.Participants),
		},
		roundevents.RoundFinalizedV2,
		nil,
		"round finalized",
	)
	if err != nil {
		h.metrics.RecordHandlerFailure(ctx, "HandleRoundFinalized")
		return nil, err
	}

	out := make([]handlerwrapper.Result, 0, len(results)*2)
	for _, r := range results {
		payload := bettingevents.BettingMarketSettledPayloadV1{
			GuildID:           r.GuildID,
			ClubUUID:          r.ClubUUID,
			RoundID:           r.RoundID,
			MarketID:          r.MarketID,
			ResultSummary:     r.ResultSummary,
			SettlementVersion: r.SettlementVersion,
		}
		out = append(out, handlerwrapper.Result{Topic: bettingevents.BettingMarketSettledV1, Payload: payload})
		if r.ClubUUID != "" {
			out = append(out, handlerwrapper.Result{
				Topic:   fmt.Sprintf("%s.%s", bettingevents.BettingMarketSettledV1, r.ClubUUID),
				Payload: payload,
			})
		}
	}

	h.metrics.RecordHandlerSuccess(ctx, "HandleRoundFinalized")
	h.metrics.RecordHandlerDuration(ctx, "HandleRoundFinalized", time.Since(start))
	return out, nil
}

func (h *EventHandlers) HandleRoundDeleted(ctx context.Context, payload *roundevents.RoundDeletedPayloadV1) ([]handlerwrapper.Result, error) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(ctx, "HandleRoundDeleted")

	results, err := h.service.VoidRoundMarkets(ctx, payload.GuildID, payload.RoundID, roundevents.RoundDeletedV2, nil, "round deleted")
	if err != nil {
		h.metrics.RecordHandlerFailure(ctx, "HandleRoundDeleted")
		return nil, err
	}

	out := make([]handlerwrapper.Result, 0, len(results)*2)
	for _, r := range results {
		payload := bettingevents.BettingMarketVoidedPayloadV1{
			GuildID:  r.GuildID,
			ClubUUID: r.ClubUUID,
			RoundID:  r.RoundID,
			MarketID: r.MarketID,
			Reason:   r.Reason,
		}
		out = append(out, handlerwrapper.Result{Topic: bettingevents.BettingMarketVoidedV1, Payload: payload})
		if r.ClubUUID != "" {
			out = append(out, handlerwrapper.Result{
				Topic:   fmt.Sprintf("%s.%s", bettingevents.BettingMarketVoidedV1, r.ClubUUID),
				Payload: payload,
			})
		}
	}

	h.metrics.RecordHandlerSuccess(ctx, "HandleRoundDeleted")
	h.metrics.RecordHandlerDuration(ctx, "HandleRoundDeleted", time.Since(start))
	return out, nil
}

func toSettlementParticipants(participants []roundtypes.Participant) []bettingservice.BettingSettlementParticipant {
	settled := make([]bettingservice.BettingSettlementParticipant, 0, len(participants))
	for _, participant := range participants {
		var score *int
		if participant.Score != nil {
			v := int(*participant.Score)
			score = &v
		}
		settled = append(settled, bettingservice.BettingSettlementParticipant{
			MemberID: string(participant.UserID),
			Response: string(participant.Response),
			Score:    score,
			IsDNF:    participant.IsDNF,
		})
	}
	return settled
}

// HandleFeatureAccessUpdated reacts to guild.feature_access.updated.v1.
// When a club's betting entitlement transitions to frozen or disabled, all
// currently open markets are suspended. Already-accepted bets are unaffected
// and will settle normally when the round finalises.
func (h *EventHandlers) HandleFeatureAccessUpdated(
	ctx context.Context,
	payload *guildevents.GuildFeatureAccessUpdatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(ctx, "HandleFeatureAccessUpdated")

	betting, ok := payload.Entitlements.Feature(guildtypes.ClubFeatureBetting)
	if !ok || betting.State == guildtypes.FeatureAccessStateEnabled {
		// Nothing to do: either no betting entitlement key or access is (still) enabled.
		h.metrics.RecordHandlerSuccess(ctx, "HandleFeatureAccessUpdated")
		h.metrics.RecordHandlerDuration(ctx, "HandleFeatureAccessUpdated", time.Since(start))
		return nil, nil
	}

	results, err := h.service.SuspendOpenMarketsForClub(ctx, payload.GuildID)
	if err != nil {
		h.metrics.RecordHandlerFailure(ctx, "HandleFeatureAccessUpdated")
		return nil, err
	}

	out := make([]handlerwrapper.Result, 0, len(results)*2)
	for _, r := range results {
		payload := bettingevents.BettingMarketSuspendedPayloadV1{
			GuildID:  r.GuildID,
			ClubUUID: r.ClubUUID,
			RoundID:  r.RoundID,
			MarketID: r.MarketID,
		}
		out = append(out, handlerwrapper.Result{Topic: bettingevents.BettingMarketSuspendedV1, Payload: payload})
		if r.ClubUUID != "" {
			out = append(out, handlerwrapper.Result{
				Topic:   fmt.Sprintf("%s.%s", bettingevents.BettingMarketSuspendedV1, r.ClubUUID),
				Payload: payload,
			})
		}
	}

	h.metrics.RecordHandlerSuccess(ctx, "HandleFeatureAccessUpdated")
	h.metrics.RecordHandlerDuration(ctx, "HandleFeatureAccessUpdated", time.Since(start))
	return out, nil
}

// HandlePointsAwarded mirrors season-point deltas into the betting wallet
// journal for all awarded players. Idempotent — duplicate round events are
// no-ops at the DB layer.
func (h *EventHandlers) HandlePointsAwarded(ctx context.Context, payload *sharedevents.PointsAwardedPayloadV1) ([]handlerwrapper.Result, error) {
	start := time.Now()
	h.metrics.RecordHandlerAttempt(ctx, "HandlePointsAwarded")

	if err := h.service.MirrorPointsToWallet(ctx, payload.GuildID, payload.RoundID, payload.Points); err != nil {
		h.metrics.RecordHandlerFailure(ctx, "HandlePointsAwarded")
		return nil, err
	}

	h.metrics.RecordHandlerSuccess(ctx, "HandlePointsAwarded")
	h.metrics.RecordHandlerDuration(ctx, "HandlePointsAwarded", time.Since(start))
	return nil, nil
}
