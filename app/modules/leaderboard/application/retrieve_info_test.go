package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name          string
		setupPipeline func(*FakeCommandPipeline)
		wantErr       bool
		wantLen       int
	}{
		{
			name: "Successfully retrieves leaderboard",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetTaggedFunc = func(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{
						{MemberID: "user1", Tag: 1},
						{MemberID: "user2", Tag: 2},
					}, nil
				}
			},
			wantLen: 2,
		},
		{
			name: "Pipeline failure bubbles up",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetTaggedFunc = func(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
					return nil, errors.New("lookup failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			pipeline := &FakeCommandPipeline{}
			if tt.setupPipeline != nil {
				tt.setupPipeline(pipeline)
			}

			s := &LeaderboardService{
				repo:    fakeRepo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			s.SetCommandPipeline(pipeline)

			res, err := s.GetLeaderboard(context.Background(), guildID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.IsSuccess() {
				t.Fatalf("expected success, got failure: %v", res.Failure)
			}
			if len(*res.Success) != tt.wantLen {
				t.Errorf("expected %d entries, got %d", tt.wantLen, len(*res.Success))
			}
		})
	}
}

func TestLeaderboardService_GetTagByUserID(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	userID := sharedtypes.DiscordID("user1")

	tests := []struct {
		name          string
		setupPipeline func(*FakeCommandPipeline)
		expectedTag   sharedtypes.TagNumber
		expectFail    bool
		wantErr       error
	}{
		{
			name: "Successfully retrieves tag number",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 5, true, nil
				}
			},
			expectedTag: 5,
		},
		{
			name: "User ID not found in leaderboard",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.GetMemberTagFunc = func(ctx context.Context, guildID, memberID string) (int, bool, error) {
					return 0, false, nil
				}
			},
			expectFail: true,
			wantErr:    sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				repo:    NewFakeLeaderboardRepo(),
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			pipeline := &FakeCommandPipeline{}
			tt.setupPipeline(pipeline)
			s.SetCommandPipeline(pipeline)

			res, err := s.GetTagByUserID(context.Background(), guildID, userID)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectFail {
				if !res.IsFailure() {
					t.Fatalf("expected failure, got success")
				}
				if !errors.Is(*res.Failure, tt.wantErr) {
					t.Errorf("expected failure error %v, got %v", tt.wantErr, *res.Failure)
				}
				return
			}

			if !res.IsSuccess() {
				t.Fatalf("expected success, got failure: %v", res.Failure)
			}
			if *res.Success != tt.expectedTag {
				t.Errorf("expected tag %v, got %v", tt.expectedTag, *res.Success)
			}
		})
	}
}

func TestLeaderboardService_CheckTagAvailability(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("test-guild")
	userID := sharedtypes.DiscordID("user-123")
	tagNumber := sharedtypes.TagNumber(42)

	tests := []struct {
		name            string
		setupPipeline   func(*FakeCommandPipeline)
		expectAvailable bool
		expectReason    string
		expectErr       bool
	}{
		{
			name: "tag available",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.CheckTagFunc = func(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error) {
					return true, "", nil
				}
			},
			expectAvailable: true,
		},
		{
			name: "tag unavailable (already taken)",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.CheckTagFunc = func(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error) {
					return false, "tag is already taken", nil
				}
			},
			expectAvailable: false,
			expectReason:    "tag is already taken",
		},
		{
			name: "command pipeline error bubbles up",
			setupPipeline: func(p *FakeCommandPipeline) {
				p.CheckTagFunc = func(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error) {
					return false, "", errors.New("connection failed")
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				repo:    NewFakeLeaderboardRepo(),
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}
			pipeline := &FakeCommandPipeline{}
			tt.setupPipeline(pipeline)
			s.SetCommandPipeline(pipeline)

			res, err := s.CheckTagAvailability(ctx, guildID, userID, tagNumber)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.IsSuccess() {
				t.Fatalf("expected success, got failure: %v", res.Failure)
			}

			result := *res.Success
			if result.Available != tt.expectAvailable {
				t.Errorf("expected available=%v, got %v", tt.expectAvailable, result.Available)
			}
			if tt.expectReason != "" && result.Reason != tt.expectReason {
				t.Errorf("expected reason %q, got %q", tt.expectReason, result.Reason)
			}
		})
	}
}
