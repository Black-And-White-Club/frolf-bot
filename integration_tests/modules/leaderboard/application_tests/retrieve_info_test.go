package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/uptrace/bun"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
				opResult := result.(leaderboardService.LeaderboardOperationResult)
				if opResult.Err != nil {
					t.Errorf("expected no internal error, got %v", opResult.Err)
				}
				if len(opResult.Leaderboard) != 1 {
					t.Errorf("expected 1 entry, got %d", len(opResult.Leaderboard))
				}
			},
		},
		{
			name:    "GetLeaderboard: Returns error in result for missing leaderboard",
			guildID: "empty_guild",
			runOperation: func(s leaderboardService.Service) (any, error) {
				return s.GetLeaderboard(ctx, "empty_guild")
			},
			validate: func(t *testing.T, result any, err error) {
				opResult := result.(leaderboardService.LeaderboardOperationResult)
				if !errors.Is(opResult.Err, leaderboarddb.ErrNoActiveLeaderboard) {
					t.Errorf("expected ErrNoActiveLeaderboard in result.Err, got %v", opResult.Err)
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
				payload := sharedevents.RoundTagLookupRequestedPayloadV1{UserID: "round_user"}
				return s.RoundGetTagByUserID(ctx, "test_guild_1", payload)
			},
			validate: func(t *testing.T, result any, err error) {
				opResult := result.(leaderboardService.LeaderboardOperationResult)
				if len(opResult.Leaderboard) != 1 || opResult.Leaderboard[0].TagNumber != 42 {
					t.Errorf("expected tag 42, got: %v", opResult.Leaderboard)
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
				tag := result.(sharedtypes.TagNumber)
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
			wantErr: true,
			validate: func(t *testing.T, result any, err error) {
				if !errors.Is(err, sql.ErrNoRows) {
					t.Errorf("expected sql.ErrNoRows, got %v", err)
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
