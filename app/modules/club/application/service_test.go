package clubservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
)

func TestGetClub(t *testing.T) {
	testUUID := uuid.New()
	testClub := &clubdb.Club{
		UUID:    testUUID,
		Name:    "Test Club",
		IconURL: nil,
	}

	tests := []struct {
		name        string
		setupRepo   func(*FakeClubRepo)
		clubUUID    uuid.UUID
		wantName    string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "happy path - club found",
			setupRepo: func(f *FakeClubRepo) {
				f.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return testClub, nil
				}
			},
			clubUUID: testUUID,
			wantName: "Test Club",
			wantErr:  false,
		},
		{
			name: "club not found",
			setupRepo: func(f *FakeClubRepo) {
				f.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return nil, clubdb.ErrNotFound
				}
			},
			clubUUID:    testUUID,
			wantErr:     true,
			wantErrType: clubdb.ErrNotFound,
		},
		{
			name: "database error",
			setupRepo: func(f *FakeClubRepo) {
				f.GetByUUIDFunc = func(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (*clubdb.Club, error) {
					return nil, errors.New("database connection failed")
				}
			},
			clubUUID: testUUID,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeClubRepo()
			tt.setupRepo(fakeRepo)

			svc := NewClubService(
				fakeRepo,
				slog.Default(),
				clubmetrics.NewNoop(),
				nil,
				nil,
			)

			result, err := svc.GetClub(context.Background(), tt.clubUUID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantName, result.Name)
		})
	}
}

func TestUpsertClubFromDiscord(t *testing.T) {
	existingClub := &clubdb.Club{
		UUID:           uuid.New(),
		Name:           "Old Name",
		DiscordGuildID: ptrString("123456789"),
	}

	tests := []struct {
		name      string
		setupRepo func(*FakeClubRepo)
		guildID   string
		clubName  string
		iconURL   *string
		wantName  string
		wantErr   bool
	}{
		{
			name: "create new club",
			setupRepo: func(f *FakeClubRepo) {
				f.GetByDiscordGuildIDFunc = func(ctx context.Context, db bun.IDB, guildID string) (*clubdb.Club, error) {
					return nil, clubdb.ErrNotFound
				}
				f.UpsertFunc = func(ctx context.Context, db bun.IDB, club *clubdb.Club) error {
					return nil
				}
			},
			guildID:  "123456789",
			clubName: "New Club",
			iconURL:  ptrString("https://example.com/icon.png"),
			wantName: "New Club",
			wantErr:  false,
		},
		{
			name: "update existing club",
			setupRepo: func(f *FakeClubRepo) {
				f.GetByDiscordGuildIDFunc = func(ctx context.Context, db bun.IDB, guildID string) (*clubdb.Club, error) {
					return existingClub, nil
				}
				f.UpsertFunc = func(ctx context.Context, db bun.IDB, club *clubdb.Club) error {
					return nil
				}
			},
			guildID:  "123456789",
			clubName: "Updated Name",
			iconURL:  nil,
			wantName: "Updated Name",
			wantErr:  false,
		},
		{
			name: "database error on lookup",
			setupRepo: func(f *FakeClubRepo) {
				f.GetByDiscordGuildIDFunc = func(ctx context.Context, db bun.IDB, guildID string) (*clubdb.Club, error) {
					return nil, errors.New("database connection failed")
				}
			},
			guildID:  "123456789",
			clubName: "Test Club",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRepo := NewFakeClubRepo()
			tt.setupRepo(fakeRepo)

			svc := NewClubService(
				fakeRepo,
				slog.Default(),
				clubmetrics.NewNoop(),
				nil,
				nil,
			)

			result, err := svc.UpsertClubFromDiscord(context.Background(), tt.guildID, tt.clubName, tt.iconURL)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantName, result.Name)
		})
	}
}

func ptrString(s string) *string {
	return &s
}
