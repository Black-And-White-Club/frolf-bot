package leaderboardservice

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
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
		rollbackSeasonID, err := s.rollbackRoundPoints(ctx, db, roundID)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("rollback failed: %w", err)
		}

		processingSeasonID, err := s.resolveRoundProcessingSeasonID(ctx, db, rollbackSeasonID)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, err
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

		// 3. Fetch Data Required for Calculation (Fix N+1)
		// Use the round's own season if this is a recalculation; otherwise use active season.

		playerIDs := make([]sharedtypes.DiscordID, len(playerResults))
		for i, res := range playerResults {
			playerIDs[i] = res.PlayerID
		}

		seasonBestTags, err := s.repo.GetSeasonBestTags(ctx, db, processingSeasonID, playerIDs)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to fetch season best tags: %w", err)
		}

		totalSeasonMembers, err := s.repo.CountSeasonMembers(ctx, db, processingSeasonID)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to count season members: %w", err)
		}

		standingsMap, err := s.repo.GetSeasonStandings(ctx, db, processingSeasonID, playerIDs)
		if err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, fmt.Errorf("failed to fetch season standings: %w", err)
		}

		// 4. Build Contexts (Domain Logic Assembly)
		playerContexts := s.buildPlayerContexts(playerIDs, standingsMap, seasonBestTags, lbResult.Success, totalSeasonMembers)

		// 5. Calculate Outcomes (Pure Domain Logic)
		calculation := s.calculateRoundOutcomes(playerResults, playerContexts, roundID, processingSeasonID, standingsMap)

		// 6. Persist Outcomes (Bulk Writes)
		if err := s.persistRoundOutcomes(ctx, db, calculation); err != nil {
			return results.OperationResult[ProcessRoundResult, error]{}, err
		}

		return results.SuccessResult[ProcessRoundResult, error](ProcessRoundResult{
			LeaderboardData: *lbResult.Success,
			PointsAwarded:   calculation.PointsAwarded,
		}), nil
	}

	// Use helper for telemetry transaction wrapper to reduce closure nesting
	return withTelemetry(s, ctx, "ProcessRound", guildID, func(ctx context.Context) (results.OperationResult[ProcessRoundResult, error], error) {
		return runInTx(s, ctx, processTx)
	})
}

// --- Helpers ---

func (s *LeaderboardService) resolveRoundProcessingSeasonID(
	ctx context.Context,
	db bun.IDB,
	rollbackSeasonID string,
) (string, error) {
	if rollbackSeasonID != "" {
		return rollbackSeasonID, nil
	}

	activeSeason, err := s.repo.GetActiveSeason(ctx, db)
	if err != nil {
		return "", fmt.Errorf("failed to fetch active season: %w", err)
	}

	if activeSeason == nil || activeSeason.ID == "" {
		return "default", nil
	}

	return activeSeason.ID, nil
}

func (s *LeaderboardService) buildPlayerContexts(
	playerIDs []sharedtypes.DiscordID,
	standingsMap map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding,
	seasonBestTags map[sharedtypes.DiscordID]int,
	lbData *leaderboardtypes.LeaderboardData,
	totalSeasonMembers int,
) map[sharedtypes.DiscordID]leaderboarddomain.PlayerContext {

	contexts := make(map[sharedtypes.DiscordID]leaderboarddomain.PlayerContext, len(playerIDs))

	// Pre-process new tags to map for O(1) lookup
	newTags := make(map[sharedtypes.DiscordID]int, len(*lbData))
	for _, entry := range *lbData {
		newTags[entry.UserID] = int(entry.TagNumber)
	}

	for _, pid := range playerIDs {
		roundsPlayed := 0
		if standing, ok := standingsMap[pid]; ok && standing != nil {
			roundsPlayed = standing.RoundsPlayed
		}

		bestTag := seasonBestTags[pid]
		if newTag, ok := newTags[pid]; ok {
			if bestTag == 0 || (newTag > 0 && newTag < bestTag) {
				bestTag = newTag
			}
		}

		contexts[pid] = leaderboarddomain.PlayerContext{
			ID:           string(pid),
			RoundsPlayed: roundsPlayed,
			BestTag:      bestTag,
			CurrentTier:  leaderboarddomain.DetermineTier(bestTag, totalSeasonMembers),
		}
	}
	return contexts
}

type roundCalculationResult struct {
	PointsAwarded    map[sharedtypes.DiscordID]int
	Histories        []*leaderboarddb.PointHistory
	UpdatedStandings []*leaderboarddb.SeasonStanding
}

func (s *LeaderboardService) calculateRoundOutcomes(
	playerResults []PlayerResult,
	contexts map[sharedtypes.DiscordID]leaderboarddomain.PlayerContext,
	roundID sharedtypes.RoundID,
	seasonID string,
	standingsMap map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding,
) *roundCalculationResult {

	// Sort players using slices.SortFunc (Go 1.21+)
	type rankedPlayer struct {
		PlayerID  sharedtypes.DiscordID
		TagNumber int
	}
	ranked := make([]rankedPlayer, len(playerResults))
	for i, res := range playerResults {
		ranked[i] = rankedPlayer{res.PlayerID, res.TagNumber}
	}

	slices.SortFunc(ranked, func(a, b rankedPlayer) int {
		return cmp.Compare(a.TagNumber, b.TagNumber)
	})

	pointsAwarded := make(map[sharedtypes.DiscordID]int, len(ranked))
	histories := make([]*leaderboarddb.PointHistory, 0, len(ranked))
	updatedStandings := make([]*leaderboarddb.SeasonStanding, 0, len(ranked))

	// Cache time.Now() once
	now := time.Now().UTC()

	for i := 0; i < len(ranked); i++ {
		winnerID := ranked[i].PlayerID
		winnerCtx := contexts[winnerID]
		pointsEarned := 0
		opponentsBeaten := 0

		for j := i + 1; j < len(ranked); j++ {
			loserID := ranked[j].PlayerID
			loserCtx := contexts[loserID]
			wPoints := leaderboarddomain.CalculateMatchup(winnerCtx, loserCtx)
			pointsEarned += int(wPoints)
			opponentsBeaten++
		}

		pointsAwarded[winnerID] = pointsEarned

		// Point History
		histories = append(histories, &leaderboarddb.PointHistory{
			MemberID:  winnerID,
			RoundID:   roundID,
			Points:    pointsEarned,
			Reason:    "Round Matchups",
			Tier:      string(winnerCtx.CurrentTier),
			Opponents: opponentsBeaten,
			SeasonID:  seasonID,
		})

		// Standing Update
		standing, ok := standingsMap[winnerID]
		if !ok || standing == nil {
			standing = &leaderboarddb.SeasonStanding{
				MemberID: winnerID,
				SeasonID: seasonID,
			}
		}
		// Ensure SeasonID is set
		if standing.SeasonID == "" {
			standing.SeasonID = seasonID
		}

		standing.TotalPoints += pointsEarned
		standing.RoundsPlayed++
		standing.SeasonBestTag = winnerCtx.BestTag
		standing.CurrentTier = string(winnerCtx.CurrentTier)
		standing.UpdatedAt = now

		updatedStandings = append(updatedStandings, standing)
	}

	return &roundCalculationResult{
		PointsAwarded:    pointsAwarded,
		Histories:        histories,
		UpdatedStandings: updatedStandings,
	}
}

func (s *LeaderboardService) persistRoundOutcomes(
	ctx context.Context,
	db bun.IDB,
	calc *roundCalculationResult,
) error {
	// Use new Bulk methods
	if err := s.repo.BulkSavePointHistory(ctx, db, calc.Histories); err != nil {
		return fmt.Errorf("failed to bulk save point history: %w", err)
	}

	if err := s.repo.BulkUpsertSeasonStandings(ctx, db, calc.UpdatedStandings); err != nil {
		return fmt.Errorf("failed to bulk upsert season standings: %w", err)
	}
	return nil
}
