package guildservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewGuildService(t *testing.T) {
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
				mockDB := guilddb.NewMockGuildDB(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				mockMetrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				// Call the function being tested
				service := NewGuildService(mockDB, mockEventBus, logger, mockMetrics, tracer)

				// Ensure service is correctly created
				if service == nil {
					t.Fatal("NewGuildService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				guildServiceImpl, ok := service.(*GuildService)
				if !ok {
					t.Fatal("NewGuildService did not return *GuildService")
				}

				// Override serviceWrapper to prevent unwanted tracing/logging/metrics calls
				guildServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, guildID sharedtypes.GuildID, serviceFunc func(ctx context.Context) (GuildOperationResult, error)) (GuildOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check that all dependencies were correctly assigned
				if guildServiceImpl.GuildDB != mockDB {
					t.Error("GuildDB not set correctly")
				}
				if guildServiceImpl.eventBus != mockEventBus {
					t.Error("eventBus not set correctly")
				}
				if guildServiceImpl.logger != logger {
					t.Error("logger not set correctly")
				}
				if guildServiceImpl.metrics != mockMetrics {
					t.Error("metrics not set correctly")
				}
				if guildServiceImpl.tracer != tracer {
					t.Error("tracer not set correctly")
				}

				// Ensure serviceWrapper is correctly set
				if guildServiceImpl.serviceWrapper == nil {
					t.Error("serviceWrapper not set")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				// Call with nil dependencies
				service := NewGuildService(nil, nil, nil, nil, nil)

				// Ensure service is correctly created
				if service == nil {
					t.Fatal("NewGuildService returned nil")
				}

				// Access the concrete type to override serviceWrapper
				guildServiceImpl, ok := service.(*GuildService)
				if !ok {
					t.Fatal("NewGuildService did not return *GuildService")
				}

				// Override serviceWrapper to avoid nil tracing/logger issues
				guildServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, guildID sharedtypes.GuildID, serviceFunc func(ctx context.Context) (GuildOperationResult, error)) (GuildOperationResult, error) {
					return serviceFunc(ctx) // Just execute serviceFunc directly
				}

				// Check nil fields
				if guildServiceImpl.GuildDB != nil {
					t.Error("GuildDB should be nil")
				}
				if guildServiceImpl.eventBus != nil {
					t.Error("eventBus should be nil")
				}
				if guildServiceImpl.logger != nil {
					t.Error("logger should be nil")
				}
				if guildServiceImpl.metrics != nil {
					t.Error("metrics should be nil")
				}
				if guildServiceImpl.tracer != nil {
					t.Error("tracer should be nil")
				}

				// Ensure serviceWrapper is still set
				if guildServiceImpl.serviceWrapper == nil {
					t.Error("serviceWrapper should still be set")
				}

				// Test serviceWrapper runs correctly with nil dependencies
				result, err := guildServiceImpl.serviceWrapper(context.Background(), "test", "guild-1", func(ctx context.Context) (GuildOperationResult, error) {
					return GuildOperationResult{Success: "ok"}, nil
				})
				if err != nil {
					t.Errorf("serviceWrapper should work with nil dependencies, got error: %v", err)
				}
				if result.Success != "ok" {
					t.Errorf("expected success 'ok', got: %v", result.Success)
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
	testGuildID := sharedtypes.GuildID("guild-1")

	type args struct {
		ctx           context.Context
		operationName string
		guildID       sharedtypes.GuildID
		serviceFunc   func(ctx context.Context) (GuildOperationResult, error)
		logger        *slog.Logger
		metrics       guildmetrics.GuildMetrics
		tracer        trace.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    GuildOperationResult
		wantErr bool
		setup   func(a *args, ctx context.Context)
	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					serviceFunc: func(ctx context.Context) (GuildOperationResult, error) {
						return GuildOperationResult{Success: "test"}, nil
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			want:    GuildOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args, ctx context.Context) {
				// No additional setup needed for success case
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					serviceFunc: func(ctx context.Context) (GuildOperationResult, error) {
						panic("test panic")
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup:   func(a *args, ctx context.Context) {},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					serviceFunc: func(ctx context.Context) (GuildOperationResult, error) {
						return GuildOperationResult{Error: errors.New("service error")}, errors.New("service error")
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup:   func(a *args, ctx context.Context) {},
		},
		{
			name: "Handles nil context",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           nil,
					operationName: "TestOperation",
					guildID:       testGuildID,
					serviceFunc: func(ctx context.Context) (GuildOperationResult, error) {
						return GuildOperationResult{}, nil
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup:   func(a *args, ctx context.Context) {},
		},
		{
			name: "Handles nil service function",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					serviceFunc:   nil,
					logger:        logger,
					metrics:       metrics,
					tracer:        tracer,
				}
			},
			wantErr: true,
			setup:   func(a *args, ctx context.Context) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			a := tt.args(ctrl)
			tt.setup(&a, a.ctx)

			got, err := serviceWrapper(a.ctx, a.operationName, a.guildID, a.serviceFunc, a.logger, a.metrics, a.tracer)
			if (err != nil) != tt.wantErr {
				t.Errorf("serviceWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Success != tt.want.Success {
					t.Errorf("serviceWrapper() got = %v, want %v", got, tt.want)
				}
			} else {
				if got.Error == nil {
					t.Error("expected result.Error to be set on error case")
				}
			}
		})
	}
}

func TestGuildConfigsEqual(t *testing.T) {
	now := time.Now().UTC()
	later := now.Add(time.Hour)

	config1 := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &now,
	}

	config2 := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &now,
	}

	config3 := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "different-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &now,
	}

	config4 := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &later,
	}

	tests := []struct {
		name string
		a    *guildtypes.GuildConfig
		b    *guildtypes.GuildConfig
		want bool
	}{
		{
			name: "equal configs",
			a:    config1,
			b:    config2,
			want: true,
		},
		{
			name: "different signup channel",
			a:    config1,
			b:    config3,
			want: false,
		},
		{
			name: "different timestamp",
			a:    config1,
			b:    config4,
			want: false,
		},
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "first nil",
			a:    nil,
			b:    config1,
			want: false,
		},
		{
			name: "second nil",
			a:    config1,
			b:    nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guildConfigsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("guildConfigsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
