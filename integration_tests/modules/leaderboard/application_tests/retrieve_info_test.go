package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/uptrace/bun"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardService "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestLeaderboardReadOperations(t *testing.T) {
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()

	ctx := context.Background()

	tests := []struct {
		name      string
		guildID   sharedtypes.GuildID
		setupData func(t *testing.T, db *bun.DB)
		// Changed to use the interface type leaderboardservice.Service
		runOperation func(s leaderboardService.Service) (any, error)
		wantErr      bool
		validate     func(t *testing.T, result any, err error)
	}{
		{
			name:    "GetLeaderboard: Success returns active leaderboard data",
			guildID: "test_guild_1",
			setupData: func(t *testing.T, db *bun.DB) {
				lb := &leaderboarddb.Leaderboard{
					GuildID: "test_guild_1", IsActive: true,
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_1", TagNumber: 1},
					},
				}
				_, err := db.NewInsert().Model(lb).Exec(ctx)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			runOperation: func(s leaderboardService.Service) (any, error) {
				return s.GetLeaderboard(ctx, "test_guild_1")
			},
			validate: func(t *testing.T, result any, err error) {
				opResult := result.(results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error])
				if err != nil {
					t.Fatalf("expected no system error, got: %v", err)
				}
				if opResult.Success == nil {
					t.Fatalf("expected success payload, got nil")
				}
				entries := *opResult.Success
				if len(entries) != 1 {
					t.Errorf("expected 1 entry, got %d", len(entries))
				}
			},
		},
		{
			name:    "GetLeaderboard: Returns error in result for missing leaderboard",
			guildID: "empty_guild",
			runOperation: func(s leaderboardService.Service) (any, error) {
				return s.GetLeaderboard(ctx, "empty_guild")
			},
			wantErr: true,
			validate: func(t *testing.T, result any, err error) {
				// Missing leaderboard results in a system error from the underlying repo
				if err == nil {
					t.Fatalf("expected system error for missing leaderboard, got nil")
				}
				if !errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
					t.Errorf("expected ErrNoActiveLeaderboard, got %v", err)
				}
			},
		},
		{
			name:    "RoundGetTagByUserID: Success returns data slice",
			guildID: "test_guild_1",
			setupData: func(t *testing.T, db *bun.DB) {
				lb := &leaderboarddb.Leaderboard{
					GuildID: "test_guild_1", IsActive: true,
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "round_user", TagNumber: 42},
					},
				}
				_, _ = db.NewInsert().Model(lb).Exec(ctx)
			},
			runOperation: func(s leaderboardService.Service) (any, error) {
				return s.RoundGetTagByUserID(ctx, "test_guild_1", "round_user")
			},
			validate: func(t *testing.T, result any, err error) {
				opResult := result.(results.OperationResult[sharedtypes.TagNumber, error])
				if err != nil {
					t.Fatalf("expected no system error, got: %v", err)
				}
				if opResult.Success == nil {
					t.Fatalf("expected success payload, got nil")
				}
				tag := *opResult.Success
				if tag != 42 {
					t.Errorf("expected tag 42, got: %v", tag)
				}
			},
		},
		{
			name:    "GetTagByUserID: Success returns raw tag",
			guildID: "test_guild_1",
			setupData: func(t *testing.T, db *bun.DB) {
				lb := &leaderboarddb.Leaderboard{
					GuildID: "test_guild_1", IsActive: true,
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "raw_user", TagNumber: 10},
					},
				}
				_, _ = db.NewInsert().Model(lb).Exec(ctx)
			},
			runOperation: func(s leaderboardService.Service) (any, error) {
				return s.GetTagByUserID(ctx, "test_guild_1", "raw_user")
			},
			validate: func(t *testing.T, result any, err error) {
				opResult := result.(results.OperationResult[sharedtypes.TagNumber, error])
				if err != nil {
					t.Fatalf("expected no system error, got: %v", err)
				}
				if opResult.Success == nil {
					t.Fatalf("expected success payload, got nil")
				}
				tag := *opResult.Success
				if tag != 10 {
					t.Errorf("expected tag 10, got %d", tag)
				}
			},
		},
		{
			name:    "GetTagByUserID: Returns sql.ErrNoRows for missing user",
			guildID: "test_guild_1",
			setupData: func(t *testing.T, db *bun.DB) {
				lb := &leaderboarddb.Leaderboard{
					GuildID: "test_guild_1", IsActive: true,
					LeaderboardData: leaderboardtypes.LeaderboardData{},
				}
				_, _ = db.NewInsert().Model(lb).Exec(ctx)
			},
			runOperation: func(s leaderboardService.Service) (any, error) {
				return s.GetTagByUserID(ctx, "test_guild_1", "ghost_user")
			},
			wantErr: false,
			validate: func(t *testing.T, result any, err error) {
				if err != nil {
					t.Fatalf("expected no system error, got %v", err)
				}
				opResult := result.(results.OperationResult[sharedtypes.TagNumber, error])
				if opResult.Failure == nil {
					t.Fatalf("expected domain failure, got nil")
				}
				if !errors.Is(*opResult.Failure, sql.ErrNoRows) {
					t.Errorf("expected sql.ErrNoRows, got %v", *opResult.Failure)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)

			if tt.setupData != nil {
				tt.setupData(t, deps.BunDB)
			}

			// deps.Service now correctly satisfies the interface signature in runOperation
			res, err := tt.runOperation(deps.Service)

			if (err != nil) != tt.wantErr {
				t.Fatalf("operation error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.validate != nil {
				tt.validate(t, res, err)
			}
		})
	}
}
