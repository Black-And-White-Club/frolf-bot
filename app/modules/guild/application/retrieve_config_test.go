package guildservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGuildService_GetGuildConfig(t *testing.T) {
	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	metrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	validConfig := &guildtypes.GuildConfig{
		GuildID: "guild-1",
	}

	tests := []struct {
		name           string
		guildID        sharedtypes.GuildID
		setupFake      func(*FakeGuildRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository)
	}{
		{
			name:    "success",
			guildID: "guild-1",
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return validConfig, nil
				}
			},
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil || (*res.Success).GuildID != "guild-1" {
					t.Errorf("expected config for guild-1, got %v", res.Success)
				}
			},
		},
		{
			name:    "not found (domain failure)",
			guildID: "guild-2",
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return nil, guilddb.ErrNotFound
				}
			},
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil || !errors.Is(*res.Failure, ErrGuildConfigNotFound) {
					t.Errorf("expected domain failure ErrGuildConfigNotFound, got %v", res.Failure)
				}
			},
		},
		{
			name:    "database error (infra failure)",
			guildID: "guild-3",
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return nil, errors.New("connection refused")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "connection refused") {
					t.Errorf("expected infra error containing 'connection refused', got %v", infraErr)
				}
			},
		},
		{
			name:           "invalid guildID (domain failure)",
			guildID:        "",
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if res.Failure == nil || !errors.Is(*res.Failure, ErrInvalidGuildID) {
					t.Errorf("expected domain failure ErrInvalidGuildID, got %v", res.Failure)
				}
				trace := fake.Trace()
				if len(trace) > 0 {
					t.Errorf("expected no repository calls for invalid ID, but got %v", trace)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeGuildRepository()
			if tt.setupFake != nil {
				tt.setupFake(fakeRepo)
			}

			s := &GuildService{
				repo:    fakeRepo,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
			}

			res, err := s.GetGuildConfig(ctx, tt.guildID)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infrastructure error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infrastructure error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, fakeRepo)
			}
		})
	}
}
