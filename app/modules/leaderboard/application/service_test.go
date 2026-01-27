package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// -----------------------------------------------------------------------------
// Lifecycle & Helper Tests
// -----------------------------------------------------------------------------

func TestNewLeaderboardService(t *testing.T) {
	fakeRepo := NewFakeLeaderboardRepo()
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewLeaderboardService(nil, fakeRepo, logger, mockMetrics, tracer)

	if service == nil {
		t.Fatalf("NewLeaderboardService returned nil")
	}
	if service.repo != fakeRepo {
		t.Errorf("repo not correctly assigned")
	}
	if service.logger != logger {
		t.Errorf("logger not correctly assigned")
	}
}

func Test_withTelemetry(t *testing.T) {
	// Setup service with no-ops
	s := &LeaderboardService{
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &leaderboardmetrics.NoOpMetrics{},
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	// Define dummy types for the generic S and F
	type SuccessPayload struct{ Data string }
	type FailurePayload struct{ Reason string }

	tests := []struct {
		name        string
		operation   string
		guildID     sharedtypes.GuildID
		op          operationFunc[SuccessPayload, FailurePayload]
		wantErrSub  string
		checkResult func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload])
	}{
		{
			name:      "handles success result",
			operation: "test_success",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.SuccessResult[SuccessPayload, FailurePayload](SuccessPayload{Data: "ok"}), nil
			},
			checkResult: func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload]) {
				if !res.IsSuccess() || res.Success.Data != "ok" {
					t.Errorf("expected success result 'ok', got %+v", res.Success)
				}
			},
		},
		{
			name:      "handles domain failure result",
			operation: "test_domain_failure",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.FailureResult[SuccessPayload, FailurePayload](FailurePayload{Reason: "denied"}), nil
			},
			checkResult: func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload]) {
				if !res.IsFailure() || res.Failure.Reason != "denied" {
					t.Errorf("expected failure result 'denied', got %+v", res.Failure)
				}
			},
		},
		{
			name:      "handles infrastructure error",
			operation: "test_infra_error",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.OperationResult[SuccessPayload, FailurePayload]{}, errors.New("db connection lost")
			},
			wantErrSub: "test_infra_error: db connection lost",
		},
		{
			name:      "recovers from panic",
			operation: "test_panic",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				panic("something exploded")
			},
			wantErrSub: "panic in test_panic: something exploded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := withTelemetry(s, context.Background(), tt.operation, tt.guildID, tt.op)

			if tt.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("expected error containing %q, got %v", tt.wantErrSub, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.checkResult != nil {
					tt.checkResult(t, res)
				}
			}
		})
	}
}

func Test_runInTx(t *testing.T) {
	type Success = string
	type Failure = string

	t.Run("executes directly when db is nil", func(t *testing.T) {
		s := &LeaderboardService{db: nil}

		res, err := runInTx(s, context.Background(), func(ctx context.Context, db bun.IDB) (results.OperationResult[Success, Failure], error) {
			if db != nil {
				return results.OperationResult[Success, Failure]{}, errors.New("expected nil db")
			}
			return results.SuccessResult[Success, Failure]("executed_no_tx"), nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsSuccess() || *res.Success != "executed_no_tx" {
			t.Errorf("expected success 'executed_no_tx', got %v", res.Success)
		}
	})

	// Note: Testing actual transaction behavior (db.RunInTx) usually requires
	// a mock DB (sqlmock) or an integration test.
	// This test focuses on the nil-guard logic you provided.
}

func TestLeaderboardService_EnsureGuildLeaderboard(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name          string
		setupFake     func(*FakeLeaderboardRepo)
		wantErr       bool
		expectedSteps []string
	}{
		{
			name: "Leaderboard exists - do nothing",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return &leaderboarddb.Leaderboard{}, nil
				}
			},
			wantErr:       false,
			expectedSteps: []string{"GetActiveLeaderboard"},
		},
		{
			name: "Leaderboard missing - create it",
			setupFake: func(f *FakeLeaderboardRepo) {
				f.GetActiveLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID) (*leaderboarddb.Leaderboard, error) {
					return nil, leaderboarddb.ErrNoActiveLeaderboard
				}
				f.CreateLeaderboardFunc = func(ctx context.Context, db bun.IDB, g sharedtypes.GuildID, lb *leaderboarddb.Leaderboard) (*leaderboarddb.Leaderboard, error) {
					return lb, nil
				}
			},
			wantErr:       false,
			expectedSteps: []string{"GetActiveLeaderboard", "CreateLeaderboard"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeLeaderboardRepo()
			tt.setupFake(fakeRepo)
			s := &LeaderboardService{
				repo:   fakeRepo,
				logger: loggerfrolfbot.NoOpLogger,
				db:     nil,
			}

			err := s.EnsureGuildLeaderboard(ctx, guildID)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureGuildLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
			}

			trace := fakeRepo.Trace()
			if len(trace) != len(tt.expectedSteps) {
				t.Errorf("expected %d steps, got %d: %v", len(tt.expectedSteps), len(trace), trace)
			}
		})
	}
}
