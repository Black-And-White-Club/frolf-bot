package clubhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleClubInfoRequest(t *testing.T) {
	validUUID := uuid.New()
	validClubInfo := &clubtypes.ClubInfo{
		UUID:    validUUID.String(),
		Name:    "Test Club",
		IconURL: nil,
	}

	tests := []struct {
		name         string
		setupService func(*FakeClubService)
		payload      *clubevents.ClubInfoRequestPayloadV1
		wantResults  int
		wantErr      bool
	}{
		{
			name: "happy path - club found",
			setupService: func(f *FakeClubService) {
				f.GetClubFunc = func(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error) {
					return validClubInfo, nil
				}
			},
			payload: &clubevents.ClubInfoRequestPayloadV1{
				ClubUUID: validUUID.String(),
			},
			wantResults: 1,
			wantErr:     false,
		},
		{
			name:         "invalid UUID",
			setupService: func(f *FakeClubService) {},
			payload: &clubevents.ClubInfoRequestPayloadV1{
				ClubUUID: "not-a-valid-uuid",
			},
			wantResults: 0,
			wantErr:     false,
		},
		{
			name: "club not found",
			setupService: func(f *FakeClubService) {
				f.GetClubFunc = func(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error) {
					return nil, clubdb.ErrNotFound
				}
			},
			payload: &clubevents.ClubInfoRequestPayloadV1{
				ClubUUID: validUUID.String(),
			},
			wantResults: 1,
			wantErr:     false,
		},
		{
			name: "service error",
			setupService: func(f *FakeClubService) {
				f.GetClubFunc = func(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error) {
					return nil, errors.New("database error")
				}
			},
			payload: &clubevents.ClubInfoRequestPayloadV1{
				ClubUUID: validUUID.String(),
			},
			wantResults: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeClubService()
			tt.setupService(fakeService)

			handler := NewClubHandlers(
				fakeService,
				slog.Default(),
				noop.NewTracerProvider().Tracer("test"),
			)

			results, err := handler.HandleClubInfoRequest(context.Background(), tt.payload)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, results, tt.wantResults)

			if tt.wantResults > 0 {
				assert.Equal(t, clubevents.ClubInfoResponseV1, results[0].Topic)

				// For "club not found" case, verify the response contains "Club Not Found"
				if tt.name == "club not found" {
					responsePayload, ok := results[0].Payload.(*clubevents.ClubInfoResponsePayloadV1)
					assert.True(t, ok, "payload should be ClubInfoResponsePayloadV1")
					assert.Equal(t, "Club Not Found", responsePayload.Name)
					assert.Equal(t, tt.payload.ClubUUID, responsePayload.UUID)
				}
			}
		})
	}
}

func TestHandleClubSyncFromDiscord(t *testing.T) {
	testGuildID := sharedtypes.GuildID("33333333333333333")
	iconURL := "https://cdn.discordapp.com/icons/123/abc.png"

	tests := []struct {
		name         string
		setupService func(*FakeClubService)
		payload      *sharedevents.ClubSyncFromDiscordRequestedPayloadV1
		wantResults  int
		wantErr      bool
		wantTrace    []string
	}{
		{
			name: "happy path - upserts club",
			setupService: func(f *FakeClubService) {
				f.UpsertClubFromDiscordFunc = func(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error) {
					return &clubtypes.ClubInfo{UUID: "some-uuid", Name: name, IconURL: iconURL}, nil
				}
			},
			payload: &sharedevents.ClubSyncFromDiscordRequestedPayloadV1{
				GuildID:   testGuildID,
				GuildName: "Test Guild",
				IconURL:   &iconURL,
			},
			wantResults: 0,
			wantErr:     false,
			wantTrace:   []string{"UpsertClubFromDiscord"},
		},
		{
			name:         "empty guild name - skipped",
			setupService: func(f *FakeClubService) {},
			payload: &sharedevents.ClubSyncFromDiscordRequestedPayloadV1{
				GuildID:   testGuildID,
				GuildName: "",
			},
			wantResults: 0,
			wantErr:     false,
			wantTrace:   []string{},
		},
		{
			name: "service error",
			setupService: func(f *FakeClubService) {
				f.UpsertClubFromDiscordFunc = func(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error) {
					return nil, errors.New("database error")
				}
			},
			payload: &sharedevents.ClubSyncFromDiscordRequestedPayloadV1{
				GuildID:   testGuildID,
				GuildName: "Test Guild",
			},
			wantResults: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeClubService()
			tt.setupService(fakeService)

			handler := NewClubHandlers(
				fakeService,
				slog.Default(),
				noop.NewTracerProvider().Tracer("test"),
			)

			results, err := handler.HandleClubSyncFromDiscord(context.Background(), tt.payload)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, results, tt.wantResults)
			if tt.wantTrace != nil {
				assert.Equal(t, tt.wantTrace, fakeService.Trace())
			}
		})
	}
}
