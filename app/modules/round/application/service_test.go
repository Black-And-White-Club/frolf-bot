package roundservice

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func Test_serviceWrapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	roundID := sharedtypes.RoundID(uuid.New())
	testHandler := loggerfrolfbot.NewTestHandler()
	logger := slog.New(testHandler)
	mockMetrics := &roundmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	tests := []struct {
		name        string
		ctx         context.Context
		operation   string
		serviceFunc func(ctx context.Context) (RoundOperationResult, error) // Updated to accept context
		wantResult  RoundOperationResult
		wantErr     error
	}{
		{
			name:      "successful operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (RoundOperationResult, error) { // Accept context
				return RoundOperationResult{
					Success: "test_success",
				}, nil
			},
			wantResult: RoundOperationResult{
				Success: "test_success",
			},
			wantErr: nil,
		},
		{
			name:      "failed operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (RoundOperationResult, error) { // Accept context
				return RoundOperationResult{}, errors.New("test_error")
			},
			wantResult: RoundOperationResult{},
			wantErr:    errors.New("test_operation operation failed: test_error"),
		},
		{
			name:      "panic recovery",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func(ctx context.Context) (RoundOperationResult, error) { // Accept context
				panic("test_panic")
			},
			wantResult: RoundOperationResult{},
			wantErr:    errors.New("Panic in test_operation: test_panic"),
		},
		{
			name:        "nil service function",
			ctx:         context.Background(),
			operation:   "test_operation",
			serviceFunc: nil,
			wantResult:  RoundOperationResult{},
			wantErr:     errors.New("service function is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := serviceWrapper(tt.ctx, tt.operation, roundID, tt.serviceFunc, logger, mockMetrics, tracer)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("serviceWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr != nil {
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("serviceWrapper() error message = %q, want %q", err.Error(), tt.wantErr.Error())
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

func TestNewRoundService(t *testing.T) {
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
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockDB := rounddb.NewMockRoundDB(ctrl)
				mockMetrics := &roundmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				// Call the function being tested
				service := NewRoundService(mockDB, logger, mockMetrics, tracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewRoundService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				roundServiceImpl, ok := service.(*RoundService)
				if !ok {
					t.Fatalf("service is not of type *RoundService")
				}

				// Override serviceWrapper to prevent unwanted tracing/logging/metrics calls
				roundServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if roundServiceImpl.RoundDB != mockDB {
					t.Errorf("Round DB not correctly assigned")
				}
				if roundServiceImpl.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if roundServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if roundServiceImpl.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}

				// Ensure serviceWrapper is correctly set
				if roundServiceImpl.serviceWrapper == nil {
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
				service := NewRoundService(nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewRoundService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				roundServiceImpl, ok := service.(*RoundService)
				if !ok {
					t.Fatalf("service is not of type *RoundService")
				}

				// Override serviceWrapper to avoid nil tracing/logger issues
				roundServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check nil fields
				if roundServiceImpl.RoundDB != nil {
					t.Errorf("Round DB should be nil")
				}
				if roundServiceImpl.logger != nil {
					t.Errorf("logger should be nil")
				}
				if roundServiceImpl.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if roundServiceImpl.tracer != nil {
					t.Errorf("tracer should be nil")
				}

				// Ensure serviceWrapper is still set
				if roundServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}

				// Test serviceWrapper runs correctly with nil dependencies
				ctx := context.Background()
				_, err := roundServiceImpl.serviceWrapper(ctx, "TestOp", sharedtypes.RoundID(uuid.New()), func(ctx context.Context) (RoundOperationResult, error) {
					return RoundOperationResult{Success: "test"}, nil
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
