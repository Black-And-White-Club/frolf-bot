package leaderboardservice

import (
	"context"
	"fmt"
	"sort"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// ProcessRound orchestrates the scoring of a round, including tag swaps and point calculations.
func (s *LeaderboardService) ProcessRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	playerResults []PlayerResult,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult[ProcessRoundResult, error], error) {

	processTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[ProcessRoundResult, error], error) {
		// 0. Idempotency Check (Rollback)
		// Undo any previous processing for this specific round to prevent double-counting.
		if err := s.rollbackRoundPoints(ctx, db, roundID); err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("rollback failed: %w", err)
		}

		// 1. Prepare Tag Assignment Requests for Batch Logic
		requests := make([]sharedtypes.TagAssignmentRequest, len(playerResults))
		for i, res := range playerResults {
			requests[i] = sharedtypes.TagAssignmentRequest{
				UserID:    res.PlayerID,
				TagNumber: sharedtypes.TagNumber(res.TagNumber),
			}
		}

		// 2. Execute Batch Tag Assignment (Handles conflicts & tag swaps)
		lbResult, err := s.executeBatchLogic(ctx, db, guildID, requests, roundID, source)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, err
		}
		if lbResult.IsFailure() {
			// Propagate domain failure (e.g. tag conflict)
			return results.FailureResult[ProcessRoundResult, error](*lbResult.Failure), nil
		}

		// 3. Calculate Points & Tiers
		// We need the "Season Best Tag" for each player to determine their Tier context.
		// We fetch this from the repository.
		playerIDs := make([]sharedtypes.DiscordID, len(playerResults))
		for i, res := range playerResults {
			playerIDs[i] = res.PlayerID
		}

		seasonBestTags, err := s.repo.GetSeasonBestTags(ctx, db, playerIDs)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to fetch season best tags: %w", err)
		}

		totalSeasonMembers, err := s.repo.CountSeasonMembers(ctx, db)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to count season members: %w", err)
		}

		// Prepare context for domain logic
		playerContexts := make(map[sharedtypes.DiscordID]leaderboarddomain.PlayerContext)

		// Batch fetch season standings to avoid N+1
		standingsMap, err := s.repo.GetSeasonStandings(ctx, db, playerIDs)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to fetch season standings: %w", err)
		}

		for _, pid := range playerIDs {
			standing := standingsMap[pid]

			roundsPlayed := 0
			if standing != nil {
				roundsPlayed = standing.RoundsPlayed
			}

			// Determine best tag including the new potential tag
			bestTag := seasonBestTags[pid]
			newTag := 0
			// Find new tag from lbResult
			if tag, ok := s.FindTagByUserID(*lbResult.Success, pid); ok {
				newTag = int(tag)
			}

			if bestTag == 0 || (newTag > 0 && newTag < bestTag) {
				bestTag = newTag
			}

			// Determine Tier based on this best tag
			tier := leaderboarddomain.DetermineTier(bestTag, totalSeasonMembers)

			playerContexts[pid] = leaderboarddomain.PlayerContext{
				ID:           string(pid),
				RoundsPlayed: roundsPlayed, // Before this round
				BestTag:      bestTag,      // Improved best tag
				CurrentTier:  tier,
			}
		}

		// Sort players by results (Tag Number) for matchup items
		type RankedPlayer struct {
			PlayerID  sharedtypes.DiscordID
			TagNumber int
		}
		ranked := make([]RankedPlayer, len(playerResults))
		for i, res := range playerResults {
			ranked[i] = RankedPlayer{res.PlayerID, res.TagNumber}
		}
		sort.Slice(ranked, func(i, j int) bool {
			return ranked[i].TagNumber < ranked[j].TagNumber
		})

		// Calculate Matchups pairwise
		pointsAwarded := make(map[sharedtypes.DiscordID]int)

		for i := 0; i < len(ranked); i++ {
			winner := playerContexts[ranked[i].PlayerID]
			pointsEarned := 0

			for j := i + 1; j < len(ranked); j++ {
				loser := playerContexts[ranked[j].PlayerID]
				wPoints := leaderboarddomain.CalculateMatchup(winner, loser)
				pointsEarned += int(wPoints)
			}

			pointsAwarded[ranked[i].PlayerID] = pointsEarned

			// Persist PointHistory
			if pointsEarned > 0 {
				history := &leaderboarddb.PointHistory{
					MemberID: ranked[i].PlayerID,
					RoundID:  roundID,
					Points:   pointsEarned,
					Reason:   "Round Matchups",
				}
				if err := s.repo.SavePointHistory(ctx, db, history); err != nil {
					return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to save point history: %w", err)
				}
			}

			// Update SeasonStanding
			standing := standingsMap[ranked[i].PlayerID]
			if standing == nil {
				standing = &leaderboarddb.SeasonStanding{
					MemberID: ranked[i].PlayerID,
				}
				// Add to map for subsequent lookups if needed, though we don't re-read it here.
				standingsMap[ranked[i].PlayerID] = standing
			}

			standing.TotalPoints += pointsEarned
			standing.RoundsPlayed++
			standing.SeasonBestTag = winner.BestTag
			standing.CurrentTier = string(winner.CurrentTier)
			standing.UpdatedAt = time.Now()

			if err := s.repo.UpsertSeasonStanding(ctx, db, standing); err != nil {
				return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to update season standing: %w", err)
			}
		}

		return results.SuccessResult[ProcessRoundResult, error](ProcessRoundResult{
			LeaderboardData: *lbResult.Success,
			PointsAwarded:   pointsAwarded,
		}), nil
	}

	return withTelemetry(s, ctx, "ProcessRound", guildID, func(ctx context.Context) (results.OperationResult[ProcessRoundResult, error], error) {
		return runInTx(s, ctx, processTx)
	})
}
