package scoreservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestScoreService_ProcessRoundScores(t *testing.T) {
	ctx := context.Background()
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testScore := sharedtypes.Score(10)
	testTag := sharedtypes.TagNumber(1)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &scoremetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	// Updated the verify signature to use the new ProcessRoundScoresResult type
	tests := []struct {
		name           string
		scores         []sharedtypes.ScoreInfo
		overwrite      bool
		setupFake      func(*FakeScoreRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository)
	}{
		{
			name:      "success - processes scores and returns tag mappings",
			overwrite: true,
			scores: []sharedtypes.ScoreInfo{
				{UserID: testUserID, Score: testScore, TagNumber: &testTag},
			},
			setupFake: func(f *FakeScoreRepository) {
				f.GetScoresForRoundFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{}, nil
				}
				f.LogScoresFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID, s []sharedtypes.ScoreInfo, src string) error {
					return nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Success == nil {
					t.Fatal("expected success result, got nil")
				}

				// Verify the TagMappings are actually populated in the result
				if len(res.Success.TagMappings) != 1 {
					t.Errorf("expected 1 tag mapping, got %d", len(res.Success.TagMappings))
				} else if res.Success.TagMappings[0].DiscordID != testUserID {
					t.Errorf("expected tag mapping for %s, got %s", testUserID, res.Success.TagMappings[0].DiscordID)
				}

				found := false
				for _, call := range fake.Trace() {
					if call == "LogScores" {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected LogScores to be called on success")
				}
			},
		},
		{
			name:      "domain failure - scores exist and no overwrite",
			scores:    []sharedtypes.ScoreInfo{{UserID: testUserID, Score: testScore}},
			overwrite: false,
			setupFake: func(f *FakeScoreRepository) {
				f.GetScoresForRoundFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return []sharedtypes.ScoreInfo{{UserID: "existing-user"}}, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if res.Failure == nil {
					t.Fatal("expected failure result for existing scores")
				}
				if !errors.Is(*res.Failure, ErrScoresAlreadyExist) {
					t.Errorf("expected ErrScoresAlreadyExist, got %v", *res.Failure)
				}
			},
		},
		{
			name:      "infra failure - database error on GetScores",
			scores:    []sharedtypes.ScoreInfo{{UserID: testUserID, Score: testScore}},
			overwrite: true,
			setupFake: func(f *FakeScoreRepository) {
				f.GetScoresForRoundFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return nil, errors.New("connection reset")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "connection reset") {
					t.Errorf("expected infra error 'connection reset', got %v", infraErr)
				}
			},
		},
		{
			name: "domain failure - validation error (invalid score)",
			scores: []sharedtypes.ScoreInfo{
				{UserID: testUserID, Score: 99}, // Invalid
			},
			overwrite: true,
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if res.Failure == nil {
					t.Fatal("expected failure for invalid score")
				}
				if !errors.Is(*res.Failure, ErrInvalidScore) {
					t.Errorf("expected ErrInvalidScore, got %v", *res.Failure)
				}
			},
		},
		{
			name:      "domain failure - empty scores list returns ErrInvalidScore",
			overwrite: true,
			scores:    []sharedtypes.ScoreInfo{},
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil {
					t.Fatal("expected failure for empty score list")
				}
				if !errors.Is(*res.Failure, ErrInvalidScore) {
					t.Errorf("expected ErrInvalidScore, got %v", *res.Failure)
				}
			},
		},
		// When every participant DNFs, the domain should return ErrAllScoresDNF rather
		// than ErrInvalidScore "cannot process empty score list" which looks like a data
		// bug rather than a valid business outcome.
		{
			name:      "domain failure - all scores are DNF returns ErrAllScoresDNF",
			overwrite: true,
			scores: []sharedtypes.ScoreInfo{
				{UserID: "p1", Score: 0, IsDNF: true},
				{UserID: "p2", Score: 0, IsDNF: true},
			},
			setupFake: func(f *FakeScoreRepository) {
				f.GetScoresForRoundFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
					return nil, nil
				}
			},
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil {
					t.Fatal("expected domain failure for all-DNF round, got success")
				}
				if !errors.Is(*res.Failure, ErrAllScoresDNF) {
					t.Errorf("expected ErrAllScoresDNF, got %v", *res.Failure)
				}
			},
		},
		{
			name:      "success - verifies sorting and tie-breaking logic",
			overwrite: true,
			scores: []sharedtypes.ScoreInfo{
				// User-A and User-B are tied on score; User-A has the lower pre-round tag
				// so it wins the tiebreak (disc golf convention: lower tag = better finisher).
				{UserID: "User-C", Score: 20},
				{UserID: "User-B", Score: 10, TagNumber: ptr(sharedtypes.TagNumber(11))},
				{UserID: "User-A", Score: 10, TagNumber: ptr(sharedtypes.TagNumber(3))},
				{UserID: "User-D", Score: 5},
			},
			verify: func(t *testing.T, res results.OperationResult[ProcessRoundScoresResult, error], infraErr error, fake *FakeScoreRepository) {
				if res.Success == nil {
					t.Fatal("expected success")
				}

				results := fake.LastLoggedScores
				if len(results) != 4 {
					t.Fatalf("expected 4 scores, got %d", len(results))
				}

				// Logic Check: Sorting order (score asc, then pre-round tag asc for ties)
				if results[0].UserID != "User-D" || results[1].UserID != "User-A" {
					t.Errorf("sorting logic failed: expected User-D then User-A, got %s then %s", results[0].UserID, results[1].UserID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeScoreRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &ScoreService{
				repo:    fakeRepo,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			res, err := s.ProcessRoundScores(ctx, testGuildID, testRoundID, tt.scores, tt.overwrite)

			if tt.expectInfraErr && err == nil {
				t.Error("expected infrastructure error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Errorf("unexpected infrastructure error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, fakeRepo)
			}
		})
	}
}

// TestProcessRoundScores_FinishRanks verifies that ProcessRoundScores correctly
// populates FinishRanksByDiscordID with competition-style ranks.
func TestProcessRoundScores_FinishRanks(t *testing.T) {
	ctx := context.Background()
	testGuildID := sharedtypes.GuildID("guild-1234")
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name       string
		scores     []sharedtypes.ScoreInfo
		wantRanks  map[sharedtypes.DiscordID]int
	}{
		{
			name: "two players tied produce equal rank with competition skip",
			// alice and bob both scored -4; carol scored -2.
			// Expected ranks: alice=1, bob=1, carol=3 (rank 2 is skipped).
			scores: []sharedtypes.ScoreInfo{
				{UserID: "alice", Score: -4, TagNumber: ptr(sharedtypes.TagNumber(1))},
				{UserID: "bob", Score: -4, TagNumber: ptr(sharedtypes.TagNumber(5))},
				{UserID: "carol", Score: -2},
			},
			wantRanks: map[sharedtypes.DiscordID]int{
				"alice": 1,
				"bob":   1,
				"carol": 3,
			},
		},
		{
			name: "no ties produce sequential ranks",
			scores: []sharedtypes.ScoreInfo{
				{UserID: "p1", Score: -5, TagNumber: ptr(sharedtypes.TagNumber(1))},
				{UserID: "p2", Score: -3, TagNumber: ptr(sharedtypes.TagNumber(2))},
				{UserID: "p3", Score: 0},
			},
			wantRanks: map[sharedtypes.DiscordID]int{
				"p1": 1,
				"p2": 2,
				"p3": 3,
			},
		},
		{
			name: "untagged player tied with tagged player shares rank 1",
			// tagged (tag=5) and untagged both scored -3.
			// Same score → same finish rank. The tag tiebreak only affects sort order
			// for tag assignment, not the competition rank.
			scores: []sharedtypes.ScoreInfo{
				{UserID: "tagged", Score: -3, TagNumber: ptr(sharedtypes.TagNumber(5))},
				{UserID: "untagged", Score: -3},
			},
			wantRanks: map[sharedtypes.DiscordID]int{
				"tagged":   1,
				"untagged": 1,
			},
		},
		{
			name: "three-way tie gives everyone rank 1",
			scores: []sharedtypes.ScoreInfo{
				{UserID: "x", Score: -2, TagNumber: ptr(sharedtypes.TagNumber(2))},
				{UserID: "y", Score: -2, TagNumber: ptr(sharedtypes.TagNumber(7))},
				{UserID: "z", Score: -2, TagNumber: ptr(sharedtypes.TagNumber(11))},
			},
			wantRanks: map[sharedtypes.DiscordID]int{
				"x": 1,
				"y": 1,
				"z": 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeScoreRepository()
			s := &ScoreService{
				repo:    fakeRepo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &scoremetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			res, err := s.ProcessRoundScores(ctx, testGuildID, testRoundID, tt.scores, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Success == nil {
				t.Fatalf("expected success result")
			}

			for id, wantRank := range tt.wantRanks {
				gotRank := res.Success.FinishRanksByDiscordID[id]
				if gotRank != wantRank {
					t.Errorf("player %s: want rank %d, got %d", id, wantRank, gotRank)
				}
			}
		})
	}
}
