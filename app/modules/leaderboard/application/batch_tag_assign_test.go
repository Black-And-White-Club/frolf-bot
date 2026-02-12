package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLeaderboardService_ExecuteBatchTagAssignment(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name          string
		setupPipeline func(*FakeCommandPipeline)
		requests      []sharedtypes.TagAssignmentRequest
		expectErr     bool
		verify        func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error)
	}{
		{
			name: "successful batch updates leaderboard",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ApplyTagsFunc = func(ctx context.Context, guildID string, requests []sharedtypes.TagAssignmentRequest, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID) (leaderboardtypes.LeaderboardData, error) {
					return leaderboardtypes.LeaderboardData{
						{UserID: "user1", TagNumber: 1},
						{UserID: "existing_user", TagNumber: 10},
					}, nil
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				data := *res.Success
				if len(data) != 2 {
					t.Fatalf("expected 2 entries in leaderboard, got %d", len(data))
				}
				if data[0].TagNumber != 1 || data[1].TagNumber != 10 {
					t.Errorf("leaderboard data not sorted correctly: %+v", data)
				}
			},
		},
		{
			name: "swap needed returns TagSwapNeededError",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ApplyTagsFunc = func(ctx context.Context, guildID string, requests []sharedtypes.TagAssignmentRequest, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID) (leaderboardtypes.LeaderboardData, error) {
					return nil, &TagSwapNeededError{
						RequestorID:  "user1",
						TargetUserID: "target_user",
						TargetTag:    1,
						CurrentTag:   5,
					}
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !res.IsFailure() {
					t.Fatal("expected failure result, got success")
				}
				var tsn *TagSwapNeededError
				if !errors.As(*res.Failure, &tsn) {
					t.Fatalf("expected TagSwapNeededError, got: %T %v", *res.Failure, *res.Failure)
				}
				if tsn.TargetUserID != "target_user" {
					t.Errorf("expected conflict with target_user, got %s", tsn.TargetUserID)
				}
			},
		},
		{
			name: "pipeline error bubbles up",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.ApplyTagsFunc = func(ctx context.Context, guildID string, requests []sharedtypes.TagAssignmentRequest, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID) (leaderboardtypes.LeaderboardData, error) {
					return nil, errors.New("db failure")
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !strings.Contains(err.Error(), "db failure") {
					t.Fatalf("expected db failure error, got: %v", err)
				}
			},
		},
		{
			name:      "missing command pipeline returns unavailable",
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !errors.Is(err, ErrCommandPipelineUnavailable) {
					t.Fatalf("expected ErrCommandPipelineUnavailable, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()

			s := &LeaderboardService{
				repo:    fakeRepo,
				logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			if tt.setupPipeline != nil {
				pipeline := &FakeCommandPipeline{}
				tt.setupPipeline(pipeline)
				s.SetCommandPipeline(pipeline)
			}

			updateID := sharedtypes.RoundID(uuid.New())
			source := sharedtypes.ServiceUpdateSourceAdminBatch
			res, err := s.ExecuteBatchTagAssignment(context.Background(), guildID, tt.requests, updateID, source)

			if tt.expectErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err)
			}
		})
	}
}
