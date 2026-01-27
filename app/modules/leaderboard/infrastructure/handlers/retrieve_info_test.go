package leaderboardhandlers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestLeaderboardHandlers_HandleGetLeaderboardRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testPayload := &leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: testGuildID,
	}

	tests := []struct {
		name          string
		setupFake     func(f *FakeService)
		wantErr       bool
		wantResultLen int
		wantTopic     string
	}{
		{
			name: "Successfully get leaderboard",
			setupFake: func(f *FakeService) {
				f.GetLeaderboardFunc = func(ctx context.Context, guildID sharedtypes.GuildID) ([]leaderboardtypes.LeaderboardEntry, error) {
					return []leaderboardtypes.LeaderboardEntry{
						{UserID: "user-1", TagNumber: 1},
					}, nil
				}
			},
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.GetLeaderboardResponseV1,
		},
		{
			name: "Service error returns Failed event",
			setupFake: func(f *FakeService) {
				f.GetLeaderboardFunc = func(ctx context.Context, guildID sharedtypes.GuildID) ([]leaderboardtypes.LeaderboardEntry, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			wantErr:       false, // Handler catches error and returns failure event
			wantResultLen: 1,
			wantTopic:     leaderboardevents.GetLeaderboardFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			tt.setupFake(fakeSvc)
			h := &LeaderboardHandlers{service: fakeSvc}

			res, err := h.HandleGetLeaderboardRequest(context.Background(), testPayload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(res) != tt.wantResultLen {
				t.Errorf("Result length = %d, want %d", len(res), tt.wantResultLen)
			}
			if len(res) > 0 && res[0].Topic != tt.wantTopic {
				t.Errorf("Topic = %s, want %s", res[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByUserIDRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testUserID := sharedtypes.DiscordID("user-456")
	testPayload := &sharedevents.DiscordTagLookupRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		UserID:        testUserID,
	}

	tests := []struct {
		name      string
		setupFake func(f *FakeService)
		wantTopic string
		wantFound bool
	}{
		{
			name: "Tag found",
			setupFake: func(f *FakeService) {
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
					return 5, nil
				}
			},
			wantTopic: sharedevents.LeaderboardTagLookupSucceededV1,
			wantFound: true,
		},
		{
			name: "Tag not found",
			setupFake: func(f *FakeService) {
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
					return 0, fmt.Errorf("not found")
				}
			},
			wantTopic: sharedevents.LeaderboardTagLookupNotFoundV1,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			tt.setupFake(fakeSvc)
			h := &LeaderboardHandlers{service: fakeSvc}

			res, _ := h.HandleGetTagByUserIDRequest(context.Background(), testPayload)

			if res[0].Topic != tt.wantTopic {
				t.Errorf("Topic = %s, want %s", res[0].Topic, tt.wantTopic)
			}
			p := res[0].Payload.(*sharedevents.DiscordTagLookupResultPayloadV1)
			if p.Found != tt.wantFound {
				t.Errorf("Found = %v, want %v", p.Found, tt.wantFound)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleRoundGetTagRequest(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testUserID := sharedtypes.DiscordID("user-456")
	testPayload := &sharedevents.RoundTagLookupRequestedPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: testGuildID},
		RoundID:       sharedtypes.RoundID(uuid.New()),
		UserID:        testUserID,
	}

	tests := []struct {
		name      string
		setupFake func(f *FakeService)
		wantErr   bool
		wantTopic string
	}{
		{
			name: "Round tag found",
			setupFake: func(f *FakeService) {
				f.RoundGetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
					return 10, nil
				}
			},
			wantErr:   false,
			wantTopic: sharedevents.RoundTagLookupFoundV1,
		},
		{
			name: "Round tag not found (sql.ErrNoRows)",
			setupFake: func(f *FakeService) {
				f.RoundGetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
					return 0, sql.ErrNoRows
				}
			},
			wantErr:   false,
			wantTopic: sharedevents.RoundTagLookupNotFoundV1,
		},
		{
			name: "Service error",
			setupFake: func(f *FakeService) {
				f.RoundGetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (sharedtypes.TagNumber, error) {
					return 0, fmt.Errorf("internal error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			tt.setupFake(fakeSvc)
			h := &LeaderboardHandlers{service: fakeSvc}

			res, err := h.HandleRoundGetTagRequest(context.Background(), testPayload)

			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr %v, got %v", tt.wantErr, err)
			}
			if !tt.wantErr && res[0].Topic != tt.wantTopic {
				t.Errorf("Topic = %s, want %s", res[0].Topic, tt.wantTopic)
			}
		})
	}
}
