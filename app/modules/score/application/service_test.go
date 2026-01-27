package scoreservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// -----------------------------------------------------------------------------
// Lifecycle & Helper Tests
// -----------------------------------------------------------------------------

func TestNewScoreService(t *testing.T) {
	fakeRepo := NewFakeScoreRepository()
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := &scoremetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")
	var db *bun.DB // bun.DB is fine as nil for simple assignment check

	service := NewScoreService(fakeRepo, nil, logger, mockMetrics, tracer, db)

	if service == nil {
		t.Fatal("NewScoreService returned nil")
	}
	if service.repo != fakeRepo {
		t.Error("repo not set correctly")
	}
	if service.logger != logger {
		t.Error("logger not set correctly")
	}
	if service.metrics != mockMetrics {
		t.Error("metrics not set correctly")
	}
	if service.db != db {
		t.Error("db not set correctly")
	}
}

func Test_withTelemetry(t *testing.T) {
	s := &ScoreService{
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &scoremetrics.NoOpMetrics{},
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	type SuccessPayload struct{ Data string }
	type FailurePayload struct{ Reason string }

	tests := []struct {
		name        string
		operation   string
		roundID     sharedtypes.RoundID
		op          operationFunc[SuccessPayload, FailurePayload]
		wantErrSub  string
		checkResult func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload])
	}{
		{
			name:      "handles success result",
			operation: "ProcessScores",
			roundID:   sharedtypes.RoundID(uuid.New()),
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.SuccessResult[SuccessPayload, FailurePayload](SuccessPayload{Data: "ok"}), nil
			},
			checkResult: func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload]) {
				if !res.IsSuccess() || res.Success.Data != "ok" {
					t.Errorf("expected success 'ok', got %+v", res.Success)
				}
			},
		},
		{
			name:      "handles domain failure result",
			operation: "ProcessScores",
			roundID:   sharedtypes.RoundID(uuid.New()),
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.FailureResult[SuccessPayload, FailurePayload](FailurePayload{Reason: "invalid"}), nil
			},
			checkResult: func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload]) {
				if !res.IsFailure() || res.Failure.Reason != "invalid" {
					t.Errorf("expected failure 'invalid', got %+v", res.Failure)
				}
			},
		},
		{
			name:      "handles infrastructure error",
			operation: "ProcessScores",
			roundID:   sharedtypes.RoundID(uuid.New()),
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.OperationResult[SuccessPayload, FailurePayload]{}, errors.New("db connection lost")
			},
			wantErrSub: "ProcessScores: db connection lost",
		},
		{
			name:      "recovers from panic",
			operation: "ProcessScores",
			roundID:   sharedtypes.RoundID(uuid.New()),
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				panic("unexpected state")
			},
			wantErrSub: "panic in ProcessScores: unexpected state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := withTelemetry(s, context.Background(), tt.operation, tt.roundID, tt.op)

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
	t.Run("executes directly when db is nil", func(t *testing.T) {
		s := &ScoreService{db: nil}

		res, err := runInTx(s, context.Background(), func(ctx context.Context, db bun.IDB) (results.OperationResult[string, string], error) {
			if db != nil {
				return results.OperationResult[string, string]{}, errors.New("expected nil db when s.db is nil")
			}
			return results.SuccessResult[string, string]("no_tx_success"), nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsSuccess() || *res.Success != "no_tx_success" {
			t.Errorf("expected success 'no_tx_success', got %v", res.Success)
		}
	})
}
