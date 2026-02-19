package leaderboardhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

func TestLeaderboardHandlers_HandleTagHistoryRequest(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")
	roundID := "round-abc"
	fixedTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		payload   *leaderboardevents.TagHistoryRequestedPayloadV1
		setupFake func(*FakeService)
		wantTopic string
		wantErr   bool
	}{
		{
			name:    "success with history entries",
			payload: &leaderboardevents.TagHistoryRequestedPayloadV1{GuildID: string(guildID), MemberID: "user-1", Limit: 10},
			setupFake: func(f *FakeService) {
				f.GetTagHistoryFunc = func(ctx context.Context, g sharedtypes.GuildID, memberID string, limit int) ([]leaderboardservice.TagHistoryView, error) {
					return []leaderboardservice.TagHistoryView{
						{ID: 1, TagNumber: 5, NewMemberID: "user-1", Reason: "round_swap", CreatedAt: fixedTime, RoundID: &roundID},
						{ID: 2, TagNumber: 3, NewMemberID: "user-1", OldMemberID: "user-2", Reason: "round_swap", CreatedAt: fixedTime},
					}, nil
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagHistoryResponseV1,
		},
		{
			name:    "success with no history (empty slice)",
			payload: &leaderboardevents.TagHistoryRequestedPayloadV1{GuildID: string(guildID), MemberID: "new-user", Limit: 10},
			setupFake: func(f *FakeService) {
				f.GetTagHistoryFunc = func(ctx context.Context, g sharedtypes.GuildID, memberID string, limit int) ([]leaderboardservice.TagHistoryView, error) {
					return []leaderboardservice.TagHistoryView{}, nil
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagHistoryResponseV1,
		},
		{
			name:    "service error returns failure topic",
			payload: &leaderboardevents.TagHistoryRequestedPayloadV1{GuildID: string(guildID), MemberID: "user-1"},
			setupFake: func(f *FakeService) {
				f.GetTagHistoryFunc = func(ctx context.Context, g sharedtypes.GuildID, memberID string, limit int) ([]leaderboardservice.TagHistoryView, error) {
					return nil, errors.New("db error")
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagHistoryFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{service: fakeSvc, logger: slog.Default()}
			got, err := h.HandleTagHistoryRequest(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) == 0 {
				t.Fatal("expected at least one result")
			}
			if got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagGraphRequest(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		payload   *leaderboardevents.TagGraphRequestedPayloadV1
		setupFake func(*FakeService)
		wantTopic string
		wantErr   bool
	}{
		{
			name:    "success returns PNG bytes",
			payload: &leaderboardevents.TagGraphRequestedPayloadV1{GuildID: string(guildID), MemberID: "user-1"},
			setupFake: func(f *FakeService) {
				f.GenerateTagGraphPNGFunc = func(ctx context.Context, g sharedtypes.GuildID, memberID string) ([]byte, error) {
					// Return minimal valid PNG header bytes
					return []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, nil
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagGraphResponseV1,
		},
		{
			name:    "empty PNG data still returns response topic",
			payload: &leaderboardevents.TagGraphRequestedPayloadV1{GuildID: string(guildID), MemberID: "user-no-history"},
			setupFake: func(f *FakeService) {
				f.GenerateTagGraphPNGFunc = func(ctx context.Context, g sharedtypes.GuildID, memberID string) ([]byte, error) {
					return []byte{}, nil // no history → placeholder rendered
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagGraphResponseV1,
		},
		{
			name:    "service error returns failure topic",
			payload: &leaderboardevents.TagGraphRequestedPayloadV1{GuildID: string(guildID), MemberID: "user-1"},
			setupFake: func(f *FakeService) {
				f.GenerateTagGraphPNGFunc = func(ctx context.Context, g sharedtypes.GuildID, memberID string) ([]byte, error) {
					return nil, errors.New("chart generation failed")
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagGraphFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{service: fakeSvc, logger: slog.Default()}
			got, err := h.HandleTagGraphRequest(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) == 0 {
				t.Fatal("expected at least one result")
			}
			if got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagListRequest(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		payload   *leaderboardevents.TagListRequestedPayloadV1
		setupFake func(*FakeService)
		wantTopic string
		wantErr   bool
	}{
		{
			name:    "success with members",
			payload: &leaderboardevents.TagListRequestedPayloadV1{GuildID: string(guildID)},
			setupFake: func(f *FakeService) {
				f.GetTagListFunc = func(ctx context.Context, g sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.TaggedMemberView, error) {
					return []leaderboardservice.TaggedMemberView{
						{MemberID: "user-1", Tag: 1},
						{MemberID: "user-2", Tag: 2},
					}, nil
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagListResponseV1,
		},
		{
			name:    "success with empty list",
			payload: &leaderboardevents.TagListRequestedPayloadV1{GuildID: string(guildID)},
			setupFake: func(f *FakeService) {
				f.GetTagListFunc = func(ctx context.Context, g sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.TaggedMemberView, error) {
					return []leaderboardservice.TaggedMemberView{}, nil
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagListResponseV1,
		},
		{
			name:    "service error returns failure topic",
			payload: &leaderboardevents.TagListRequestedPayloadV1{GuildID: string(guildID)},
			setupFake: func(f *FakeService) {
				f.GetTagListFunc = func(ctx context.Context, g sharedtypes.GuildID, clubUUID *string) ([]leaderboardservice.TaggedMemberView, error) {
					return nil, errors.New("db error")
				}
			},
			wantTopic: leaderboardevents.LeaderboardTagListFailedV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			if tt.setupFake != nil {
				tt.setupFake(fakeSvc)
			}

			h := &LeaderboardHandlers{service: fakeSvc, logger: slog.Default()}
			got, err := h.HandleTagListRequest(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) == 0 {
				t.Fatal("expected at least one result")
			}
			if got[0].Topic != tt.wantTopic {
				t.Errorf("topic = %s, want %s", got[0].Topic, tt.wantTopic)
			}
		})
	}
}
