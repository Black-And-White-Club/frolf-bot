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
			name:      "success - verifies sorting and tie-breaking logic",
			overwrite: true,
			scores: []sharedtypes.ScoreInfo{
				{UserID: "User-C", Score: 20},
				{UserID: "User-B", Score: 10},
				{UserID: "User-A", Score: 10},
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

				// Logic Check: Sorting order
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
