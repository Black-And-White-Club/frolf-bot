package userservice

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestUserService_GetClubUUIDByDiscordGuildID_CachesValue(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	expected := uuid.New()

	repoCalls := 0
	repo := NewFakeUserRepository()
	repo.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
		repoCalls++
		return expected, nil
	}

	service := NewUserService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)), &usermetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), nil)

	first, err := service.GetClubUUIDByDiscordGuildID(ctx, guildID)
	if err != nil {
		t.Fatalf("unexpected error on first lookup: %v", err)
	}
	second, err := service.GetClubUUIDByDiscordGuildID(ctx, guildID)
	if err != nil {
		t.Fatalf("unexpected error on second lookup: %v", err)
	}

	if first != expected || second != expected {
		t.Fatalf("expected cached UUID %s, got %s and %s", expected, first, second)
	}
	if repoCalls != 1 {
		t.Fatalf("expected one repository call, got %d", repoCalls)
	}
}

func TestUserService_GetClubUUIDByDiscordGuildID_RefetchesAfterExpiry(t *testing.T) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-2")
	expected := uuid.New()

	repoCalls := 0
	repo := NewFakeUserRepository()
	repo.GetClubUUIDByDiscordGuildIDFn = func(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error) {
		repoCalls++
		return expected, nil
	}

	service := NewUserService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)), &usermetrics.NoOpMetrics{}, noop.NewTracerProvider().Tracer("test"), nil)
	service.clubUUIDCacheTTL = time.Minute

	if _, err := service.GetClubUUIDByDiscordGuildID(ctx, guildID); err != nil {
		t.Fatalf("unexpected error on first lookup: %v", err)
	}

	service.clubUUIDCacheMu.Lock()
	entry := service.clubUUIDCache[guildID]
	entry.expiresAt = time.Now().UTC().Add(-time.Second)
	service.clubUUIDCache[guildID] = entry
	service.clubUUIDCacheMu.Unlock()

	if _, err := service.GetClubUUIDByDiscordGuildID(ctx, guildID); err != nil {
		t.Fatalf("unexpected error on second lookup: %v", err)
	}

	if repoCalls != 2 {
		t.Fatalf("expected two repository calls after cache expiry, got %d", repoCalls)
	}
}
