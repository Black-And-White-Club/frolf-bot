package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	tag1, tag2 := sharedtypes.TagNumber(1), sharedtypes.TagNumber(2)
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		setupFake func(*FakeLeaderboardRepo)
		wantErr   error
		wantLen   int
	}{
		{
			name: "Successfully retrieves leaderboard",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: []leaderboardtypes.LeaderboardEntry{
							{TagNumber: tag1, UserID: "user1"},
							{TagNumber: tag2, UserID: "user2"},
						},
						GuildID: g,
					}, nil
				}
			},
			wantLen: 2,
		},
		{
			name: "Handles no active leaderboard",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return nil, leaderboarddb.ErrNoActiveLeaderboard
				}
			},
			wantErr: leaderboarddb.ErrNoActiveLeaderboard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			tt.setupFake(fakeRepo)
			s := &LeaderboardService{
				repo:    fakeRepo,
				logger:  loggerfrolfbot.NoOpLogger,
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}

			got, err := s.GetLeaderboard(context.Background(), guildID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("expected %d entries, got %d", tt.wantLen, len(got))
			}
		})
	}
}

func TestLeaderboardService_GetTagByUserID(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	userID := sharedtypes.DiscordID("user1")

	tests := []struct {
		name        string
		setupFake   func(*FakeLeaderboardRepo)
		expectedTag sharedtypes.TagNumber
		wantErr     error
	}{
		{
			name: "Successfully retrieves tag number",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: []leaderboardtypes.LeaderboardEntry{{UserID: userID, TagNumber: 5}},
					}, nil
				}
			},
			expectedTag: 5,
		},
		{
			name: "User ID not found in leaderboard",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{LeaderboardData: []leaderboardtypes.LeaderboardEntry{}}, nil
				}
			},
			wantErr: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			tt.setupFake(fakeRepo)
			s := &LeaderboardService{repo: fakeRepo, logger: loggerfrolfbot.NoOpLogger}

			tag, err := s.GetTagByUserID(context.Background(), guildID, userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tag != tt.expectedTag {
				t.Errorf("expected tag %v, got %v", tt.expectedTag, tag)
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
		setupFake       func(*FakeLeaderboardRepo)
		expectAvailable bool
		expectReason    string
		expectErr       bool
	}{
		{
			name: "tag available",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: []leaderboardtypes.LeaderboardEntry{{UserID: "other", TagNumber: 1}},
					}, nil
				}
			},
			expectAvailable: true,
		},
		{
			name: "tag unavailable (already taken)",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: []leaderboardtypes.LeaderboardEntry{{UserID: "someone-else", TagNumber: 42}},
					}, nil
				}
			},
			expectAvailable: false,
			expectReason:    "tag is already taken",
		},
		{
			name: "no active leaderboard",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return nil, leaderboarddb.ErrNoActiveLeaderboard
				}
			},
			expectAvailable: false,
			expectReason:    "no active leaderboard",
		},
		{
			name: "database error bubbles up",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return nil, errors.New("connection failed")
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			tt.setupFake(fakeRepo)

			s := &LeaderboardService{
				repo:   fakeRepo,
				logger: loggerfrolfbot.NoOpLogger,
			}

			result, err := s.CheckTagAvailability(ctx, guildID, userID, tagNumber)

			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Available != tt.expectAvailable {
				t.Errorf("expected available=%v, got %v", tt.expectAvailable, result.Available)
			}
			if tt.expectReason != "" && result.Reason != tt.expectReason {
				t.Errorf("expected reason %q, got %q", tt.expectReason, result.Reason)
			}
		})
	}
}
