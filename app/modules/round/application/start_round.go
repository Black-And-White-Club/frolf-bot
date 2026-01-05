package roundservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessRoundStart handles the start of a round, updates participant data, updates DB, and notifies Discord.
// Multi-guild: require guildID for all round operations
func (s *RoundService) ProcessRoundStart(ctx context.Context, payload roundevents.RoundStartedPayloadV1) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ProcessRoundStart", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing round start",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("guild_id", string(payload.GuildID)),
		)

		// Fetch the round from DB
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get round from database",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("guild_id", string(payload.GuildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil
		}

		// Update the round state to "in progress" - call DB method directly
		err = s.RoundDB.UpdateRoundState(ctx, payload.GuildID, payload.RoundID, roundtypes.RoundStateInProgress)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round state to in progress",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("guild_id", string(payload.GuildID)),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "UpdateRoundState")
			return RoundOperationResult{
				Failure: &roundevents.RoundErrorPayloadV1{
					GuildID: payload.GuildID,
					RoundID: payload.RoundID,
					Error:   err.Error(),
				},
			}, nil
		}

		// Convert []roundtypes.Participant to []roundevents.RoundParticipantV1
		participants := make([]roundevents.RoundParticipantV1, len(round.Participants))
		for i, p := range round.Participants {
			participants[i] = roundevents.RoundParticipantV1{
				UserID:    sharedtypes.DiscordID(p.UserID),
				TagNumber: p.TagNumber,
				Response:  roundtypes.Response(p.Response),
				Score:     p.Score,
			}
		}

		// Use the payload data for the Discord event (not the DB data)
		discordPayload := &roundevents.DiscordRoundStartPayloadV1{
			GuildID:        payload.GuildID,
			RoundID:        round.ID,
			Title:          payload.Title,        // Use payload title
			Location:       payload.Location,     // Use payload location
			StartTime:      payload.StartTime,    // Use payload start time
			Participants:   participants,         // Use DB participants (current state)
			EventMessageID: round.EventMessageID, // Use DB event message ID
		}

		s.logger.InfoContext(ctx, "Round start processed",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("guild_id", string(payload.GuildID)),
			attr.Int("participant_count", len(participants)),
		)

		return RoundOperationResult{
			Success: discordPayload,
		}, nil
	})
}
