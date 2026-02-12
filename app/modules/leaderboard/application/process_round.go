package leaderboardservice

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// ProcessRound delegates round processing to the normalized command pipeline.
func (s *LeaderboardService) ProcessRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	playerResults []PlayerResult,
	_ sharedtypes.ServiceUpdateSource,
) (results.OperationResult[ProcessRoundResult, error], error) {
	return withTelemetry(s, ctx, "ProcessRound", guildID, func(ctx context.Context) (results.OperationResult[ProcessRoundResult, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[ProcessRoundResult, error]{}, ErrCommandPipelineUnavailable
		}

		ranked := make([]PlayerResult, len(playerResults))
		copy(ranked, playerResults)
		slices.SortFunc(ranked, func(a, b PlayerResult) int {
			// Treat 0 (no tag) as max int to sort to bottom
			aTag := a.TagNumber
			if aTag <= 0 {
				aTag = math.MaxInt
			}
			bTag := b.TagNumber
			if bTag <= 0 {
				bTag = math.MaxInt
			}

			if c := cmp.Compare(aTag, bTag); c != 0 {
				return c
			}
			return cmp.Compare(a.PlayerID, b.PlayerID)
		})

		participants := make([]RoundParticipantInput, 0, len(ranked))
		for i, res := range ranked {
			participants = append(participants, RoundParticipantInput{
				MemberID:   string(res.PlayerID),
				FinishRank: i + 1,
			})
		}

		output, err := s.commandPipeline.ProcessRound(ctx, ProcessRoundCommand{
			GuildID:      string(guildID),
			RoundID:      roundID.UUID(),
			Participants: participants,
		})
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("process round command: %w", err)
		}
		if output == nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("process round command returned nil output")
		}

		entries := make(leaderboardtypes.LeaderboardData, 0, len(output.FinalParticipantTags))
		for memberID, tag := range output.FinalParticipantTags {
			if tag <= 0 {
				continue
			}
			entries = append(entries, leaderboardtypes.LeaderboardEntry{
				UserID:    sharedtypes.DiscordID(memberID),
				TagNumber: sharedtypes.TagNumber(tag),
			})
		}
		slices.SortFunc(entries, func(a, b leaderboardtypes.LeaderboardEntry) int {
			if c := cmp.Compare(a.TagNumber, b.TagNumber); c != 0 {
				return c
			}
			return cmp.Compare(a.UserID, b.UserID)
		})

		points := make(map[sharedtypes.DiscordID]int, len(output.PointAwards))
		for _, award := range output.PointAwards {
			points[sharedtypes.DiscordID(award.MemberID)] = award.Points
		}

		return results.SuccessResult[ProcessRoundResult, error](ProcessRoundResult{
			LeaderboardData: entries,
			PointsAwarded:   points,
		}), nil
	})
}
