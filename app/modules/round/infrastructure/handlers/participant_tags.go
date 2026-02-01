package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScheduledRoundTagSync processes tag updates emitted by the Leaderboard Funnel.
// It synchronizes upcoming rounds with the new source-of-truth tags from the leaderboard.
func (h *RoundHandlers) HandleScheduledRoundTagSync(
	ctx context.Context,
	payload *sharedevents.SyncRoundsTagRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Processing leaderboard tag sync request for rounds",
		attr.String("guild_id", string(payload.GuildID)),
		attr.Int("tags_changed", len(payload.ChangedTags)),
	)

	// 1. Update the Round module's database
	result, err := h.service.UpdateScheduledRoundsWithNewTags(
		ctx,
		&roundtypes.UpdateScheduledRoundsWithNewTagsRequest{
			GuildID:     payload.GuildID,
			ChangedTags: payload.ChangedTags,
		},
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "Internal error during round tag sync", attr.Error(err))
		return nil, err
	}

	// 2. Publish the result to trigger Discord updates
	// Success Topic: roundevents.ScheduledRoundsSyncedV1
	return mapOperationResult(result.Map(
		func(s *roundtypes.ScheduledRoundsSyncResult) any {
			updatedRounds := make([]roundevents.RoundUpdateInfoV1, len(s.Updates))
			totalParticipantsUpdated := 0
			for i, u := range s.Updates {
				totalParticipantsUpdated += u.ParticipantsChangedCount
				roundInfo := roundevents.RoundUpdateInfoV1{
					GuildID:             s.GuildID,
					RoundID:             u.RoundID,
					EventMessageID:      u.EventMessageID,
					UpdatedParticipants: u.Participants,
					ParticipantsChanged: u.ParticipantsChangedCount,
				}
				if u.Round != nil {
					roundInfo.Title = u.Round.Title
					roundInfo.StartTime = u.Round.StartTime
					roundInfo.Location = u.Round.Location
				}
				updatedRounds[i] = roundInfo
			}

			return &roundevents.ScheduledRoundsSyncedPayloadV1{
				GuildID:       s.GuildID,
				UpdatedRounds: updatedRounds,
				Summary: roundevents.UpdateSummaryV1{
					GuildID:              s.GuildID,
					TotalRoundsProcessed: s.TotalChecked,
					RoundsUpdated:        len(s.Updates),
					ParticipantsUpdated:  totalParticipantsUpdated,
				},
			}
		},
		func(f error) any { return f },
	),
		roundevents.ScheduledRoundsSyncedV1,
		roundevents.RoundUpdateErrorV1,
	), nil
}
