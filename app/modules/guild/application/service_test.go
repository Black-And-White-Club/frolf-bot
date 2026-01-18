package guildservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
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

				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockRepo := guilddb.NewMockRepository(ctrl)
				mockMetrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				service := NewGuildService(mockRepo, logger, mockMetrics, tracer)

				if service == nil {
					t.Fatal("NewGuildService returned nil")
				}

				if service.repo != mockRepo {
					t.Error("repo not set correctly")
				}

				if service.logger != logger {
					t.Error("logger not set correctly")
				}
				if service.metrics != mockMetrics {
					t.Error("metrics not set correctly")
				}
				if service.tracer != tracer {
					t.Error("tracer not set correctly")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				service := NewGuildService(nil, nil, nil, nil)

				if service == nil {
					t.Fatal("NewGuildService returned nil")
				}

				if service.repo != nil {
					t.Error("repo should be nil")
				}

				if service.logger != nil {
					t.Error("logger should be nil")
				}
				if service.metrics != nil {
					t.Error("metrics should be nil")
				}
				if service.tracer != nil {
					t.Error("tracer should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestWithTelemetry(t *testing.T) {
	testGuildID := sharedtypes.GuildID("guild-1")

	type args struct {
		ctx           context.Context
		operationName string
		guildID       sharedtypes.GuildID
		op            func(ctx context.Context) (results.OperationResult, error)
		logger        *slog.Logger
		metrics       guildmetrics.GuildMetrics
		tracer        trace.Tracer
	}

	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    results.OperationResult
		wantErr bool
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
					op: func(ctx context.Context) (results.OperationResult, error) {
						return results.SuccessResult("test"), nil
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			want:    results.SuccessResult("test"),
			wantErr: false,
		},
		{
			name: "Handles panic in operation",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					op: func(ctx context.Context) (results.OperationResult, error) {
						panic("test panic")
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
		},
		{
			name: "Handles operation returning an error",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					op: func(ctx context.Context) (results.OperationResult, error) {
						return results.OperationResult{}, errors.New("service error")
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
		},
		{
			name: "Handles nil operation",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &guildmetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					guildID:       testGuildID,
					op:            nil,
					logger:        logger,
					metrics:       metrics,
					tracer:        tracer,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			a := tt.args(ctrl)

			svc := &GuildService{logger: a.logger, metrics: a.metrics, tracer: a.tracer}
			got, err := svc.withTelemetry(a.ctx, a.operationName, a.guildID, a.op)
			if (err != nil) != tt.wantErr {
				t.Errorf("withTelemetry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Success != tt.want.Success {
					t.Errorf("withTelemetry() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestConfigsEqual(t *testing.T) {
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
			got := configsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("configsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
