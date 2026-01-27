package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func TestLeaderboardService_ExecuteBatchTagAssignment(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		setupFake func(*FakeLeaderboardRepo)
		requests  []sharedtypes.TagAssignmentRequest
		expectErr bool
		verify    func(t *testing.T, data leaderboardtypes.LeaderboardData, err error)
	}{
		{
			name: "successful batch updates leaderboard",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "existing_user", TagNumber: 10}},
					}, nil
				}
				f.UpdateLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, data leaderboardtypes.LeaderboardData, uID sharedtypes.RoundID, s sharedtypes.ServiceUpdateSource) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{LeaderboardData: data}, nil
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: false,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if len(data) != 2 {
					t.Fatalf("expected 2 entries in leaderboard, got %d", len(data))
				}
				// Verify sorting (GenerateUpdatedSnapshot logic)
				if data[0].TagNumber != 1 || data[1].TagNumber != 10 {
					t.Errorf("leaderboard data not sorted correctly: %+v", data)
				}
			},
		},
		{
			name: "swap needed returns TagSwapNeededError",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					// user1 wants tag 1 but target_user currently holds tag 1
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: "user1", TagNumber: 5},
							{UserID: "target_user", TagNumber: 1},
						},
					}, nil
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				var tsn *TagSwapNeededError
				if !errors.As(err, &tsn) {
					t.Fatalf("expected TagSwapNeededError, got: %T %v", err, err)
				}
				if tsn.TargetUserID != "target_user" {
					t.Errorf("expected conflict with target_user, got %s", tsn.TargetUserID)
				}
			},
		},
		{
			name: "UpdateLeaderboard infrastructure error bubbles up",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{LeaderboardData: leaderboardtypes.LeaderboardData{}}, nil
				}
				f.UpdateLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, data leaderboardtypes.LeaderboardData, uID sharedtypes.RoundID, s sharedtypes.ServiceUpdateSource) (*leaderboarddb.Leaderboard, error) {
					return nil, errors.New("db failure")
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "db failure") {
					t.Fatalf("expected db failure error, got: %v", err)
				}
			},
		},
		{
			name: "GetActiveLeaderboard failure bubbles up",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return nil, errors.New("no access")
				}
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "no access") {
					t.Fatalf("expected no access error, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			// Initialize service with fake repo
			// Note: s.db is left nil so ExecuteBatchTagAssignment skips the real Bun transaction wrapper
			s := &LeaderboardService{
				repo:   fakeRepo,
				logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
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
