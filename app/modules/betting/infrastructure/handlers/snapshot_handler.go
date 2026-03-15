package bettinghandlers

import (
	"context"
	"fmt"
	"time"

	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	"github.com/google/uuid"
)

// HandleBettingSnapshotRequest responds to NATS request/reply calls on
// betting.snapshot.request.v1.{club_uuid}. It returns the public market
// snapshot for the club — no wallet or per-user bet data is included.
func (h *EventHandlers) HandleBettingSnapshotRequest(
	ctx context.Context,
	payload *bettingevents.BettingSnapshotRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	clubUUID, err := uuid.Parse(payload.ClubUUID)
	if err != nil {
		return nil, fmt.Errorf("betting snapshot: invalid club_uuid %q: %w", payload.ClubUUID, err)
	}

	snapshot, err := h.service.GetMarketSnapshot(ctx, clubUUID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic: bettingevents.BettingSnapshotResponseV1,
			Payload: &bettingevents.BettingSnapshotResponsePayloadV1{
				ClubUUID: payload.ClubUUID,
				Error:    err.Error(),
			},
		}}, nil
	}

	resp := &bettingevents.BettingSnapshotResponsePayloadV1{
		ClubUUID:    snapshot.ClubUUID,
		GuildID:     snapshot.GuildID,
		SeasonID:    snapshot.SeasonID,
		AccessState: snapshot.AccessState,
	}

	if snapshot.Round != nil {
		resp.Round = &bettingevents.BettingSnapshotRound{
			ID:        snapshot.Round.ID,
			Title:     snapshot.Round.Title,
			StartTime: snapshot.Round.StartTime.Format(time.RFC3339),
		}
	}

	buildSnapshotMarket := func(m *bettingservice.BettingMarket) *bettingevents.BettingSnapshotMarket {
		options := make([]bettingevents.BettingSnapshotOption, 0, len(m.Options))
		for _, o := range m.Options {
			options = append(options, bettingevents.BettingSnapshotOption{
				OptionKey:          o.OptionKey,
				MemberID:           o.MemberID,
				Label:              o.Label,
				ProbabilityPercent: o.ProbabilityPercent,
				DecimalOdds:        o.DecimalOdds,
				Metadata:           o.Metadata,
			})
		}
		return &bettingevents.BettingSnapshotMarket{
			ID:        m.ID,
			Type:      m.Type,
			Title:     m.Title,
			Status:    m.Status,
			LocksAt:   m.LocksAt.Format(time.RFC3339),
			Ephemeral: m.Ephemeral,
			Result:    m.Result,
			Options:   options,
		}
	}

	if snapshot.Market != nil {
		resp.Market = buildSnapshotMarket(snapshot.Market)
	}

	if len(snapshot.Markets) > 0 {
		resp.Markets = make([]bettingevents.BettingSnapshotMarket, 0, len(snapshot.Markets))
		for i := range snapshot.Markets {
			sm := buildSnapshotMarket(&snapshot.Markets[i])
			resp.Markets = append(resp.Markets, *sm)
		}
	}

	// Honour reply-to inbox for request/reply pattern (same as leaderboard).
	topic := bettingevents.BettingSnapshotResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{{Topic: topic, Payload: resp}}, nil
}
