package leaderboardservice

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ProcessRoundCommand represents the input for processing a round.
type ProcessRoundCommand struct {
	GuildID      string
	RoundID      uuid.UUID
	Participants []RoundParticipantInput
}

// RoundParticipantInput represents a single participant's finish data.
type RoundParticipantInput struct {
	MemberID   string
	FinishRank int // 1-based finish position
}

// ProcessRoundOutput represents the result of round processing.
type ProcessRoundOutput struct {
	TagChanges           []leaderboarddomain.TagChange
	PointAwards          []leaderboarddomain.PointAward
	FinalParticipantTags map[string]int
	PointsSkipped        bool
	SeasonID             string
	WasIdempotent        bool
}

// TaggedMemberView is a normalized read model for current guild tag state.
type TaggedMemberView struct {
	MemberID string
	Tag      int
}

// ProcessRound executes the spec's full ProcessRound workflow:
//  1. Acquire guild advisory lock
//  2. Compute processing hash + idempotency check
//  3. Load current tag state from league_members
//  4. Stream 1: Tag allocation (closed pool)
//  5. Persist tag changes to league_members + tag_history
//  6. Stream 2: Points calculation (if active season)
//  7. Persist points to season_standings + point_history
//  8. Record round outcome
func (s *LeaderboardService) processRoundCommandCore(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
	var output *ProcessRoundOutput

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var txErr error
		output, txErr = s.processRoundInTx(ctx, tx, cmd)
		return txErr
	})
	if err != nil {
		return nil, fmt.Errorf("LeaderboardService.ProcessRound: %w", err)
	}
	return output, nil
}

func (s *LeaderboardService) processRoundInTx(ctx context.Context, tx bun.Tx, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
	// 1. Acquire per-guild transactional lock
	if err := s.memberRepo.AcquireGuildLock(ctx, tx, cmd.GuildID); err != nil {
		return nil, fmt.Errorf("acquire guild lock: %w", err)
	}

	// 2. Compute processing hash for idempotency
	hashInputs := make([]leaderboarddomain.RoundInput, len(cmd.Participants))
	for i, p := range cmd.Participants {
		hashInputs[i] = leaderboarddomain.RoundInput{
			MemberID:   p.MemberID,
			FinishRank: p.FinishRank,
		}
	}
	processingHash := leaderboarddomain.ComputeProcessingHash(hashInputs)

	// Check existing outcome
	existing, err := s.outcomeRepo.GetRoundOutcome(ctx, tx, cmd.GuildID, cmd.RoundID)
	if err != nil {
		return nil, fmt.Errorf("check existing outcome: %w", err)
	}

	if existing != nil && existing.ProcessingHash == processingHash {
		// Same data already processed - idempotent no-op
		s.logger.InfoContext(ctx, "Round already processed with same data (idempotent no-op)",
			slog.String("guild_id", cmd.GuildID),
			slog.String("round_id", cmd.RoundID.String()),
		)

		memberIDs := make([]string, len(cmd.Participants))
		for i, participant := range cmd.Participants {
			memberIDs[i] = participant.MemberID
		}
		members, err := s.memberRepo.GetMembersByIDs(ctx, tx, cmd.GuildID, memberIDs)
		if err != nil {
			return nil, fmt.Errorf("load idempotent participant tags: %w", err)
		}

		finalParticipantTags := make(map[string]int, len(members))
		for _, member := range members {
			if member.CurrentTag == nil || *member.CurrentTag <= 0 {
				continue
			}
			finalParticipantTags[member.MemberID] = *member.CurrentTag
		}

		seasonID := ""
		pointsSkipped := true
		if existing.SeasonID != nil && *existing.SeasonID != "" {
			seasonID = *existing.SeasonID
			pointsSkipped = false
		}

		return &ProcessRoundOutput{
			FinalParticipantTags: finalParticipantTags,
			SeasonID:             seasonID,
			PointsSkipped:        pointsSkipped,
			WasIdempotent:        true,
		}, nil
	}

	if existing != nil && existing.ProcessingHash != processingHash {
		// Different data for same round - this is a recalculation
		// Rollback previous points before reprocessing
		s.logger.InfoContext(ctx, "Round data changed, triggering recalculation",
			slog.String("guild_id", cmd.GuildID),
			slog.String("round_id", cmd.RoundID.String()),
		)

		// SAFETY: Prevent tag modifications for historical rounds to avoid corrupting current state.
		// Only allow recalculation (points only) or block entirely if too old.
		if time.Since(existing.ProcessedAt) > 24*time.Hour {
			return nil, fmt.Errorf("cannot recalculate round older than 24h due to tag state drift")
		}

		if err := s.rollbackPreviousRound(ctx, tx, cmd.GuildID, cmd.RoundID); err != nil {
			return nil, fmt.Errorf("rollback previous round: %w", err)
		}
	}

	// 3. Load current tag state for participants
	memberIDs := make([]string, len(cmd.Participants))
	for i, p := range cmd.Participants {
		memberIDs[i] = p.MemberID
	}

	members, err := s.memberRepo.GetMembersByIDs(ctx, tx, cmd.GuildID, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("load member tag state: %w", err)
	}

	memberTagMap := make(map[string]int, len(members))
	for _, m := range members {
		if m.CurrentTag != nil {
			memberTagMap[m.MemberID] = *m.CurrentTag
		}
	}

	// 4. Stream 1: Tag allocation (closed pool)
	tagInputs := make([]leaderboarddomain.TagAllocationInput, len(cmd.Participants))
	for i, p := range cmd.Participants {
		tagInputs[i] = leaderboarddomain.TagAllocationInput{
			MemberID:   p.MemberID,
			FinishRank: p.FinishRank,
			CurrentTag: memberTagMap[p.MemberID], // 0 if no tag
		}
	}

	tagChanges := leaderboarddomain.AllocateTagsClosedPool(tagInputs)

	// 5. Persist tag changes
	if len(tagChanges) > 0 {
		if err := s.persistTagChanges(ctx, tx, cmd.GuildID, &cmd.RoundID, tagChanges, "round_swap"); err != nil {
			return nil, fmt.Errorf("persist tag changes: %w", err)
		}
	}

	// Ensure all participants have a league_members row (even if no tag change)
	if err := s.ensureParticipantMembers(ctx, tx, cmd.GuildID, cmd.Participants, memberTagMap, tagChanges); err != nil {
		return nil, fmt.Errorf("ensure participant members: %w", err)
	}

	// 6. Resolve active season
	rollbackSeasonID := ""
	if existing != nil && existing.SeasonID != nil {
		rollbackSeasonID = *existing.SeasonID
	}
	season, err := s.resolveActiveSeason(ctx, tx, cmd.GuildID, rollbackSeasonID)
	if err != nil {
		return nil, fmt.Errorf("resolve season: %w", err)
	}

	output := &ProcessRoundOutput{
		TagChanges: tagChanges,
	}

	// Build final participant tag state using domain function.
	assignments := leaderboarddomain.ComputeFinalTagState(memberTagMap, tagChanges)
	finalParticipantTags := make(map[string]int, len(assignments))
	for _, a := range assignments {
		finalParticipantTags[a.MemberID] = a.Tag
	}
	output.FinalParticipantTags = finalParticipantTags

	// 7. Stream 2: Points calculation
	if !leaderboarddomain.ShouldAwardPoints(season) {
		output.PointsSkipped = true
		s.logger.InfoContext(ctx, "Points skipped (no active season)",
			slog.String("guild_id", cmd.GuildID),
		)
	} else {
		output.SeasonID = season.SeasonID

		awards, err := s.calculateAndPersistPoints(ctx, tx, cmd, memberTagMap, tagChanges, season.SeasonID)
		if err != nil {
			return nil, fmt.Errorf("calculate and persist points: %w", err)
		}
		output.PointAwards = awards
	}

	// 8. Record round outcome for idempotency
	seasonIDPtr := &output.SeasonID
	if output.PointsSkipped {
		seasonIDPtr = nil
	}
	if err := s.outcomeRepo.UpsertRoundOutcome(ctx, tx, &leaderboarddb.RoundOutcome{
		GuildID:        cmd.GuildID,
		RoundID:        cmd.RoundID,
		SeasonID:       seasonIDPtr,
		ProcessingHash: processingHash,
		ProcessedAt:    time.Now().UTC(),
	}); err != nil {
		return nil, fmt.Errorf("record round outcome: %w", err)
	}

	return output, nil
}

// persistTagChanges updates league_members and writes tag_history entries.
func (s *LeaderboardService) persistTagChanges(
	ctx context.Context,
	tx bun.Tx,
	guildID string,
	roundID *uuid.UUID,
	changes []leaderboarddomain.TagChange,
	reason string,
) error {
	// Build bulk upsert for league_members
	membersToUpdate := make([]leaderboarddb.LeagueMember, len(changes))
	for i, ch := range changes {
		tag := ch.TagNumber
		membersToUpdate[i] = leaderboarddb.LeagueMember{
			GuildID:    guildID,
			MemberID:   ch.NewMemberID,
			CurrentTag: &tag,
		}
	}

	if err := s.memberRepo.BulkUpsertMembers(ctx, tx, membersToUpdate); err != nil {
		return fmt.Errorf("bulk upsert members: %w", err)
	}

	// Write tag history entries
	histEntries := make([]leaderboarddb.TagHistoryEntry, len(changes))
	for i, ch := range changes {
		var oldMember *string
		if ch.OldMemberID != "" {
			oldMember = &ch.OldMemberID
		}
		histEntries[i] = leaderboarddb.TagHistoryEntry{
			GuildID:     guildID,
			RoundID:     roundID,
			TagNumber:   ch.TagNumber,
			OldMemberID: oldMember,
			NewMemberID: ch.NewMemberID,
			Reason:      reason,
			Metadata:    "{}",
		}
	}

	return s.tagHistRepo.BulkInsertTagHistory(ctx, tx, histEntries)
}

// ensureParticipantMembers ensures all round participants exist in league_members.
func (s *LeaderboardService) ensureParticipantMembers(
	ctx context.Context,
	tx bun.Tx,
	guildID string,
	participants []RoundParticipantInput,
	existingTags map[string]int,
	tagChanges []leaderboarddomain.TagChange,
) error {
	// Build map of new tags from changes
	changedTags := make(map[string]int, len(tagChanges))
	for _, ch := range tagChanges {
		changedTags[ch.NewMemberID] = ch.TagNumber
	}

	var toUpsert []leaderboarddb.LeagueMember
	for _, p := range participants {
		if _, changed := changedTags[p.MemberID]; changed {
			continue // already handled in persistTagChanges
		}
		// Ensure the member exists even if no tag change
		tag := existingTags[p.MemberID]
		var tagPtr *int
		if tag > 0 {
			tagPtr = &tag
		}
		toUpsert = append(toUpsert, leaderboarddb.LeagueMember{
			GuildID:    guildID,
			MemberID:   p.MemberID,
			CurrentTag: tagPtr,
		})
	}

	if len(toUpsert) > 0 {
		return s.memberRepo.BulkUpsertMembers(ctx, tx, toUpsert)
	}
	return nil
}

func (s *LeaderboardService) resolveActiveSeason(ctx context.Context, tx bun.Tx, guildID string, rollbackSeasonID string) (leaderboarddomain.SeasonInfo, error) {
	season, err := s.repo.GetActiveSeason(ctx, tx, guildID)
	if err != nil {
		return leaderboarddomain.SeasonInfo{}, err
	}
	var state *leaderboarddomain.SeasonState
	if season != nil {
		state = &leaderboarddomain.SeasonState{
			GuildID:  season.GuildID,
			SeasonID: season.ID,
			IsActive: season.IsActive,
		}
	}
	return leaderboarddomain.ResolveSeasonForRound(rollbackSeasonID, state), nil
}

func (s *LeaderboardService) calculateAndPersistPoints(
	ctx context.Context,
	tx bun.Tx,
	cmd ProcessRoundCommand,
	existingTags map[string]int,
	tagChanges []leaderboarddomain.TagChange,
	seasonID string,
) ([]leaderboarddomain.PointAward, error) {
	// Build final tag state using domain function.
	tagAssignments := leaderboarddomain.ComputeFinalTagState(existingTags, tagChanges)
	finalTags := make(map[string]int, len(tagAssignments))
	for _, a := range tagAssignments {
		finalTags[a.MemberID] = a.Tag
	}

	// Fetch season data for tier calculation
	memberIDs := make([]sharedtypes.DiscordID, len(cmd.Participants))
	for i, p := range cmd.Participants {
		memberIDs[i] = sharedtypes.DiscordID(p.MemberID)
	}

	seasonBestTags, err := s.repo.GetSeasonBestTags(ctx, tx, cmd.GuildID, seasonID, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch season best tags: %w", err)
	}

	totalMembers, err := s.repo.CountSeasonMembers(ctx, tx, cmd.GuildID, seasonID)
	if err != nil {
		return nil, fmt.Errorf("count season members: %w", err)
	}

	standingsMap, err := s.repo.GetSeasonStandings(ctx, tx, cmd.GuildID, seasonID, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch season standings: %w", err)
	}

	// Build RoundParticipant slice for domain calculation
	participants := make([]leaderboarddomain.RoundParticipant, len(cmd.Participants))
	for i, p := range cmd.Participants {
		discordID := sharedtypes.DiscordID(p.MemberID)
		tag := finalTags[p.MemberID]

		roundsPlayed := 0
		if standing, ok := standingsMap[discordID]; ok && standing != nil {
			roundsPlayed = standing.RoundsPlayed
		}

		bestTag := leaderboarddomain.UpdateBestTag(seasonBestTags[discordID], tag)

		participants[i] = leaderboarddomain.RoundParticipant{
			MemberID:     p.MemberID,
			TagNumber:    tag,
			RoundsPlayed: roundsPlayed,
			BestTag:      bestTag,
			CurrentTier:  leaderboarddomain.DetermineTier(bestTag, totalMembers),
		}
	}

	// Calculate points using domain logic
	awards := leaderboarddomain.CalculateRoundPoints(participants)

	// Persist point history
	histories := make([]*leaderboarddb.PointHistory, len(awards))
	for i, award := range awards {
		histories[i] = &leaderboarddb.PointHistory{
			SeasonID:  seasonID,
			MemberID:  sharedtypes.DiscordID(award.MemberID),
			RoundID:   sharedtypes.RoundID(cmd.RoundID),
			Points:    award.Points,
			Reason:    "Round Matchups",
			Tier:      string(award.Tier),
			Opponents: award.OpponentsBeaten,
		}
	}

	if err := s.repo.BulkSavePointHistory(ctx, tx, cmd.GuildID, histories); err != nil {
		return nil, fmt.Errorf("bulk save point history: %w", err)
	}

	// Update season standings
	now := time.Now().UTC()
	standings := make([]*leaderboarddb.SeasonStanding, len(awards))
	for i, award := range awards {
		discordID := sharedtypes.DiscordID(award.MemberID)
		existing := standingsMap[discordID]

		totalPoints := award.Points
		roundsPlayed := 1
		if existing != nil {
			totalPoints += existing.TotalPoints
			roundsPlayed += existing.RoundsPlayed
		}

		bestTag := leaderboarddomain.UpdateBestTag(seasonBestTags[discordID], finalTags[award.MemberID])

		standings[i] = &leaderboarddb.SeasonStanding{
			SeasonID:      seasonID,
			MemberID:      discordID,
			TotalPoints:   totalPoints,
			RoundsPlayed:  roundsPlayed,
			SeasonBestTag: bestTag,
			CurrentTier:   string(award.Tier),
			UpdatedAt:     now,
		}
	}

	if err := s.repo.BulkUpsertSeasonStandings(ctx, tx, cmd.GuildID, standings); err != nil {
		return nil, fmt.Errorf("bulk upsert season standings: %w", err)
	}

	return awards, nil
}

func (s *LeaderboardService) rollbackPreviousRound(ctx context.Context, tx bun.Tx, guildID string, roundID uuid.UUID) error {
	// Fetch previous point history for this round
	history, err := s.repo.GetPointHistoryForRound(ctx, tx, guildID, sharedtypes.RoundID(roundID))
	if err != nil {
		return fmt.Errorf("fetch previous point history: %w", err)
	}

	if len(history) == 0 {
		return nil
	}

	// Decrement standings for each player
	for _, ph := range history {
		if err := s.repo.DecrementSeasonStanding(ctx, tx, guildID, ph.MemberID, ph.SeasonID, ph.Points); err != nil {
			return fmt.Errorf("decrement standing for %s: %w", ph.MemberID, err)
		}
	}

	// Delete the point history entries
	return s.repo.DeletePointHistoryForRound(ctx, tx, guildID, sharedtypes.RoundID(roundID))
}

// ResetTags clears all tags for a guild and reassigns based on qualifying round finish order.
func (s *LeaderboardService) resetTagsCore(ctx context.Context, guildID string, finishOrder []string) ([]leaderboarddomain.TagChange, error) {
	var changes []leaderboarddomain.TagChange

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.memberRepo.AcquireGuildLock(ctx, tx, guildID); err != nil {
			return fmt.Errorf("acquire guild lock: %w", err)
		}

		// Clear all tags
		if err := s.memberRepo.ClearAllTags(ctx, tx, guildID); err != nil {
			return fmt.Errorf("clear all tags: %w", err)
		}

		// Allocate new tags
		changes = leaderboarddomain.AllocateTagsFromReset(finishOrder)

		// Persist
		if err := s.persistTagChanges(ctx, tx, guildID, nil, changes, "reset"); err != nil {
			return fmt.Errorf("persist reset tag changes: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("LeaderboardService.ResetTags: %w", err)
	}
	return changes, nil
}

// StartSeason creates a new season, deactivating any existing active season.
func (s *LeaderboardService) startSeasonCore(ctx context.Context, guildID, seasonID, seasonName string) error {
	if msg := leaderboarddomain.ValidateSeasonStart(seasonID, seasonName); msg != "" {
		return fmt.Errorf("validation: %s", msg)
	}

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.memberRepo.AcquireGuildLock(ctx, tx, guildID); err != nil {
			return err
		}

		if err := s.repo.DeactivateAllSeasons(ctx, tx, guildID); err != nil {
			return fmt.Errorf("deactivate seasons: %w", err)
		}

		now := time.Now().UTC()
		return s.repo.CreateSeason(ctx, tx, guildID, &leaderboarddb.Season{
			ID:        seasonID,
			Name:      seasonName,
			IsActive:  true,
			StartDate: now,
		})
	})

	if err != nil {
		return fmt.Errorf("LeaderboardService.StartSeason: %w", err)
	}
	return nil
}

// EndSeason deactivates the current active season for a guild.
func (s *LeaderboardService) endSeasonCore(ctx context.Context, guildID string) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.memberRepo.AcquireGuildLock(ctx, tx, guildID); err != nil {
			return err
		}
		return s.repo.DeactivateAllSeasons(ctx, tx, guildID)
	})

	if err != nil {
		return fmt.Errorf("LeaderboardService.EndSeason: %w", err)
	}
	return nil
}

// GetTaggedMembers returns the current normalized tag state for a guild, sorted by tag.
func (s *LeaderboardService) getTaggedMembersCore(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
	members, err := s.memberRepo.GetMembersByGuild(ctx, s.db, guildID)
	if err != nil {
		return nil, fmt.Errorf("LeaderboardService.GetTaggedMembers: %w", err)
	}

	views := make([]TaggedMemberView, 0, len(members))
	for _, member := range members {
		if member.CurrentTag == nil || *member.CurrentTag <= 0 {
			continue
		}
		views = append(views, TaggedMemberView{
			MemberID: member.MemberID,
			Tag:      *member.CurrentTag,
		})
	}

	slices.SortFunc(views, func(a, b TaggedMemberView) int {
		if c := cmp.Compare(a.Tag, b.Tag); c != 0 {
			return c
		}
		return cmp.Compare(a.MemberID, b.MemberID)
	})

	return views, nil
}

// GetMemberTag returns a member's current normalized tag for a guild.
func (s *LeaderboardService) getMemberTagCore(ctx context.Context, guildID, memberID string) (int, bool, error) {
	member, err := s.memberRepo.GetMemberByID(ctx, s.db, guildID, memberID)
	if err != nil {
		return 0, false, fmt.Errorf("LeaderboardService.GetMemberTag: %w", err)
	}
	if member == nil || member.CurrentTag == nil || *member.CurrentTag <= 0 {
		return 0, false, nil
	}
	return *member.CurrentTag, true, nil
}

// CheckTagAvailability validates whether a specific tag can be claimed by a member in a guild.
func (s *LeaderboardService) checkTagAvailabilityCore(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error) {
	if tagNumber <= 0 {
		return false, "tag_number_must_be_positive", nil
	}

	members, err := s.memberRepo.GetMembersByGuild(ctx, s.db, guildID)
	if err != nil {
		return false, "", fmt.Errorf("LeaderboardService.CheckTagAvailability: %w", err)
	}

	for _, member := range members {
		if member.CurrentTag == nil || *member.CurrentTag != tagNumber {
			continue
		}
		if member.MemberID != memberID {
			return false, "tag is already taken", nil
		}
	}

	return true, "", nil
}

// ApplyTagAssignments applies explicit user->tag assignments in one transaction and returns
// the resulting normalized leaderboard snapshot.
func (s *LeaderboardService) applyTagAssignmentsCore(
	ctx context.Context,
	guildID string,
	requests []sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	updateID sharedtypes.RoundID,
) (leaderboardtypes.LeaderboardData, error) {
	var data leaderboardtypes.LeaderboardData

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := s.memberRepo.AcquireGuildLock(ctx, tx, guildID); err != nil {
			return err
		}

		var applyErr error
		data, applyErr = s.applyTagAssignmentsInTx(ctx, tx, guildID, requests, source, updateID)
		return applyErr
	})
	if err != nil {
		return nil, fmt.Errorf("LeaderboardService.ApplyTagAssignments: %w", err)
	}
	return data, nil
}

type tagAssignment struct {
	memberID string
	tag      int
}

func (s *LeaderboardService) applyTagAssignmentsInTx(
	ctx context.Context,
	tx bun.Tx,
	guildID string,
	requests []sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	updateID sharedtypes.RoundID,
) (leaderboardtypes.LeaderboardData, error) {
	dedup := make(map[string]int, len(requests))
	for _, req := range requests {
		if req.TagNumber <= 0 {
			return nil, fmt.Errorf("invalid tag number %d for user %s", req.TagNumber, req.UserID)
		}
		dedup[string(req.UserID)] = int(req.TagNumber)
	}

	assignments := make([]tagAssignment, 0, len(dedup))
	for memberID, tag := range dedup {
		assignments = append(assignments, tagAssignment{memberID: memberID, tag: tag})
	}
	slices.SortFunc(assignments, func(a, b tagAssignment) int {
		if c := cmp.Compare(a.tag, b.tag); c != 0 {
			return c
		}
		return cmp.Compare(a.memberID, b.memberID)
	})

	if len(assignments) == 0 {
		return s.normalizedLeaderboardData(ctx, tx, guildID)
	}

	requestedUsers := make(map[string]struct{}, len(assignments))
	requestedTags := make(map[int]string, len(assignments))
	for _, assignment := range assignments {
		requestedUsers[assignment.memberID] = struct{}{}
		if otherUser, exists := requestedTags[assignment.tag]; exists && otherUser != assignment.memberID {
			return nil, fmt.Errorf("duplicate requested tag %d for users %s and %s", assignment.tag, otherUser, assignment.memberID)
		}
		requestedTags[assignment.tag] = assignment.memberID
	}

	members, err := s.memberRepo.GetMembersByGuild(ctx, tx, guildID)
	if err != nil {
		return nil, fmt.Errorf("load members: %w", err)
	}

	currentUserTag := make(map[string]int, len(members))
	currentTagUser := make(map[int]string, len(members))
	for _, member := range members {
		if member.CurrentTag == nil || *member.CurrentTag <= 0 {
			continue
		}
		currentUserTag[member.MemberID] = *member.CurrentTag
		currentTagUser[*member.CurrentTag] = member.MemberID
	}

	for _, assignment := range assignments {
		holderID, occupied := currentTagUser[assignment.tag]
		if !occupied || holderID == assignment.memberID {
			continue
		}
		if _, holderRequested := requestedUsers[holderID]; holderRequested {
			continue
		}

		currentTag := sharedtypes.TagNumber(0)
		if tag, ok := currentUserTag[assignment.memberID]; ok {
			currentTag = sharedtypes.TagNumber(tag)
		}

		return nil, &TagSwapNeededError{
			RequestorID:  sharedtypes.DiscordID(assignment.memberID),
			TargetUserID: sharedtypes.DiscordID(holderID),
			TargetTag:    sharedtypes.TagNumber(assignment.tag),
			CurrentTag:   currentTag,
		}
	}

	clearRows := make([]leaderboarddb.LeagueMember, 0, len(assignments))
	for _, assignment := range assignments {
		if _, exists := currentUserTag[assignment.memberID]; !exists {
			continue
		}
		clearRows = append(clearRows, leaderboarddb.LeagueMember{
			GuildID:    guildID,
			MemberID:   assignment.memberID,
			CurrentTag: nil,
		})
	}
	if err := s.memberRepo.BulkUpsertMembers(ctx, tx, clearRows); err != nil {
		return nil, fmt.Errorf("clear current tags for reassignment: %w", err)
	}

	upserts := make([]leaderboarddb.LeagueMember, 0, len(assignments))
	for _, assignment := range assignments {
		tag := assignment.tag
		upserts = append(upserts, leaderboarddb.LeagueMember{
			GuildID:    guildID,
			MemberID:   assignment.memberID,
			CurrentTag: &tag,
		})
	}
	if err := s.memberRepo.BulkUpsertMembers(ctx, tx, upserts); err != nil {
		return nil, fmt.Errorf("apply tag assignments: %w", err)
	}

	afterUserTag := maps.Clone(currentUserTag)
	for _, assignment := range assignments {
		afterUserTag[assignment.memberID] = assignment.tag
	}

	beforeTagUser := make(map[int]string, len(currentTagUser))
	for tag, memberID := range currentTagUser {
		beforeTagUser[tag] = memberID
	}
	afterTagUser := make(map[int]string, len(afterUserTag))
	for memberID, tag := range afterUserTag {
		afterTagUser[tag] = memberID
	}

	reason := historyReasonFromSource(source)
	var roundID *uuid.UUID
	if rid := uuid.UUID(updateID); rid != uuid.Nil {
		roundID = &rid
	}

	historyEntries := make([]leaderboarddb.TagHistoryEntry, 0, len(afterTagUser))
	for tag, newMemberID := range afterTagUser {
		oldMemberID, hadOld := beforeTagUser[tag]
		if hadOld && oldMemberID == newMemberID {
			continue
		}

		var oldPtr *string
		if hadOld {
			oldCopy := oldMemberID
			oldPtr = &oldCopy
		}

		historyEntries = append(historyEntries, leaderboarddb.TagHistoryEntry{
			GuildID:     guildID,
			RoundID:     roundID,
			TagNumber:   tag,
			OldMemberID: oldPtr,
			NewMemberID: newMemberID,
			Reason:      reason,
			Metadata:    "{}",
		})
	}
	slices.SortFunc(historyEntries, func(a, b leaderboarddb.TagHistoryEntry) int {
		if c := cmp.Compare(a.TagNumber, b.TagNumber); c != 0 {
			return c
		}
		return cmp.Compare(a.NewMemberID, b.NewMemberID)
	})
	if err := s.tagHistRepo.BulkInsertTagHistory(ctx, tx, historyEntries); err != nil {
		return nil, fmt.Errorf("write tag history: %w", err)
	}

	return s.normalizedLeaderboardData(ctx, tx, guildID)
}

func (s *LeaderboardService) normalizedLeaderboardData(ctx context.Context, db bun.IDB, guildID string) (leaderboardtypes.LeaderboardData, error) {
	members, err := s.memberRepo.GetMembersByGuild(ctx, db, guildID)
	if err != nil {
		return nil, fmt.Errorf("load normalized leaderboard data: %w", err)
	}

	data := make(leaderboardtypes.LeaderboardData, 0, len(members))
	for _, member := range members {
		if member.CurrentTag == nil || *member.CurrentTag <= 0 {
			continue
		}
		data = append(data, leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(member.MemberID),
			TagNumber: sharedtypes.TagNumber(*member.CurrentTag),
		})
	}
	slices.SortFunc(data, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if c := cmp.Compare(a.TagNumber, b.TagNumber); c != 0 {
			return c
		}
		return cmp.Compare(a.UserID, b.UserID)
	})
	return data, nil
}

func historyReasonFromSource(source sharedtypes.ServiceUpdateSource) string {
	switch source {
	case sharedtypes.ServiceUpdateSourceCreateUser:
		return "claim"
	case sharedtypes.ServiceUpdateSourceTagSwap, sharedtypes.ServiceUpdateSourceProcessScores:
		return "round_swap"
	default:
		return "admin_fix"
	}
}
