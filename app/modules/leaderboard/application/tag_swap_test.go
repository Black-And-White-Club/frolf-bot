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
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLeaderboardService_TagSwapRequested(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	requestorID := sharedtypes.DiscordID("user1")
	targetID := sharedtypes.DiscordID("user2")

	tests := []struct {
		name          string
		setupPipeline func(*FakeCommandPipeline)
		userID        sharedtypes.DiscordID
		targetTag     sharedtypes.TagNumber
		expectErr     bool
		verify        func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error)
	}{
		{
			name: "Successful tag swap",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					if memberID == string(requestorID) {
						return 1, true, nil
					}
					return 0, false, nil
				}
				p.GetTaggedFunc = func(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{
						{MemberID: string(requestorID), Tag: 1},
						{MemberID: string(targetID), Tag: 2},
					}, nil
				}
				p.ApplyTagsFunc = func(ctx context.Context, guildID string, requests []sharedtypes.TagAssignmentRequest, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID) (leaderboardtypes.LeaderboardData, error) {
					if len(requests) != 2 {
						t.Fatalf("expected 2 swap assignments, got %d", len(requests))
					}
					return leaderboardtypes.LeaderboardData{
						{UserID: requestorID, TagNumber: 2},
						{UserID: targetID, TagNumber: 1},
					}, nil
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !res.IsSuccess() {
					t.Fatalf("expected success, got failure: %v", res.Failure)
				}
				data := *res.Success
				for _, entry := range data {
					if entry.UserID == requestorID && entry.TagNumber != 2 {
						t.Errorf("expected requestor to have tag 2, got %d", entry.TagNumber)
					}
					if entry.UserID == targetID && entry.TagNumber != 1 {
						t.Errorf("expected target to have tag 1, got %d", entry.TagNumber)
					}
				}
			},
		},
		{
			name: "Cannot swap tag with self",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 1, true, nil
				}
				p.GetTaggedFunc = func(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{{MemberID: string(requestorID), Tag: 1}}, nil
				}
			},
			userID:    requestorID,
			targetTag: 1,
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !res.IsFailure() {
					t.Fatal("expected failure result, got success")
				}
				if !strings.Contains((*res.Failure).Error(), "cannot swap tag with self") {
					t.Errorf("expected self swap error, got: %v", *res.Failure)
				}
			},
		},
		{
			name: "Requesting user not on leaderboard",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 0, false, nil
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !res.IsFailure() {
					t.Fatal("expected failure result, got success")
				}
				if !strings.Contains((*res.Failure).Error(), "requesting user not on leaderboard") {
					t.Errorf("expected missing user error, got: %v", *res.Failure)
				}
			},
		},
		{
			name: "Target tag not currently assigned",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 1, true, nil
				}
				p.GetTaggedFunc = func(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{{MemberID: string(requestorID), Tag: 1}}, nil
				}
			},
			userID:    requestorID,
			targetTag: 99,
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !res.IsFailure() {
					t.Fatal("expected failure result, got success")
				}
				if !strings.Contains((*res.Failure).Error(), "target tag not currently assigned") {
					t.Errorf("expected unassigned tag error, got: %v", *res.Failure)
				}
			},
		},
		{
			name: "Get tag failure bubbles up",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 0, false, errors.New("database connection failed")
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !strings.Contains(err.Error(), "database connection failed") {
					t.Errorf("expected db failure, got: %v", err)
				}
			},
		},
		{
			name: "ApplyTagAssignments failure bubbles up",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 1, true, nil
				}
				p.GetTaggedFunc = func(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{
						{MemberID: string(requestorID), Tag: 1},
						{MemberID: string(targetID), Tag: 2},
					}, nil
				}
				p.ApplyTagsFunc = func(ctx context.Context, guildID string, requests []sharedtypes.TagAssignmentRequest, source sharedtypes.ServiceUpdateSource, updateID sharedtypes.RoundID) (leaderboardtypes.LeaderboardData, error) {
					return nil, errors.New("persistence error")
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if !strings.Contains(err.Error(), "persistence error") {
					t.Errorf("expected update failure, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := &FakeCommandPipeline{}
			if tt.setupPipeline != nil {
				tt.setupPipeline(pipeline)
			}

			s := &LeaderboardService{
				repo:    NewFakeLeaderboardRepo(),
				logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			s.SetCommandPipeline(pipeline)

			res, err := s.TagSwapRequested(context.Background(), guildID, tt.userID, tt.targetTag)

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
