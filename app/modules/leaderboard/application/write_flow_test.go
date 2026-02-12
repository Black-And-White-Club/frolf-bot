package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

type fakeLeagueMemberRepo struct {
	acquireGuildLockCalls int
	acquireGuildLockErr   error
	clearAllTagsErr       error

	getMembersByGuildFunc func(ctx context.Context, db bun.IDB, guildID string) ([]leaderboarddb.LeagueMember, error)
	getMembersByIDsFunc   func(ctx context.Context, db bun.IDB, guildID string, memberIDs []string) ([]leaderboarddb.LeagueMember, error)
	getMemberByIDFunc     func(ctx context.Context, db bun.IDB, guildID, memberID string) (*leaderboarddb.LeagueMember, error)
	getMemberByTagFunc    func(ctx context.Context, db bun.IDB, guildID string, tag int) (*leaderboarddb.LeagueMember, error)
	getMembersByTagsFunc  func(ctx context.Context, db bun.IDB, guildID string, tags []int) ([]leaderboarddb.LeagueMember, error)
	upsertMemberErr       error
	bulkUpsertMembersErr  error
	bulkUpsertCalls       [][]leaderboarddb.LeagueMember
}

func (f *fakeLeagueMemberRepo) GetMembersByGuild(ctx context.Context, db bun.IDB, guildID string) ([]leaderboarddb.LeagueMember, error) {
	if f.getMembersByGuildFunc != nil {
		return f.getMembersByGuildFunc(ctx, db, guildID)
	}
	return nil, nil
}

func (f *fakeLeagueMemberRepo) GetMembersByIDs(ctx context.Context, db bun.IDB, guildID string, memberIDs []string) ([]leaderboarddb.LeagueMember, error) {
	if f.getMembersByIDsFunc != nil {
		return f.getMembersByIDsFunc(ctx, db, guildID, memberIDs)
	}
	return nil, nil
}

func (f *fakeLeagueMemberRepo) GetMemberByID(ctx context.Context, db bun.IDB, guildID, memberID string) (*leaderboarddb.LeagueMember, error) {
	if f.getMemberByIDFunc != nil {
		return f.getMemberByIDFunc(ctx, db, guildID, memberID)
	}
	return nil, nil
}

func (f *fakeLeagueMemberRepo) GetMemberByTag(ctx context.Context, db bun.IDB, guildID string, tag int) (*leaderboarddb.LeagueMember, error) {
	if f.getMemberByTagFunc != nil {
		return f.getMemberByTagFunc(ctx, db, guildID, tag)
	}
	return nil, nil
}

func (f *fakeLeagueMemberRepo) GetMembersByTags(ctx context.Context, db bun.IDB, guildID string, tags []int) ([]leaderboarddb.LeagueMember, error) {
	if f.getMembersByTagsFunc != nil {
		return f.getMembersByTagsFunc(ctx, db, guildID, tags)
	}
	return nil, nil
}

func (f *fakeLeagueMemberRepo) UpsertMember(ctx context.Context, db bun.IDB, member *leaderboarddb.LeagueMember) error {
	return f.upsertMemberErr
}

func (f *fakeLeagueMemberRepo) BulkUpsertMembers(ctx context.Context, db bun.IDB, members []leaderboarddb.LeagueMember) error {
	cp := make([]leaderboarddb.LeagueMember, len(members))
	copy(cp, members)
	f.bulkUpsertCalls = append(f.bulkUpsertCalls, cp)
	return f.bulkUpsertMembersErr
}

func (f *fakeLeagueMemberRepo) ClearAllTags(ctx context.Context, db bun.IDB, guildID string) error {
	return f.clearAllTagsErr
}

func (f *fakeLeagueMemberRepo) AcquireGuildLock(ctx context.Context, db bun.IDB, guildID string) error {
	f.acquireGuildLockCalls++
	return f.acquireGuildLockErr
}

type fakeTagHistoryRepo struct {
	bulkInsertErr    error
	bulkInsertCalls  int
	lastBulkInserted []leaderboarddb.TagHistoryEntry
}

func (f *fakeTagHistoryRepo) BulkInsertTagHistory(ctx context.Context, db bun.IDB, entries []leaderboarddb.TagHistoryEntry) error {
	f.bulkInsertCalls++
	f.lastBulkInserted = make([]leaderboarddb.TagHistoryEntry, len(entries))
	copy(f.lastBulkInserted, entries)
	return f.bulkInsertErr
}

func (f *fakeTagHistoryRepo) GetTagHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) ([]leaderboarddb.TagHistoryEntry, error) {
	return nil, nil
}

func (f *fakeTagHistoryRepo) GetTagHistoryForMember(ctx context.Context, db bun.IDB, guildID, memberID string, limit int) ([]leaderboarddb.TagHistoryEntry, error) {
	return nil, nil
}

func (f *fakeTagHistoryRepo) GetLatestTagHistory(ctx context.Context, db bun.IDB, guildID string, limit int) ([]leaderboarddb.TagHistoryEntry, error) {
	return nil, nil
}

type fakeRoundOutcomeRepo struct {
	getOutcomeFunc    func(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) (*leaderboarddb.RoundOutcome, error)
	upsertOutcomeFunc func(ctx context.Context, db bun.IDB, outcome *leaderboarddb.RoundOutcome) error
}

func (f *fakeRoundOutcomeRepo) GetRoundOutcome(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) (*leaderboarddb.RoundOutcome, error) {
	if f.getOutcomeFunc != nil {
		return f.getOutcomeFunc(ctx, db, guildID, roundID)
	}
	return nil, nil
}

func (f *fakeRoundOutcomeRepo) UpsertRoundOutcome(ctx context.Context, db bun.IDB, outcome *leaderboarddb.RoundOutcome) error {
	if f.upsertOutcomeFunc != nil {
		return f.upsertOutcomeFunc(ctx, db, outcome)
	}
	return nil
}

func newWriteFlowTestService(repo *FakeLeaderboardRepo, members *fakeLeagueMemberRepo, tags *fakeTagHistoryRepo, outcomes *fakeRoundOutcomeRepo) *LeaderboardService {
	return &LeaderboardService{
		repo:        repo,
		memberRepo:  members,
		tagHistRepo: tags,
		outcomeRepo: outcomes,
		logger:      loggerfrolfbot.NoOpLogger,
		metrics:     &leaderboardmetrics.NoOpMetrics{},
		tracer:      noop.NewTracerProvider().Tracer("test"),
	}
}

func TestAdjustPoints_AcquiresGuildLock(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{}
	svc := newWriteFlowTestService(repo, members, &fakeTagHistoryRepo{}, &fakeRoundOutcomeRepo{})

	res, err := svc.AdjustPoints(context.Background(), sharedtypes.GuildID("guild-1"), sharedtypes.DiscordID("member-1"), 10, "admin fix")
	if err != nil {
		t.Fatalf("AdjustPoints returned error: %v", err)
	}
	if !res.IsSuccess() {
		t.Fatalf("expected success result")
	}
	if members.acquireGuildLockCalls != 1 {
		t.Fatalf("expected AcquireGuildLock to be called once, got %d", members.acquireGuildLockCalls)
	}
}

func TestProcessRoundInTx_IdempotentPath(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{}
	outcomes := &fakeRoundOutcomeRepo{}
	svc := newWriteFlowTestService(repo, members, &fakeTagHistoryRepo{}, outcomes)
	svc.logger = slog.Default()

	cmd := ProcessRoundCommand{
		GuildID: "guild-1",
		RoundID: uuid.New(),
		Participants: []RoundParticipantInput{
			{MemberID: "u1", FinishRank: 1},
			{MemberID: "u2", FinishRank: 2},
		},
	}
	hashInputs := []leaderboarddomain.RoundInput{
		{MemberID: "u1", FinishRank: 1},
		{MemberID: "u2", FinishRank: 2},
	}
	hash := leaderboarddomain.ComputeProcessingHash(hashInputs)
	seasonID := "season-1"

	outcomes.getOutcomeFunc = func(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) (*leaderboarddb.RoundOutcome, error) {
		return &leaderboarddb.RoundOutcome{
			GuildID:        guildID,
			RoundID:        roundID,
			SeasonID:       &seasonID,
			ProcessingHash: hash,
			ProcessedAt:    time.Now().UTC(),
		}, nil
	}
	members.getMembersByIDsFunc = func(ctx context.Context, db bun.IDB, guildID string, memberIDs []string) ([]leaderboarddb.LeagueMember, error) {
		tag1 := 1
		tag2 := 2
		return []leaderboarddb.LeagueMember{
			{GuildID: guildID, MemberID: "u1", CurrentTag: &tag1},
			{GuildID: guildID, MemberID: "u2", CurrentTag: &tag2},
		}, nil
	}

	out, err := svc.processRoundInTx(context.Background(), bun.Tx{}, cmd)
	if err != nil {
		t.Fatalf("processRoundInTx returned error: %v", err)
	}
	if !out.WasIdempotent {
		t.Fatalf("expected idempotent output")
	}
	if members.acquireGuildLockCalls != 1 {
		t.Fatalf("expected AcquireGuildLock to be called once, got %d", members.acquireGuildLockCalls)
	}
	if out.FinalParticipantTags["u1"] != 1 || out.FinalParticipantTags["u2"] != 2 {
		t.Fatalf("unexpected final tags: %+v", out.FinalParticipantTags)
	}
}

func TestProcessRoundInTx_RecalculationWindowExceeded(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{}
	outcomes := &fakeRoundOutcomeRepo{}
	svc := newWriteFlowTestService(repo, members, &fakeTagHistoryRepo{}, outcomes)

	cmd := ProcessRoundCommand{
		GuildID: "guild-1",
		RoundID: uuid.New(),
		Participants: []RoundParticipantInput{
			{MemberID: "u1", FinishRank: 1},
		},
	}
	outcomes.getOutcomeFunc = func(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) (*leaderboarddb.RoundOutcome, error) {
		return &leaderboarddb.RoundOutcome{
			GuildID:        guildID,
			RoundID:        roundID,
			ProcessingHash: "different-hash",
			ProcessedAt:    time.Now().UTC().Add(-RecalculationWindow - time.Minute),
		}, nil
	}

	_, err := svc.processRoundInTx(context.Background(), bun.Tx{}, cmd)
	if err == nil {
		t.Fatalf("expected recalculation window error")
	}
}

func TestPersistTagChanges_WritesMembersAndHistory(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{}
	tags := &fakeTagHistoryRepo{}
	svc := newWriteFlowTestService(repo, members, tags, &fakeRoundOutcomeRepo{})

	roundID := uuid.New()
	changes := []leaderboarddomain.TagChange{
		{TagNumber: 1, OldMemberID: "u2", NewMemberID: "u1"},
		{TagNumber: 2, OldMemberID: "u1", NewMemberID: "u2"},
	}

	err := svc.persistTagChanges(context.Background(), bun.Tx{}, "guild-1", &roundID, changes, "round_swap")
	if err != nil {
		t.Fatalf("persistTagChanges returned error: %v", err)
	}
	if len(members.bulkUpsertCalls) != 2 {
		t.Fatalf("expected 2 bulk upsert calls, got %d", len(members.bulkUpsertCalls))
	}
	if tags.bulkInsertCalls != 1 || len(tags.lastBulkInserted) != 2 {
		t.Fatalf("expected single history bulk insert with 2 entries")
	}
	if tags.lastBulkInserted[0].Reason != "round_swap" || tags.lastBulkInserted[1].Reason != "round_swap" {
		t.Fatalf("expected tag history reason round_swap, got %+v", tags.lastBulkInserted)
	}
}

func TestCalculateAndPersistPoints_PersistsAwardsAndStandings(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{}
	svc := newWriteFlowTestService(repo, members, &fakeTagHistoryRepo{}, &fakeRoundOutcomeRepo{})

	cmd := ProcessRoundCommand{
		GuildID: "guild-1",
		RoundID: uuid.New(),
		Participants: []RoundParticipantInput{
			{MemberID: "u1", FinishRank: 1},
			{MemberID: "u2", FinishRank: 2},
		},
	}

	var savedHistories []*leaderboarddb.PointHistory
	var savedStandings []*leaderboarddb.SeasonStanding

	repo.GetSeasonBestTagsFunc = func(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error) {
		return map[sharedtypes.DiscordID]int{
			"u1": 1,
			"u2": 2,
		}, nil
	}
	repo.CountSeasonMembersFunc = func(ctx context.Context, db bun.IDB, guildID string, seasonID string) (int, error) {
		return 10, nil
	}
	repo.GetSeasonStandingsFunc = func(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding, error) {
		return map[sharedtypes.DiscordID]*leaderboarddb.SeasonStanding{
			"u1": {MemberID: "u1", SeasonID: seasonID, TotalPoints: 50, RoundsPlayed: 3},
		}, nil
	}
	repo.BulkSavePointHistoryFunc = func(ctx context.Context, db bun.IDB, guildID string, histories []*leaderboarddb.PointHistory) error {
		savedHistories = histories
		return nil
	}
	repo.BulkUpsertSeasonStandingsFunc = func(ctx context.Context, db bun.IDB, guildID string, standings []*leaderboarddb.SeasonStanding) error {
		savedStandings = standings
		return nil
	}

	awards, err := svc.calculateAndPersistPoints(context.Background(), bun.Tx{}, cmd, map[string]int{"u1": 1, "u2": 2}, nil, "season-1")
	if err != nil {
		t.Fatalf("calculateAndPersistPoints returned error: %v", err)
	}
	if len(awards) != 2 {
		t.Fatalf("expected 2 awards, got %d", len(awards))
	}
	if len(savedHistories) != 2 || len(savedStandings) != 2 {
		t.Fatalf("expected 2 histories and 2 standings, got %d and %d", len(savedHistories), len(savedStandings))
	}
}

func TestRollbackPreviousRound_DeletesHistoryAfterDecrement(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	svc := newWriteFlowTestService(repo, &fakeLeagueMemberRepo{}, &fakeTagHistoryRepo{}, &fakeRoundOutcomeRepo{})

	roundID := uuid.New()
	repo.GetPointHistoryForRoundFunc = func(ctx context.Context, db bun.IDB, guildID string, rid sharedtypes.RoundID) ([]leaderboarddb.PointHistory, error) {
		return []leaderboarddb.PointHistory{
			{MemberID: "u1", SeasonID: "season-1", Points: 25},
			{MemberID: "u2", SeasonID: "season-1", Points: 10},
		}, nil
	}

	decrementCalls := 0
	deleteCalled := false
	repo.DecrementSeasonStandingFunc = func(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID, seasonID string, pointsToRemove int) error {
		decrementCalls++
		return nil
	}
	repo.DeletePointHistoryForRoundFunc = func(ctx context.Context, db bun.IDB, guildID string, rid sharedtypes.RoundID) error {
		deleteCalled = true
		return nil
	}

	err := svc.rollbackPreviousRound(context.Background(), bun.Tx{}, "guild-1", roundID)
	if err != nil {
		t.Fatalf("rollbackPreviousRound returned error: %v", err)
	}
	if decrementCalls != 2 {
		t.Fatalf("expected 2 decrement calls, got %d", decrementCalls)
	}
	if !deleteCalled {
		t.Fatalf("expected point history delete to be called")
	}
}

func TestApplyTagAssignmentsInTx_SuccessfulSwap(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{}
	tags := &fakeTagHistoryRepo{}
	svc := newWriteFlowTestService(repo, members, tags, &fakeRoundOutcomeRepo{})

	members.getMembersByIDsFunc = func(ctx context.Context, db bun.IDB, guildID string, memberIDs []string) ([]leaderboarddb.LeagueMember, error) {
		tag1 := 1
		tag2 := 2
		return []leaderboarddb.LeagueMember{
			{GuildID: guildID, MemberID: "u1", CurrentTag: &tag1},
			{GuildID: guildID, MemberID: "u2", CurrentTag: &tag2},
		}, nil
	}
	members.getMembersByTagsFunc = func(ctx context.Context, db bun.IDB, guildID string, requestedTags []int) ([]leaderboarddb.LeagueMember, error) {
		tag1 := 1
		tag2 := 2
		return []leaderboarddb.LeagueMember{
			{GuildID: guildID, MemberID: "u1", CurrentTag: &tag1},
			{GuildID: guildID, MemberID: "u2", CurrentTag: &tag2},
		}, nil
	}
	members.getMembersByGuildFunc = func(ctx context.Context, db bun.IDB, guildID string) ([]leaderboarddb.LeagueMember, error) {
		tag1 := 1
		tag2 := 2
		return []leaderboarddb.LeagueMember{
			{GuildID: guildID, MemberID: "u1", CurrentTag: &tag2},
			{GuildID: guildID, MemberID: "u2", CurrentTag: &tag1},
		}, nil
	}

	data, err := svc.applyTagAssignmentsInTx(
		context.Background(),
		bun.Tx{},
		"guild-1",
		[]sharedtypes.TagAssignmentRequest{
			{UserID: "u1", TagNumber: 2},
			{UserID: "u2", TagNumber: 1},
		},
		sharedtypes.ServiceUpdateSourceTagSwap,
		sharedtypes.RoundID(uuid.New()),
	)
	if err != nil {
		t.Fatalf("applyTagAssignmentsInTx returned error: %v", err)
	}
	if len(data) != 2 || len(tags.lastBulkInserted) != 2 {
		t.Fatalf("expected 2 leaderboard entries and 2 tag history entries, got %d and %d", len(data), len(tags.lastBulkInserted))
	}
}

func TestApplyTagAssignmentsInTx_DuplicateTagRejected(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	svc := newWriteFlowTestService(repo, &fakeLeagueMemberRepo{}, &fakeTagHistoryRepo{}, &fakeRoundOutcomeRepo{})

	_, err := svc.applyTagAssignmentsInTx(
		context.Background(),
		bun.Tx{},
		"guild-1",
		[]sharedtypes.TagAssignmentRequest{
			{UserID: "u1", TagNumber: 5},
			{UserID: "u2", TagNumber: 5},
		},
		sharedtypes.ServiceUpdateSourceTagSwap,
		sharedtypes.RoundID(uuid.New()),
	)
	if err == nil {
		t.Fatalf("expected duplicate tag validation error")
	}
}

func TestAdjustPoints_LockFailureBubblesUp(t *testing.T) {
	repo := NewFakeLeaderboardRepo()
	members := &fakeLeagueMemberRepo{acquireGuildLockErr: errors.New("lock failed")}
	svc := newWriteFlowTestService(repo, members, &fakeTagHistoryRepo{}, &fakeRoundOutcomeRepo{})

	_, err := svc.AdjustPoints(context.Background(), sharedtypes.GuildID("guild-1"), sharedtypes.DiscordID("member-1"), 10, "admin fix")
	if err == nil {
		t.Fatalf("expected lock acquisition error")
	}
}
