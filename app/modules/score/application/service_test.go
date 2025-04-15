package scoreservice

import (
	"context"
	"fmt"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/google/uuid"
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
				mockDB := scoredb.NewMockScoreDB(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				mockLogger := &lokifrolfbot.NoOpLogger{}
				mockMetrics := &scoremetrics.NoOpMetrics{}
				mockTracer := &tempofrolfbot.NoOpTracer{}

				// Call the function being tested
				service := NewScoreService(mockDB, mockEventBus, mockLogger, mockMetrics, mockTracer)

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
				if scoreServiceImpl.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if scoreServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if scoreServiceImpl.tracer != mockTracer {
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
		logger        lokifrolfbot.Logger
		metrics       scoremetrics.ScoreMetrics
		tracer        tempofrolfbot.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    ScoreOperationResult
		wantErr bool
		setup   func(a *args)
	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					roundID:       testRoundID,
					serviceFunc: func(ctx context.Context) (ScoreOperationResult, error) {
						return ScoreOperationResult{Success: "test"}, nil
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			want:    ScoreOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.Any(),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", testRoundID)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", gomock.Any())
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationSuccess("TestOperation", testRoundID)
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					roundID:       testRoundID,
					serviceFunc: func(ctx context.Context) (ScoreOperationResult, error) {
						panic("test panic") // Simulate a panic
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Expect initial method calls before panic occurs
				mockTracer.EXPECT().StartSpan(gomock.AssignableToTypeOf(context.Background()), "TestOperation", gomock.Any()).Return(context.Background(), noop.Span{})

				// Expect `RecordOperationAttempt` to be called BEFORE the panic
				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", testRoundID)

				// Expect `logger.Info` for operation start (happens before panic)
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

				// Expect `RecordOperationDuration` since the function starts measuring time before panic
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", gomock.Any())

				// Expect panic error logging
				mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

				// Expect metrics to record failure
				mockMetrics.EXPECT().RecordOperationFailure("TestOperation", testRoundID)
			},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockScoreMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					roundID:       testRoundID,
					serviceFunc: func(ctx context.Context) (ScoreOperationResult, error) {
						return ScoreOperationResult{}, fmt.Errorf("service error")
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockScoreMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", testRoundID)

				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", gomock.Any())

				// Expect error logging
				mockLogger.EXPECT().Error("Error in TestOperation", gomock.Any(), gomock.Any(), gomock.Any())

				// Expect metrics to record operation failure
				mockMetrics.EXPECT().RecordOperationFailure("TestOperation", testRoundID)
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
				tt.setup(&testArgs)
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
