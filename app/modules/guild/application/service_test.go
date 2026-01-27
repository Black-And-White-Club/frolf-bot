package guildservice

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// -----------------------------------------------------------------------------
// Lifecycle & Helper Tests
// -----------------------------------------------------------------------------

func TestNewGuildService(t *testing.T) {
	fakeRepo := NewFakeGuildRepository()
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")
	// For testing NewGuildService, bun.DB can be nil or a shell
	var db *bun.DB

	service := NewGuildService(fakeRepo, logger, mockMetrics, tracer, db)

	if service == nil {
		t.Fatal("NewGuildService returned nil")
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
}

func Test_withTelemetry(t *testing.T) {
	s := &GuildService{
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &guildmetrics.NoOpMetrics{},
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

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
			operation: "TestOp",
			guildID:   "guild-1",
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
			operation: "TestOp",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.FailureResult[SuccessPayload, FailurePayload](FailurePayload{Reason: "bad_req"}), nil
			},
			checkResult: func(t *testing.T, res results.OperationResult[SuccessPayload, FailurePayload]) {
				if !res.IsFailure() || res.Failure.Reason != "bad_req" {
					t.Errorf("expected failure 'bad_req', got %+v", res.Failure)
				}
			},
		},
		{
			name:      "handles infrastructure error",
			operation: "TestOp",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				return results.OperationResult[SuccessPayload, FailurePayload]{}, errors.New("db down")
			},
			wantErrSub: "TestOp: db down",
		},
		{
			name:      "recovers from panic",
			operation: "TestOp",
			guildID:   "guild-1",
			op: func(ctx context.Context) (results.OperationResult[SuccessPayload, FailurePayload], error) {
				panic("boom")
			},
			wantErrSub: "panic in TestOp: boom",
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
	t.Run("executes directly when db is nil", func(t *testing.T) {
		s := &GuildService{db: nil}

		res, err := runInTx(s, context.Background(), func(ctx context.Context, db bun.IDB) (results.OperationResult[string, string], error) {
			if db != nil {
				return results.OperationResult[string, string]{}, errors.New("expected nil db")
			}
			return results.SuccessResult[string, string]("no_tx"), nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.IsSuccess() || *res.Success != "no_tx" {
			t.Errorf("expected success 'no_tx', got %v", res.Success)
		}
	})
}
