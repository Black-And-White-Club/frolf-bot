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
	"github.com/uptrace/bun"
)

func TestLeaderboardService_TagSwapRequested(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	requestorID := sharedtypes.DiscordID("user1")
	targetID := sharedtypes.DiscordID("user2")

	tests := []struct {
		name      string
		setupFake func(*FakeLeaderboardRepo)
		userID    sharedtypes.DiscordID
		targetTag sharedtypes.TagNumber
		expectErr bool
		verify    func(t *testing.T, data leaderboardtypes.LeaderboardData, err error)
	}{
		{
			name: "Successful tag swap",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: requestorID, TagNumber: 1},
							{UserID: targetID, TagNumber: 2},
						},
					}, nil
				}
				f.UpdateLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, data leaderboardtypes.LeaderboardData, uID sharedtypes.RoundID, s sharedtypes.ServiceUpdateSource) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{LeaderboardData: data}, nil
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: false,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				// Verify swap logic: User1 should now have 2, User2 should have 1
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
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: requestorID, TagNumber: 1},
						},
					}, nil
				}
			},
			userID:    requestorID,
			targetTag: 1,
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "cannot swap tag with self") {
					t.Errorf("expected self swap error, got: %v", err)
				}
			},
		},
		{
			name: "Requesting user not on leaderboard",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: targetID, TagNumber: 2},
						},
					}, nil
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "requesting user not on leaderboard") {
					t.Errorf("expected missing user error, got: %v", err)
				}
			},
		},
		{
			name: "Target tag not currently assigned",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: requestorID, TagNumber: 1},
						},
					}, nil
				}
			},
			userID:    requestorID,
			targetTag: 99,
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "target tag not currently assigned") {
					t.Errorf("expected unassigned tag error, got: %v", err)
				}
			},
		},
		{
			name: "GetActiveLeaderboard failure bubbles up",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return nil, errors.New("database connection failed")
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "database connection failed") {
					t.Errorf("expected db failure, got: %v", err)
				}
			},
		},
		{
			name: "UpdateLeaderboard failure bubbles up",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{
						LeaderboardData: leaderboardtypes.LeaderboardData{
							{UserID: requestorID, TagNumber: 1},
							{UserID: targetID, TagNumber: 2},
						},
					}, nil
				}
				f.UpdateLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, data leaderboardtypes.LeaderboardData, uID sharedtypes.RoundID, s sharedtypes.ServiceUpdateSource) (*leaderboarddb.Leaderboard, error) {
					return nil, errors.New("persistence error")
				}
			},
			userID:    requestorID,
			targetTag: 2,
			expectErr: true,
			verify: func(t *testing.T, data leaderboardtypes.LeaderboardData, err error) {
				if !strings.Contains(err.Error(), "persistence error") {
					t.Errorf("expected update failure, got: %v", err)
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

			// Initialize service. s.db is nil to trigger the tagSwapLogic direct path
			s := &LeaderboardService{
				repo:   fakeRepo,
				logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
			}

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
