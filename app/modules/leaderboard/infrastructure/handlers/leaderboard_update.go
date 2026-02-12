package leaderboardhandlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/google/uuid"
)

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
// This is for score processing after round completion - updates leaderboard with new participant tags.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateRequested(
	ctx context.Context,
	payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	return h.handleLeaderboardUpdateWithServiceCommand(ctx, payload)
}

func (h *LeaderboardHandlers) handleLeaderboardUpdateWithServiceCommand(
	ctx context.Context,
	payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	participants := buildParticipantsFromUpdatePayload(payload)
	if len(participants) == 0 {
		return nil, fmt.Errorf("leaderboard update payload has no participants")
	}

	output, err := h.service.ProcessRoundCommand(ctx, leaderboardservice.ProcessRoundCommand{
		GuildID:      string(payload.GuildID),
		RoundID:      uuid.UUID(payload.RoundID),
		Participants: participants,
	})
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, fmt.Errorf("process round returned nil output")
	}
	       updatedData := leaderboardDataFromFinalTags(output.FinalParticipantTags)
	        leaderboardData := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID, len(updatedData))
	        for _, entry := range updatedData {
	                leaderboardData[entry.TagNumber] = entry.UserID
	        }
	
	        changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(output.TagChanges))
	        for _, change := range output.TagChanges {
	                changedTags[sharedtypes.DiscordID(change.NewMemberID)] = sharedtypes.TagNumber(change.TagNumber)
	        }
	
	        results := []handlerwrapper.Result{		{
			Topic: leaderboardevents.LeaderboardUpdatedV1,
			Payload: &leaderboardevents.LeaderboardUpdatedPayloadV1{
				GuildID:         payload.GuildID,
				RoundID:         payload.RoundID,
				LeaderboardData: leaderboardData,
			},
		},
		{
			Topic: sharedevents.SyncRoundsTagRequestV1,
			Payload: &sharedevents.SyncRoundsTagRequestPayloadV1{
				GuildID:     payload.GuildID,
				Source:      "leaderboard_update",
				UpdatedAt:   time.Now().UTC(),
				ChangedTags: changedTags,
			},
		},
	}

	if !output.PointsSkipped && len(output.PointAwards) > 0 {
		pointsAwarded := make(map[sharedtypes.DiscordID]int, len(output.PointAwards))
		for _, award := range output.PointAwards {
			pointsAwarded[sharedtypes.DiscordID(award.MemberID)] = award.Points
		}

		pointsPayload := &sharedevents.PointsAwardedPayloadV1{
			GuildID: payload.GuildID,
			RoundID: payload.RoundID,
			Points:  pointsAwarded,
		}
		if h.roundLookup != nil {
			round, err := h.roundLookup.GetRound(ctx, payload.GuildID, payload.RoundID)
			if err != nil {
				h.logger.WarnContext(ctx, "failed to fetch round for points enrichment", "error", err)
			} else if round != nil {
				pointsPayload.EventMessageID = round.EventMessageID
				pointsPayload.Title = round.Title
				pointsPayload.Location = round.Location
				pointsPayload.StartTime = round.StartTime

				if round.Participants != nil {
					pointsPayload.Participants = make([]roundtypes.Participant, len(round.Participants))
					copy(pointsPayload.Participants, round.Participants)
					for i := range pointsPayload.Participants {
						if pts, ok := pointsPayload.Points[pointsPayload.Participants[i].UserID]; ok {
							p := pts
							pointsPayload.Participants[i].Points = &p
						}
					}
				}
				if round.Teams != nil {
					pointsPayload.Teams = make([]roundtypes.NormalizedTeam, len(round.Teams))
					for i, t := range round.Teams {
						pointsPayload.Teams[i] = t
						if t.Members != nil {
							pointsPayload.Teams[i].Members = make([]roundtypes.TeamMember, len(t.Members))
							copy(pointsPayload.Teams[i].Members, t.Members)
						}
						if t.HoleScores != nil {
							pointsPayload.Teams[i].HoleScores = make([]int, len(t.HoleScores))
							copy(pointsPayload.Teams[i].HoleScores, t.HoleScores)
						}
					}
				}
			}
		}

		pointsResult := handlerwrapper.Result{
			Topic:   sharedevents.PointsAwardedV1,
			Payload: pointsPayload,
		}
		if pointsPayload.EventMessageID != "" {
			pointsResult.Metadata = map[string]string{
				"discord_message_id": pointsPayload.EventMessageID,
			}
		}
		results = append(results, pointsResult)
	}

	results = h.addParallelIdentityResults(ctx, results, leaderboardevents.LeaderboardUpdatedV1, payload.GuildID)
	propagateCorrelationID(ctx, results)

	return results, nil
}

func buildParticipantsFromUpdatePayload(payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1) []leaderboardservice.RoundParticipantInput {
	if payload == nil {
		return nil
	}

	if len(payload.Participants) > 0 {
		participants := make([]leaderboardservice.RoundParticipantInput, 0, len(payload.Participants))
		for i, participant := range payload.Participants {
			finishRank := participant.FinishRank
			if finishRank <= 0 {
				// Fallback to index-based rank if explicit rank is missing,
				// assuming the participants list is sorted by finish order.
				finishRank = i + 1
			}
			participants = append(participants, leaderboardservice.RoundParticipantInput{
				MemberID:   string(participant.MemberID),
				FinishRank: finishRank,
			})
		}
		return participants
	}

	participants := make([]leaderboardservice.RoundParticipantInput, 0, len(payload.SortedParticipantTags))
	for i, tagUserPair := range payload.SortedParticipantTags {
		parts := strings.Split(tagUserPair, ":")
		if len(parts) != 2 {
			continue
		}
		// Note: Implicitly deriving FinishRank from the list index (i + 1).
		// We assume 'SortedParticipantTags' is ordered by rank (best to worst).
		participants = append(participants, leaderboardservice.RoundParticipantInput{
			MemberID:   parts[1],
			FinishRank: i + 1,
		})
	}
	return participants
}
