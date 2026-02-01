package guildservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

func TestGuildService_DeleteGuildConfig(t *testing.T) {
	ctx := context.Background()

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
				f.DeleteConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) error {
					return nil
				}
				f.GetConfigIncludeDeletedFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return &guildtypes.GuildConfig{GuildID: id}, nil
				}
			},
			expectInfraErr: false,
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
				// Verify call sequence: Delete then GetIncludeDeleted
				trace := fake.Trace()
				if len(trace) < 2 || trace[0] != "DeleteConfig" || trace[1] != "GetConfigIncludeDeleted" {
					t.Errorf("unexpected call sequence: %v", trace)
				}
			},
		},
		{
			name:    "infrastructure error - db failure on DeleteConfig",
			guildID: "guild-4",
			setupFake: func(f *FakeGuildRepository) {
				f.DeleteConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) error {
					return errors.New("delete error")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "delete error") {
					t.Errorf("expected infrastructure error containing 'delete error', got: %v", infraErr)
				}
			},
		},
		{
			name:           "domain failure - invalid guildID",
			guildID:        "",
			expectInfraErr: false,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr != nil {
					t.Errorf("expected nil infra error for invalid ID, got: %v", infraErr)
				}
				if res.Failure == nil || !errors.Is(*res.Failure, ErrInvalidGuildID) {
					t.Errorf("expected domain failure ErrInvalidGuildID, got %v", res.Failure)
				}
			},
		},
		{
			name:    "infrastructure error - failure on final state fetch",
			guildID: "guild-5",
			setupFake: func(f *FakeGuildRepository) {
				f.DeleteConfigFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) error {
					return nil
				}
				f.GetConfigIncludeDeletedFunc = func(ctx context.Context, db bun.IDB, id sharedtypes.GuildID) (*guildtypes.GuildConfig, error) {
					return nil, errors.New("fetch error")
				}
			},
			expectInfraErr: true,
			verify: func(t *testing.T, res GuildConfigResult, infraErr error, fake *FakeGuildRepository) {
				if infraErr == nil || !strings.Contains(infraErr.Error(), "fetch error") {
					t.Errorf("expected infrastructure error containing 'fetch error', got: %v", infraErr)
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

			res, err := s.DeleteGuildConfig(ctx, tt.guildID)

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
