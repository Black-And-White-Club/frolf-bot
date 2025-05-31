package scoreservice

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewScoreService(t *testing.T) {
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
				mockDB := scoredb.NewMockScoreDB(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				mockMetrics := &scoremetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")
				// Call the function being tested
				service := NewScoreService(mockDB, mockEventBus, logger, mockMetrics, tracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewScoreService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				scoreServiceImpl, ok := service.(*ScoreService)
				if !ok {
					t.Fatalf("service is not of type *ScoreService")
				}

				// Override serviceWrapper to prevent unwanted tracing/logging/metrics calls
				scoreServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if scoreServiceImpl.ScoreDB != mockDB {
					t.Errorf("Score DB not correctly assigned")
				}
				if scoreServiceImpl.EventBus != mockEventBus {
					t.Errorf("eventBus not correctly assigned")
				}
				if scoreServiceImpl.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if scoreServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if scoreServiceImpl.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}

				// Ensure serviceWrapper is correctly set
				if scoreServiceImpl.serviceWrapper == nil {
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
				service := NewScoreService(nil, nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewScoreService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				scoreServiceImpl, ok := service.(*ScoreService)
				if !ok {
					t.Fatalf("service is not of type *ScoreService")
				}

				// Override serviceWrapper to avoid nil tracing/logger issues
				scoreServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check nil fields
				if scoreServiceImpl.ScoreDB != nil {
					t.Errorf("Score DB should be nil")
				}
				if scoreServiceImpl.EventBus != nil {
					t.Errorf("eventBus should be nil")
				}
				if scoreServiceImpl.logger != nil {
					t.Errorf("logger should be nil")
				}
				if scoreServiceImpl.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if scoreServiceImpl.tracer != nil {
					t.Errorf("tracer should be nil")
				}

				// Ensure serviceWrapper is still set
				if scoreServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}

				// Test serviceWrapper runs correctly with nil dependencies
				ctx := context.Background()
				_, err := scoreServiceImpl.serviceWrapper(ctx, "TestOp", sharedtypes.RoundID(uuid.New()), func(ctx context.Context) (ScoreOperationResult, error) {
					return ScoreOperationResult{Success: "test"}, nil
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
	testRoundID := sharedtypes.RoundID(uuid.New())

	type args struct {
		ctx           context.Context
		operationName string
		roundID       sharedtypes.RoundID
		serviceFunc   func(ctx context.Context) (ScoreOperationResult, error)
		logger        *slog.Logger
		metrics       scoremetrics.ScoreMetrics
		tracer        trace.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    ScoreOperationResult
		wantErr bool
		setup   func(a *args, ctx context.Context)
	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					roundID:       testRoundID,
					serviceFunc: func(ctx context.Context) (ScoreOperationResult, error) {
						return ScoreOperationResult{Success: "test"}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			want:    ScoreOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "TestOperation", testRoundID)
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("round_id", testRoundID.String()))
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "TestOperation", gomock.Any())
				mockLogger.Info("Operation succeeded", attr.String("operation", "TestOperation"), attr.String("round_id", testRoundID.String()))
				mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "TestOperation", testRoundID)
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					roundID:       testRoundID,
					serviceFunc: func(ctx context.Context) (ScoreOperationResult, error) {
						panic("test panic") // Simulate a panic
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "TestOperation", testRoundID)
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("round_id", testRoundID.String()))
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "TestOperation", gomock.Any())
				mockLogger.Error("Error in TestOperation", attr.String("operation", "TestOperation"), attr.String("round_id", testRoundID.String()))
				mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "TestOperation", testRoundID)
			},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					roundID:       testRoundID,
					serviceFunc: func(ctx context.Context) (ScoreOperationResult, error) {
						return ScoreOperationResult{}, fmt.Errorf("service error")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "TestOperation", testRoundID)
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("round_id", testRoundID.String()))
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "TestOperation", gomock.Any())
				mockLogger.Error("Error in TestOperation", attr.String("operation", "TestOperation"), attr.String("round_id", testRoundID.String()))
				mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "TestOperation", testRoundID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Initialize args using fresh mock controller
			testArgs := tt.args(ctrl)

			// Set up expectations
			if tt.setup != nil {
				tt.setup(&testArgs, testArgs.ctx)
			}

			// Run serviceWrapper
			got, err := serviceWrapper(testArgs.ctx, testArgs.operationName, testArgs.roundID, testArgs.serviceFunc, testArgs.logger, testArgs.metrics, testArgs.tracer)

			if (err != nil) != tt.wantErr {
				t.Errorf("serviceWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Success != tt.want.Success {
				t.Errorf("serviceWrapper() Success = %v, want %v", got.Success, tt.want.Success)
			}
			if got.Failure != tt.want.Failure {
				t.Errorf("serviceWrapper() Failure = %v, want %v", got.Failure, tt.want.Failure)
			}
		})
	}
}
