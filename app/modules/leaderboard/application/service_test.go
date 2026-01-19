package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"

	"strings"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardService(t *testing.T) {
	// Define test cases
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates service with all dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Create mock dependencies
				mockDB := leaderboarddb.NewMockRepository(ctrl)
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := &leaderboardmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				// Call the function being tested (pass nil for *bun.DB)
				service := NewLeaderboardService(nil, mockDB, logger, mockMetrics, tracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewLeaderboardService returned nil")
				}

				// service is already a *LeaderboardService
				leaderboardServiceImpl := service

				// Check that all dependencies were correctly assigned
				if leaderboardServiceImpl.repo != mockDB {
					t.Errorf("repo not correctly assigned")
				}
				if leaderboardServiceImpl.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if leaderboardServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if leaderboardServiceImpl.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
				// withTelemetry is provided on the service and should not panic when used
				_, err := leaderboardServiceImpl.withTelemetry(context.Background(), "TestOp", sharedtypes.GuildID("test-guild"), func(ctx context.Context) (results.OperationResult, error) {
					return results.SuccessResult(&leaderboardevents.GetLeaderboardResponsePayloadV1{}), nil
				})
				if err != nil {
					t.Errorf("withTelemetry returned unexpected error: %v", err)
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Call with nil dependencies (pass nil for *bun.DB)
				service := NewLeaderboardService(nil, nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewLeaderboardService returned nil")
				}

				// service is already a *LeaderboardService
				leaderboardServiceImpl := service

				// Check nil fields
				if leaderboardServiceImpl.repo != nil {
					t.Errorf("repo should be nil")
				}
				if leaderboardServiceImpl.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if leaderboardServiceImpl.tracer != nil {
					t.Errorf("tracer should be nil")
				}

				// Test withTelemetry runs correctly with nil dependencies
				_, err := leaderboardServiceImpl.withTelemetry(context.Background(), "TestOp", sharedtypes.GuildID("test-guild"), func(ctx context.Context) (results.OperationResult, error) {
					return results.SuccessResult(&leaderboardevents.GetLeaderboardResponsePayloadV1{}), nil
				})
				if err != nil {
					t.Errorf("withTelemetry should execute the provided function without error, got: %v", err)
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_withTelemetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	tracer := noop.NewTracerProvider().Tracer("test")

	// Use a mock metrics implementation for verifying expected metric calls
	mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)

	s := &LeaderboardService{
		repo:    nil,
		logger:  logger,
		metrics: mockMetrics,
		tracer:  tracer,
		db:      nil,
	}

	tests := []struct {
		name        string
		ctx         context.Context
		operation   string
		serviceFunc func(ctx context.Context) (results.OperationResult, error)
		wantErrSub  string // substring expected in error, empty for success
		verify      func(t *testing.T, res results.OperationResult)
	}{
		{
			name:      "successful operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (results.OperationResult, error) {
				return results.SuccessResult(&leaderboardevents.GetLeaderboardResponsePayloadV1{
					Leaderboard: leaderboardtypes.LeaderboardData{{UserID: "test", TagNumber: 1}},
				}), nil
			},
			wantErrSub: "",
			verify: func(t *testing.T, res results.OperationResult) {
				if !res.IsSuccess() {
					t.Fatalf("expected success result, got failure: %v", res.Failure)
				}
				payload, ok := res.Success.(*leaderboardevents.GetLeaderboardResponsePayloadV1)
				if !ok {
					t.Fatalf("unexpected success payload type: %T", res.Success)
				}
				if len(payload.Leaderboard) != 1 || payload.Leaderboard[0].UserID != "test" {
					t.Fatalf("unexpected leaderboard payload: %#v", payload.Leaderboard)
				}
			},
		},
		{
			name:      "failed operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (results.OperationResult, error) {
				return results.OperationResult{}, errors.New("test_error")
			},
			wantErrSub: "test_operation: test_error",
			verify:     func(t *testing.T, res results.OperationResult) {},
		},
		{
			name:      "panic recovery",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (results.OperationResult, error) {
				panic("test_panic")
			},
			wantErrSub: "panic in test_operation",
			verify:     func(t *testing.T, res results.OperationResult) {},
		},
		{
			name:        "nil service function",
			ctx:         context.Background(),
			operation:   "test_operation",
			serviceFunc: nil,
			wantErrSub:  "panic",
			verify:      func(t *testing.T, res results.OperationResult) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up expected calls for the mockMetrics
			if tt.name == "successful operation" {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), tt.operation, "LeaderboardService", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
			} else {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), tt.operation, "LeaderboardService", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
			}

			res, err := s.withTelemetry(tt.ctx, tt.operation, sharedtypes.GuildID("test-guild"), tt.serviceFunc)
			if tt.wantErrSub == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				tt.verify(t, res)
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("expected error to contain %q, got %q", tt.wantErrSub, err.Error())
				}
			}
		})
	}
}
