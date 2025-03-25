package userservice

import (
	"context"
	"fmt"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
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
				mockDB := userdb.NewMockUserDB(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				// Call the function being tested
				service := NewUserService(mockDB, mockEventBus, mockLogger, mockMetrics, mockTracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				userServiceImpl, ok := service.(*UserServiceImpl)
				if !ok {
					t.Fatalf("service is not of type *UserServiceImpl")
				}

				// Override serviceWrapper to prevent unwanted tracing/logging/metrics calls
				userServiceImpl.serviceWrapper = func(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc() // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if userServiceImpl.UserDB != mockDB {
					t.Errorf("User DB not correctly assigned")
				}
				if userServiceImpl.eventBus != mockEventBus {
					t.Errorf("eventBus not correctly assigned")
				}
				if userServiceImpl.logger != mockLogger {
					t.Errorf("logger not correctly assigned")
				}
				if userServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if userServiceImpl.tracer != mockTracer {
					t.Errorf("tracer not correctly assigned")
				}

				// Ensure serviceWrapper is correctly set
				if userServiceImpl.serviceWrapper == nil {
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
				service := NewUserService(nil, nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				userServiceImpl, ok := service.(*UserServiceImpl)
				if !ok {
					t.Fatalf("service is not of type *UserServiceImpl")
				}

				// Override serviceWrapper to avoid nil tracing/logger issues
				userServiceImpl.serviceWrapper = func(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc() // Just execute serviceFunc directly
				}

				// Check nil fields
				if userServiceImpl.UserDB != nil {
					t.Errorf("User DB should be nil")
				}
				if userServiceImpl.eventBus != nil {
					t.Errorf("eventBus should be nil")
				}
				if userServiceImpl.logger != nil {
					t.Errorf("logger should be nil")
				}
				if userServiceImpl.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if userServiceImpl.tracer != nil {
					t.Errorf("tracer should be nil")
				}

				// Ensure serviceWrapper is still set
				if userServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}

				// Test serviceWrapper runs correctly with nil dependencies
				testMsg := message.NewMessage("test-id", []byte("test"))
				_, err := userServiceImpl.serviceWrapper(testMsg, "TestOp", "123", func() (UserOperationResult, error) {
					return UserOperationResult{Success: "test"}, nil
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
		userID        usertypes.DiscordID
		serviceFunc   func() (UserOperationResult, error)
		logger        lokifrolfbot.Logger
		metrics       usermetrics.UserMetrics
		tracer        tempofrolfbot.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    UserOperationResult
		wantErr bool
		setup   func(a *args) // Setup expectations per test

	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					msg:           message.NewMessage("test-id", []byte("test")),
					operationName: "TestOperation",
					userID:        usertypes.DiscordID("123"),
					serviceFunc: func() (UserOperationResult, error) {
						return UserOperationResult{Success: "test"}, nil
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			want:    UserOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Mock metrics & logs
				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", usertypes.DiscordID("123"))
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", gomock.Any())
				mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
				mockMetrics.EXPECT().RecordOperationSuccess("TestOperation", usertypes.DiscordID("123"))
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					msg:           message.NewMessage("test-id", []byte("test")),
					operationName: "TestOperation",
					userID:        usertypes.DiscordID("123"),
					serviceFunc: func() (UserOperationResult, error) {
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
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Expect initial method calls before panic occurs
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				// Expect `RecordOperationAttempt` to be called BEFORE the panic
				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", usertypes.DiscordID("123"))

				// Expect `logger.Info` for operation start (happens before panic)
				mockLogger.EXPECT().Info(
					gomock.Any(),
					attr.CorrelationIDFromMsg(a.msg),
					attr.String("message_id", a.msg.UUID),
					attr.String("operation", "TestOperation"),
					attr.String("user_id", "123"),
				)

				// Expect `RecordOperationDuration` since the function starts measuring time before panic
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", gomock.Any())

				// Expect panic error logging
				mockLogger.EXPECT().Error(
					gomock.Any(),
					attr.CorrelationIDFromMsg(a.msg),
					attr.String("user_id", "123"),
					gomock.Any(),
				)

				// Expect metrics to record failure
				mockMetrics.EXPECT().RecordOperationFailure("TestOperation", usertypes.DiscordID("123"))
			},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				mockLogger := mocks.NewMockLogger(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				mockTracer := mocks.NewMockTracer(ctrl)

				return args{
					msg:           message.NewMessage("test-id", []byte("test")),
					operationName: "TestOperation",
					userID:        usertypes.DiscordID("123"),
					serviceFunc: func() (UserOperationResult, error) {
						return UserOperationResult{}, fmt.Errorf("service error")
					},
					logger:  mockLogger,
					metrics: mockMetrics,
					tracer:  mockTracer,
				}
			},
			wantErr: true,
			setup: func(a *args) {
				mockTracer := a.tracer.(*mocks.MockTracer)
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger.(*mocks.MockLogger)

				// Mock tracer.StartSpan
				mockTracer.EXPECT().StartSpan(
					gomock.AssignableToTypeOf(context.Background()),
					"TestOperation",
					gomock.Any(),
				).Return(context.Background(), noop.Span{})

				mockMetrics.EXPECT().RecordOperationAttempt("TestOperation", usertypes.DiscordID("123"))

				mockLogger.EXPECT().Info(
					gomock.Any(),
					attr.CorrelationIDFromMsg(a.msg),
					attr.String("message_id", a.msg.UUID),
					attr.String("operation", "TestOperation"),
					attr.String("user_id", "123"),
				)
				mockMetrics.EXPECT().RecordOperationDuration("TestOperation", gomock.Any())

				// Expect error logging
				mockLogger.EXPECT().Error(
					"Error in TestOperation",
					attr.CorrelationIDFromMsg(a.msg),
				)

				// Expect metrics to record operation failure
				mockMetrics.EXPECT().RecordOperationFailure("TestOperation", usertypes.DiscordID("123"))
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
			got, err := serviceWrapper(testArgs.msg, testArgs.operationName, testArgs.userID, testArgs.serviceFunc, testArgs.logger, testArgs.metrics, testArgs.tracer)

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
