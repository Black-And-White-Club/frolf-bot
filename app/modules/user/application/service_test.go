package userservice

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates service with all dependencies",
			test: func(t *testing.T) {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockDB := userdb.NewMockUserDB(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				service := NewUserService(mockDB, mockEventBus, logger, mockMetrics, tracer)

				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				userServiceImpl, ok := service.(*UserServiceImpl)
				if !ok {
					t.Fatalf("service is not of type *UserServiceImpl")
				}

				// Override serviceWrapper to avoid side effects during test
				userServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc(ctx)
				}

				if userServiceImpl.UserDB != mockDB {
					t.Errorf("UserDB not correctly assigned")
				}
				if userServiceImpl.eventBus != mockEventBus {
					t.Errorf("eventBus not correctly assigned")
				}
				if userServiceImpl.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if userServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if userServiceImpl.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}

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

				service := NewUserService(nil, nil, nil, nil, nil)

				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				userServiceImpl, ok := service.(*UserServiceImpl)
				if !ok {
					t.Fatalf("service is not of type *UserServiceImpl")
				}

				userServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc(ctx)
				}

				if userServiceImpl.UserDB != nil {
					t.Errorf("UserDB should be nil")
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

				if userServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}

				ctx := context.Background()
				_, err := userServiceImpl.serviceWrapper(ctx, "TestOp", "123", func(ctx context.Context) (UserOperationResult, error) {
					return UserOperationResult{Success: "test"}, nil
				})
				if err != nil {
					t.Errorf("serviceWrapper should execute the provided function without error, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_serviceWrapper(t *testing.T) {
	type args struct {
		ctx           context.Context
		operationName string
		userID        sharedtypes.DiscordID
		serviceFunc   func(ctx context.Context) (UserOperationResult, error)
		logger        *slog.Logger
		metrics       usermetrics.UserMetrics
		tracer        trace.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    UserOperationResult
		wantErr bool
		setup   func(a *args, ctx context.Context)
	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					serviceFunc: func(ctx context.Context) (UserOperationResult, error) {
						return UserOperationResult{Success: "test"}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			want:    UserOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger

				mockMetrics.EXPECT().RecordOperationAttempt(ctx, "TestOperation", sharedtypes.DiscordID("123"))
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationDuration(ctx, "TestOperation", gomock.Any(), gomock.Any())
				mockLogger.Info("Operation succeeded", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationSuccess(ctx, "TestOperation", sharedtypes.DiscordID("123"))
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					serviceFunc: func(ctx context.Context) (UserOperationResult, error) {
						panic("test panic")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger

				mockMetrics.EXPECT().RecordOperationAttempt(ctx, "TestOperation", sharedtypes.DiscordID("123"))
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationDuration(ctx, "TestOperation", gomock.Any(), gomock.Any())
				mockLogger.Error("Panic in TestOperation: test panic", attr.String("user_id", "123"), attr.Any("panic", "test panic"))
				mockMetrics.EXPECT().RecordOperationFailure(ctx, "TestOperation", sharedtypes.DiscordID("123"))
			},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					serviceFunc: func(ctx context.Context) (UserOperationResult, error) {
						return UserOperationResult{}, fmt.Errorf("service error")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger

				mockMetrics.EXPECT().RecordOperationAttempt(ctx, "TestOperation", sharedtypes.DiscordID("123"))
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationDuration(ctx, "TestOperation", gomock.Any(), gomock.Any())
				mockLogger.Error("Error in TestOperation: service error", attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationFailure(ctx, "TestOperation", sharedtypes.DiscordID("123"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testArgs := tt.args(ctrl)
			ctx, span := testArgs.tracer.Start(testArgs.ctx, testArgs.operationName)
			defer span.End()

			if tt.setup != nil {
				tt.setup(&testArgs, ctx)
			}

			got, err := serviceWrapper(testArgs.ctx, testArgs.operationName, testArgs.userID, testArgs.serviceFunc, testArgs.logger, testArgs.metrics, testArgs.tracer)
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
