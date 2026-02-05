package clubhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
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
