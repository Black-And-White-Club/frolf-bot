package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"

	"strings"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
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
				mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				// Call the function being tested (pass nil for *bun.DB)
				service := NewLeaderboardService(nil, mockDB, logger, mockMetrics, tracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewLeaderboardService returned nil")
				}

				// service is already a *LeaderboardService
				leaderboardServiceImpl := service

				// Override serviceWrapper to prevent unwanted tracing/logging/metrics calls
				leaderboardServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if leaderboardServiceImpl.LeaderboardDB != mockDB {
					t.Errorf("Leaderboard DB not correctly assigned")
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

				// Ensure serviceWrapper is correctly set
				if leaderboardServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
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

				// Override serviceWrapper to avoid nil tracing/logger issues
				leaderboardServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check nil fields
				if leaderboardServiceImpl.LeaderboardDB != nil {
					t.Errorf("Leaderboard DB should be nil")
				}
				if leaderboardServiceImpl.logger != nil {
					t.Errorf("logger should be nil")
				}
				if leaderboardServiceImpl.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if leaderboardServiceImpl.tracer != nil {
					t.Errorf("tracer should be nil")
				}

				// Ensure serviceWrapper is still set
				if leaderboardServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}

				// Test serviceWrapper runs correctly with nil dependencies
				_, err := leaderboardServiceImpl.serviceWrapper(context.Background(), "TestOp", func(ctx context.Context) (LeaderboardOperationResult, error) {
					return LeaderboardOperationResult{
						Leaderboard: leaderboardtypes.LeaderboardData{},
						TagChanges:  []TagChange{},
						Err:         nil,
					}, nil
				})
				if err != nil {
					t.Errorf("serviceWrapper should execute the provided function without error, got: %v", err)
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_serviceWrapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name        string
		ctx         context.Context
		operation   string
		serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)
		wantResult  LeaderboardOperationResult
		wantErr     error
	}{
		{
			name:      "successful operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (LeaderboardOperationResult, error) {
				return LeaderboardOperationResult{
					Leaderboard: leaderboardtypes.LeaderboardData{{UserID: "test", TagNumber: 1}},
					TagChanges:  []TagChange{},
					Err:         nil,
				}, nil
			},
			wantResult: LeaderboardOperationResult{
				Leaderboard: leaderboardtypes.LeaderboardData{{UserID: "test", TagNumber: 1}},
				TagChanges:  []TagChange{},
				Err:         nil,
			},
			wantErr: nil,
		},
		{
			name:      "failed operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (LeaderboardOperationResult, error) {
				return LeaderboardOperationResult{}, errors.New("test_error")
			},
			wantResult: LeaderboardOperationResult{},
			wantErr:    errors.New("test_operation operation failed: test_error"),
		},
		{
			name:      "panic recovery",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (LeaderboardOperationResult, error) {
				panic("test_panic")
			},
			wantResult: LeaderboardOperationResult{},
			wantErr:    errors.New("Panic in test_operation: test_panic"),
		},
		{
			name:        "nil service function",
			ctx:         context.Background(),
			operation:   "test_operation",
			serviceFunc: nil,
			wantResult:  LeaderboardOperationResult{},
			wantErr:     errors.New("service function is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up expected calls for the mockMetrics
			if tt.name == "successful operation" {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), tt.operation, "LeaderboardService", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
			} else if tt.name == "failed operation" {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), tt.operation, "LeaderboardService", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
			} else if tt.name == "panic recovery" {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), tt.operation, "LeaderboardService", gomock.Any()).Times(1)
				mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), tt.operation, "LeaderboardService").Times(1)
			}

			gotResult, err := serviceWrapper(tt.ctx, tt.operation, tt.serviceFunc, logger, mockMetrics, tracer)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("serviceWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr != nil {
				if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("serviceWrapper() error message = %q, want to contain %q", err.Error(), tt.wantErr.Error())
				}
			}
			if !leaderboardsEqual(gotResult.Leaderboard, tt.wantResult.Leaderboard) {
				t.Errorf("serviceWrapper() Leaderboard = %v, want %v", gotResult.Leaderboard, tt.wantResult.Leaderboard)
			}
			if !tagChangesEqual(gotResult.TagChanges, tt.wantResult.TagChanges) {
				t.Errorf("serviceWrapper() TagChanges = %v, want %v", gotResult.TagChanges, tt.wantResult.TagChanges)
			}
			// Compare Err by nil-ness and message
			if (gotResult.Err == nil) != (tt.wantResult.Err == nil) {
				t.Errorf("serviceWrapper() Err = %v, want %v", gotResult.Err, tt.wantResult.Err)
			} else if gotResult.Err != nil && tt.wantResult.Err != nil {
				if gotResult.Err.Error() != tt.wantResult.Err.Error() {
					t.Errorf("serviceWrapper() Err = %v, want %v", gotResult.Err, tt.wantResult.Err)
				}
			}
		})
	}
}

// Helper to compare leaderboards without reflect
func leaderboardsEqual(a, b leaderboardtypes.LeaderboardData) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].UserID != b[i].UserID || a[i].TagNumber != b[i].TagNumber {
			return false
		}
	}
	return true
}

// Helper to compare tag changes without reflect
func tagChangesEqual(a, b []TagChange) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].UserID != b[i].UserID || a[i].OldTag != b[i].OldTag || a[i].NewTag != b[i].NewTag {
			return false
		}
	}
	return true
}
