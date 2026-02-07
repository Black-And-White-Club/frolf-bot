package roundhandlers

import (
	"context"
	"errors"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleNativeEventCreated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testDiscordEventID := "event-123"

	testPayload := &roundevents.NativeEventCreatedPayloadV1{
		GuildID:        testGuildID,
		RoundID:        testRoundID,
		DiscordEventID: testDiscordEventID,
	}

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService)
		payload        *roundevents.NativeEventCreatedPayloadV1
		wantErr        bool
		wantResultLen  int
		expectedErrMsg string
	}{
		{
			name: "Successfully handle NativeEventCreated",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateDiscordEventIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordEventID string) (*roundtypes.Round, error) {
					return &roundtypes.Round{
						ID:             testRoundID,
						DiscordEventID: testDiscordEventID,
					}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 0, // Terminal sink
		},
		{
			name: "Service error returns error",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateDiscordEventIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordEventID string) (*roundtypes.Round, error) {
					return nil, errors.New("db error")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "db error",
		},
		{
			name: "Service returns nil round",
			fakeSetup: func(fake *FakeService) {
				fake.UpdateDiscordEventIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordEventID string) (*roundtypes.Round, error) {
					return nil, nil
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "updated round object is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
			}

			ctx := context.Background()
			results, err := h.HandleNativeEventCreated(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleNativeEventCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleNativeEventCreated() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleNativeEventCreated() result length = %d, want %d", len(results), tt.wantResultLen)
			}
		})
	}
}

func TestRoundHandlers_HandleNativeEventLookupRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testGuildID := sharedtypes.GuildID("guild-123")
	testDiscordEventID := "event-123"

	testPayload := &roundevents.NativeEventLookupRequestPayloadV1{
		GuildID:        testGuildID,
		DiscordEventID: testDiscordEventID,
	}

	tests := []struct {
		name           string
		fakeSetup      func(*FakeService)
		payload        *roundevents.NativeEventLookupRequestPayloadV1
		wantErr        bool
		wantResultLen  int
		wantFound      bool
		wantRoundID    sharedtypes.RoundID
		expectedErrMsg string
	}{
		{
			name: "Successfully found round",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundByDiscordEventIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error) {
					return &roundtypes.Round{ID: testRoundID}, nil
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantFound:     true,
			wantRoundID:   testRoundID,
		},
		{
			name: "Round not found (ErrRoundNotFound) returns Success with Found=false",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundByDiscordEventIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error) {
					return nil, roundservice.ErrRoundNotFound
				}
			},
			payload:       testPayload,
			wantErr:       false,
			wantResultLen: 1,
			wantFound:     false,
		},
		{
			name: "DB Error returns Error (Retry)",
			fakeSetup: func(fake *FakeService) {
				fake.GetRoundByDiscordEventIDFunc = func(ctx context.Context, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error) {
					return nil, errors.New("db connection failed")
				}
			},
			payload:        testPayload,
			wantErr:        true,
			expectedErrMsg: "db connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeService := NewFakeService()
			if tt.fakeSetup != nil {
				tt.fakeSetup(fakeService)
			}

			h := &RoundHandlers{
				service:     fakeService,
				userService: NewFakeUserService(),
			}

			ctx := context.Background()
			results, err := h.HandleNativeEventLookupRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleNativeEventLookupRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
				t.Errorf("HandleNativeEventLookupRequest() error = %v, expected %v", err.Error(), tt.expectedErrMsg)
			}
			if len(results) != tt.wantResultLen {
				t.Errorf("HandleNativeEventLookupRequest() result length = %d, want %d", len(results), tt.wantResultLen)
			}

			if tt.wantResultLen > 0 {
				resPayload, ok := results[0].Payload.(*roundevents.NativeEventLookupResultPayloadV1)
				if !ok {
					t.Fatalf("Expected NativeEventLookupResultPayloadV1, got %T", results[0].Payload)
				}
				if resPayload.Found != tt.wantFound {
					t.Errorf("Expected Found=%v, got %v", tt.wantFound, resPayload.Found)
				}
				if tt.wantFound && resPayload.RoundID != tt.wantRoundID {
					t.Errorf("Expected RoundID=%v, got %v", tt.wantRoundID, resPayload.RoundID)
				}
			}
		})
	}
}
