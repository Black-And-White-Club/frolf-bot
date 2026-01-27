package guildservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func TestGuildService_CreateGuildConfig(t *testing.T) {
	setupTime := time.Now().UTC()
	validConfig := &guildtypes.GuildConfig{
		GuildID:              "guild-1",
		SignupChannelID:      "signup-chan",
		EventChannelID:       "event-chan",
		LeaderboardChannelID: "leaderboard-chan",
		UserRoleID:           "role-1",
		SignupEmoji:          ":frolf:",
		AutoSetupCompleted:   true,
		SetupCompletedAt:     &setupTime,
	}

	tests := []struct {
		name           string
		setupFake      func(*FakeGuildRepository)
		config         *guildtypes.GuildConfig
		expectInfraErr bool // For network/db connection errors
		verify         func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository)
	}{
		{
			name:   "success",
			config: validConfig, // Added: Ensure config is not nil
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return nil, guilddb.ErrNotFound // Ensure it doesn't exist yet
				}
			},
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Fatalf("unexpected infrastructure error: %v", infraErr)
				}
				if res.Failure != nil {
					t.Fatalf("expected domain success, got failure: %v", *res.Failure)
				}
				if res.Success == nil || (*res.Success).GuildID != "guild-1" {
					t.Errorf("expected success payload for guild-1, got %v", res.Success)
				}
			},
		},
		{
			name:   "conflict - guild already exists",
			config: validConfig, // Added: Ensure config is not nil
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return validConfig, nil // Return existing config to trigger conflict logic
				}
			},
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if res.Failure == nil || !errors.Is(*res.Failure, ErrGuildConfigConflict) {
					t.Fatalf("expected domain failure ErrGuildConfigConflict, got: %v", res.Failure)
				}
			},
		},
		{
			name: "infrastructure error - db failure on GetConfig",
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return nil, errors.New("connection reset")
				}
			},
			config:         validConfig,
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "connection reset") {
					t.Fatalf("expected connection reset error, got: %v", infraErr)
				}
			},
		},
		{
			name: "infrastructure error - db failure on SaveConfig",
			setupFake: func(f *FakeGuildRepository) {
				f.GetConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return nil, guilddb.ErrNotFound
				}
				f.SaveConfigFunc = func(ctx context.Context, db bun.IDB, cfg *guildtypes.GuildConfig) error {
					return errors.New("disk full")
				}
			},
			config:         validConfig,
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "disk full") {
					t.Fatalf("expected disk full error, got: %v", infraErr)
				}
			},
		},
		{
			name:           "domain failure - nil config",
			config:         nil,
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if !errors.Is(infraErr, ErrNilConfig) {
					t.Fatalf("expected ErrNilConfig, got %v", infraErr)
				}
			},
		},
		{
			name:           "domain failure - validation error",
			config:         &guildtypes.GuildConfig{GuildID: "valid"}, // Missing other required fields
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if !res.IsFailure() {
					t.Fatal("expected domain validation failure, but got success")
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
				repo:   fakeRepo,
				logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
			}

			res, err := s.CreateGuildConfig(context.Background(), tt.config)

			if tt.expectInfraErr && err == nil {
				t.Fatal("expected infra error but got nil")
			}
			if !tt.expectInfraErr && err != nil {
				t.Fatalf("unexpected infra error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, res, err, fakeRepo)
			}
		})
	}
}
