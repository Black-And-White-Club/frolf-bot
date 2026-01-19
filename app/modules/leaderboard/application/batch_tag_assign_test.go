package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_ExecuteBatchTagAssignment(t *testing.T) {
	tests := []struct {
		name        string
		mockDBSetup func(*leaderboarddb.MockRepository)
		requests    []sharedtypes.TagAssignmentRequest
		updateID    uuid.UUID
		source      sharedtypes.ServiceUpdateSource
		expectErr   bool
		verify      func(t *testing.T, res results.OperationResult, err error)
	}{
		{
			name: "successful batch updates leaderboard",
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "existing_user", TagNumber: 10}},
				}
				updatedLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "user1", TagNumber: 1}, {UserID: "existing_user", TagNumber: 10}},
				}
				mockDB.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), guildID, gomock.Any(), gomock.Any(), gomock.Any()).Return(updatedLeaderboard, nil)
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			updateID:  uuid.New(),
			source:    sharedtypes.ServiceUpdateSourceAdminBatch,
			expectErr: false,
			verify: func(t *testing.T, res results.OperationResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !res.IsSuccess() {
					t.Fatalf("expected success result, got failure: %v", res.Failure)
				}
				payload, ok := res.Success.(*leaderboardevents.LeaderboardBatchTagAssignedPayloadV1)
				if !ok {
					t.Fatalf("unexpected success payload type: %T", res.Success)
				}
				if len(payload.Assignments) != 2 {
					t.Fatalf("expected 2 entries in leaderboard, got %d", len(payload.Assignments))
				}
			},
		},
		{
			name: "swap needed returns TagSwapNeededError",
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				guildID := sharedtypes.GuildID("test-guild")
				// user1 wants tag 1 but target_user currently holds tag 1
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{{UserID: "user1", TagNumber: 5}, {UserID: "target_user", TagNumber: 1}},
				}
				mockDB.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(currentLeaderboard, nil)
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			updateID:  uuid.New(),
			source:    sharedtypes.ServiceUpdateSourceManual,
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult, err error) {
				if err == nil {
					t.Fatalf("expected TagSwapNeededError, got nil")
				}
				var tsn *TagSwapNeededError
				if !errors.As(err, &tsn) {
					t.Fatalf("expected TagSwapNeededError, got: %T %v", err, err)
				}
			},
		},
		{
			name: "UpdateLeaderboard infrastructure error bubbles up",
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				guildID := sharedtypes.GuildID("test-guild")
				currentLeaderboard := &leaderboarddbtypes.Leaderboard{LeaderboardData: leaderboardtypes.LeaderboardData{}}
				mockDB.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(currentLeaderboard, nil)
				mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), guildID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db failure"))
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			updateID:  uuid.New(),
			source:    sharedtypes.ServiceUpdateSourceAdminBatch,
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult, err error) {
				if err == nil || (!strings.Contains(err.Error(), "failed to commit update") && !strings.Contains(err.Error(), "db failure")) {
					t.Fatalf("expected db error, got: %v", err)
				}
			},
		},
		{
			name: "GetActiveLeaderboardIDB failure bubbles up",
			mockDBSetup: func(mockDB *leaderboarddb.MockRepository) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().GetActiveLeaderboardIDB(gomock.Any(), gomock.Any(), guildID).Return(nil, errors.New("no access"))
			},
			requests:  []sharedtypes.TagAssignmentRequest{{UserID: "user1", TagNumber: 1}},
			updateID:  uuid.New(),
			source:    sharedtypes.ServiceUpdateSourceAdminBatch,
			expectErr: true,
			verify: func(t *testing.T, res results.OperationResult, err error) {
				if err == nil {
					t.Fatalf("expected fetch error, got nil")
				}
				// The service wraps operation errors with the operation name; accept either the original message
				// or the wrapped form to keep tests robust across refactors.
				if !(strings.Contains(err.Error(), "failed to fetch current leaderboard") || strings.Contains(err.Error(), "no access") || strings.Contains(err.Error(), "ExecuteBatchTagAssignment")) {
					t.Fatalf("expected fetch error, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			mockDB := leaderboarddb.NewMockRepository(ctrl)

			// minimal no-op tracer since service requires it
			tracer := noop.NewTracerProvider().Tracer("test")

			if tt.mockDBSetup != nil {
				tt.mockDBSetup(mockDB)
			}

			s := &LeaderboardService{
				repo:    mockDB,
				logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
				metrics: &leaderboardmetrics.NoOpMetrics{},
				tracer:  tracer,
			}

			// call the refactored entrypoint
			res, err := s.ExecuteBatchTagAssignment(ctx, sharedtypes.GuildID("test-guild"), tt.requests, sharedtypes.RoundID(tt.updateID), tt.source)

			if tt.expectErr && err == nil {
				t.Fatalf("expected error but got nil")
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
