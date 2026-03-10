package roundhandlers

import (
	"context"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleAdminBackfillCheckRequested handles the pre-check request-reply to inform
// the PWA of any finalized rounds after the requested backfill date.
func (h *RoundHandlers) HandleAdminBackfillCheckRequested(ctx context.Context, payload *roundevents.AdminBackfillCheckRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	rounds, err := h.service.GetFinalizedRoundsAfter(ctx, payload.GuildID, payload.StartTime)
	if err != nil {
		return nil, err
	}

	titles := make([]string, 0, len(rounds))
	for _, r := range rounds {
		titles = append(titles, string(r.Title))
	}

	response := &roundevents.AdminBackfillCheckResponsePayloadV1{
		SubsequentRoundCount: len(rounds),
		RoundTitles:          titles,
	}

	topic := "round.admin.backfill.check.response.v1"
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{{Topic: topic, Payload: response}}, nil
}

// HandleAdminBackfillRoundRequested validates the admin role, creates a historical round
// in UPCOMING state (no Discord event), then triggers the existing admin import pipeline.
func (h *RoundHandlers) HandleAdminBackfillRoundRequested(ctx context.Context, payload *roundevents.AdminBackfillRoundRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Admin backfill round requested",
		attr.String("guild_id", string(payload.GuildID)),
		attr.String("admin_id", string(payload.AdminID)),
		attr.String("title", payload.Title),
		attr.Time("start_time", payload.StartTime),
	)

	if err := h.ensureAdminRole(ctx, payload.GuildID, payload.AdminID); err != nil {
		return nil, err
	}

	result, err := h.service.StoreHistoricalRound(
		ctx,
		payload.GuildID,
		payload.AdminID,
		roundtypes.Title(payload.Title),
		roundtypes.Location(payload.Location),
		payload.StartTime,
	)
	if err != nil {
		return nil, err
	}
	if result.Failure != nil {
		return nil, *result.Failure
	}

	round := (*result.Success).Round

	importID := payload.ImportID
	if importID == "" {
		importID = uuid.New().String()
	}

	uploadPayload := &roundevents.ScorecardUploadedPayloadV1{
		ImportID:                importID,
		Source:                  adminPwaImportSource,
		GuildID:                 payload.GuildID,
		RoundID:                 round.ID,
		UserID:                  payload.AdminID,
		ChannelID:               "",
		MessageID:               "",
		FileData:                payload.FileData,
		FileName:                payload.FileName,
		Notes:                   payload.Notes,
		AllowGuestPlayers:       true,
		OverwriteExistingScores: true,
		Timestamp:               time.Now().UTC(),
	}

	h.logger.InfoContext(ctx, "Historical round created, triggering import pipeline",
		attr.StringUUID("round_id", round.ID.String()),
		attr.String("import_id", importID),
	)

	return []handlerwrapper.Result{{Topic: roundevents.ScorecardAdminUploadRequestedV2, Payload: uploadPayload}}, nil
}
