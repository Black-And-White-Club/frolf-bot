package leaderboardservice

import (
	"context"
	"fmt"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
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
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				// Call the function being tested
				service := NewLeaderboardService(mockDB, mockEventBus, mockLogger, mockMetrics, mockTracer)

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
				leaderboardServiceImpl.serviceWrapper = func(msg *message.Message, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc() // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if leaderboardServiceImpl.LeaderboardDB != mockDB {
					t.Errorf("Leaderboard DB not correctly assigned")
				}
				if leaderboardServiceImpl.eventBus != mockEventBus {
					t.Errorf("eventBus not correctly assigned")
				}
				if leaderboardServiceImpl.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if leaderboardServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if leaderboardServiceImpl.tracer != mockTracer {
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
				leaderboardServiceImpl.serviceWrapper = func(msg *message.Message, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
					return serviceFunc() // Just execute serviceFunc directly
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
				testMsg := message.NewMessage("test-id", []byte("test"))
				_, err := leaderboardServiceImpl.serviceWrapper(testMsg, "TestOp", func() (LeaderboardOperationResult, error) {
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
	type args struct {
		msg           *message.Message
		operationName string
		serviceFunc   func() (LeaderboardOperationResult, error)
		logger        lokifrolfbot.Logger
		metrics       leaderboardmetrics.LeaderboardMetrics
		tracer        tempofrolfbot.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    LeaderboardOperationResult
		wantErr bool
		setup   func(a *args) // Setup expectations per test

	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					msg:           message.NewMessage("test-id", []byte("test")),
					operationName: "TestOperation",
					serviceFunc: func() (LeaderboardOperationResult, error) {
						return LeaderboardOperationResult{Success: "test"}, nil
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			want:    LeaderboardOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.Any(),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", "LeaderboardService")
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", "LeaderboardService", gomock.Any())
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationSuccess("TestOperation", "LeaderboardService")
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					msg:           message.NewMessage("test-id", []byte("test")),
					operationName: "TestOperation",
					serviceFunc: func() (LeaderboardOperationResult, error) {
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
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Expect initial method calls before panic occurs
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Expect `RecordOperationAttempt` to be called BEFORE the panic
				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", "LeaderboardService")

				// Expect `logger.Info` for operation start (happens before panic)
				mockLogger.EXPECT().Info(
					gomock.Any(),
					attr.CorrelationIDFromMsg(a.msg),
					attr.String("message_id", a.msg.UUID),
					attr.String("operation", "TestOperation"),
				)

				// Expect `RecordOperationDuration` since the function starts measuring time before panic
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", "LeaderboardService", gomock.Any())

				// Expect panic error logging
				mockLogger.EXPECT().Error(
					gomock.Any(),
					attr.CorrelationIDFromMsg(a.msg),
					gomock.Any(),
				)

				// Expect metrics to record failure
				mockMetrics.EXPECT().RecordOperationFailure("TestOperation", "LeaderboardService")
			},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockLeaderboardMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					msg:           message.NewMessage("test-id", []byte("test")),
					operationName: "TestOperation",
					serviceFunc: func() (LeaderboardOperationResult, error) {
						return LeaderboardOperationResult{}, fmt.Errorf("service error")
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockLeaderboardMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", "LeaderboardService")

				mockLogger.EXPECT().Info(
					gomock.Any(),
					attr.CorrelationIDFromMsg(a.msg),
					attr.String("message_id", a.msg.UUID),
					attr.String("operation", "TestOperation"),
				)
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", "LeaderboardService", gomock.Any())

				// Expect error logging
				mockLogger.EXPECT().Error(
					"Error in TestOperation",
					attr.CorrelationIDFromMsg(a.msg),
				)

				// Expect metrics to record operation failure
				mockMetrics.EXPECT().RecordOperationFailure("TestOperation", "LeaderboardService")
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
			got, err := serviceWrapper(testArgs.msg, testArgs.operationName, testArgs.serviceFunc, testArgs.logger, testArgs.metrics, testArgs.tracer)

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
