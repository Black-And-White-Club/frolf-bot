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

func TestGuildService_UpdateGuildConfig(t *testing.T) {
	ctx := context.Background()
	logger := loggerfrolfbot.NoOpLogger
	metrics := &guildmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	validConfig := &guildtypes.GuildConfig{
		GuildID:         "guild-1",
		SignupChannelID: "signup-chan",
	}

	tests := []struct {
		name           string
		config         *guildtypes.GuildConfig
		setupFake      func(*FakeGuildRepository)
		expectInfraErr bool
		verify         func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository)
	}{
		{
			name:   "success",
			config: validConfig,
			setupFake: func(f *FakeGuildRepository) {
				f.UpdateConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID, updates *guilddb.UpdateFields) error {
					return nil
				}
			},
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil || (*res.Success).GuildID != "guild-1" {
					t.Errorf("expected success payload for guild-1, got %v", res.Success)
				}
				if fake.Trace()[0] != "UpdateConfig" {
					t.Errorf("expected UpdateConfig to be called, got %v", fake.Trace())
				}
			},
		},
		{
			name:   "not found (domain failure)",
			config: &guildtypes.GuildConfig{GuildID: "guild-2"},
			setupFake: func(f *FakeGuildRepository) {
				f.UpdateConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID, updates *guilddb.UpdateFields) error {
					return guilddb.ErrNoRowsAffected
				}
			},
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil || !errors.Is(*res.Failure, ErrGuildConfigNotFound) {
					t.Fatalf("expected domain failure ErrGuildConfigNotFound, got %v", res.Failure)
				}
			},
		},
		{
			name:   "database error (infra failure)",
			config: &guildtypes.GuildConfig{GuildID: "guild-4"},
			setupFake: func(f *FakeGuildRepository) {
				f.UpdateConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID, updates *guilddb.UpdateFields) error {
					return errors.New("update error")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "update error") {
					t.Fatalf("expected update error, got %v", infraErr)
				}
			},
		},
		{
			name:           "invalid guildID (domain failure)",
			config:         &guildtypes.GuildConfig{GuildID: ""},
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infra error: %v", infraErr)
				}
				if res.Failure == nil || !errors.Is(*res.Failure, ErrInvalidGuildID) {
					t.Fatalf("expected domain failure ErrInvalidGuildID, got %v", res.Failure)
				}
				if len(fake.Trace()) > 0 {
					t.Errorf("repo should not have been called for invalid ID")
				}
			},
		},
		{
			name:           "nil config (infra failure)",
			config:         nil,
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if !errors.Is(infraErr, ErrNilConfig) {
					t.Fatalf("expected ErrNilConfig, got %v", infraErr)
				}
				if len(fake.Trace()) > 0 {
					t.Errorf("repo should not have been called for nil config")
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

			res, err := s.UpdateGuildConfig(ctx, tt.config)

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
