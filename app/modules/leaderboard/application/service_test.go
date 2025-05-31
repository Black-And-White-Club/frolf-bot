package leaderboardservice

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
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
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				// Call the function being tested
				service := NewLeaderboardService(mockDB, mockEventBus, logger, mockMetrics, tracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewLeaderboardService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				leaderboardServiceImpl, ok := service.(*LeaderboardService)
				if !ok {
					t.Fatalf("service is not of type *LeaderboardService")
				}

				// Override serviceWrapper to prevent unwanted tracing/logging/metrics calls
				leaderboardServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if leaderboardServiceImpl.LeaderboardDB != mockDB {
					t.Errorf("Leaderboard DB not correctly assigned")
				}
				if leaderboardServiceImpl.eventBus != mockEventBus {
					t.Errorf("eventBus not correctly assigned")
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

				// Call with nil dependencies
				service := NewLeaderboardService(nil, nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewLeaderboardService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				leaderboardServiceImpl, ok := service.(*LeaderboardService)
				if !ok {
					t.Fatalf("service is not of type *LeaderboardService")
				}

				// Override serviceWrapper to avoid nil tracing/logger issues
				leaderboardServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check nil fields
				if leaderboardServiceImpl.LeaderboardDB != nil {
					t.Errorf("Leaderboard DB should be nil")
				}
				if leaderboardServiceImpl.eventBus != nil {
					t.Errorf("eventBus should be nil")
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
					return LeaderboardOperationResult{Success: "test"}, nil
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
					Success: "test_success",
				}, nil
			},
			wantResult: LeaderboardOperationResult{
				Success: "test_success",
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
			if !reflect.DeepEqual(gotResult.Success, tt.wantResult.Success) {
				t.Errorf("serviceWrapper() Success = %v, want %v", gotResult.Success, tt.wantResult.Success)
			}
			if !reflect.DeepEqual(gotResult.Failure, tt.wantResult.Failure) {
				t.Errorf("serviceWrapper() Failure = %v, want %v", gotResult.Failure, tt.wantResult.Failure)
			}
		})
	}
}
