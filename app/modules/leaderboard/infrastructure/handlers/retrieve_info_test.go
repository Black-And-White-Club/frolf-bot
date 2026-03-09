package leaderboardhandlers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
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
				f.GetLeaderboardFunc = func(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
					return results.SuccessResult[[]leaderboardtypes.LeaderboardEntry, error]([]leaderboardtypes.LeaderboardEntry{
						{UserID: "user-1", TagNumber: 1},
					}), nil
				}
			},
			wantErr:       false,
			wantResultLen: 1,
			wantTopic:     leaderboardevents.GetLeaderboardResponseV1,
		},
		{
			name: "Service error returns Failed event",
			setupFake: func(f *FakeService) {
				f.GetLeaderboardFunc = func(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
					return results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error]{}, fmt.Errorf("database error")
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
			h := &LeaderboardHandlers{service: fakeSvc, userService: NewFakeUserService(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

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

func TestLeaderboardHandlers_HandleGetLeaderboardRequest_ProfileLookupUsesTagFallbackIDs(t *testing.T) {
	testGuildID := sharedtypes.GuildID("test-guild-123")
	testPayload := &leaderboardevents.GetLeaderboardRequestedPayloadV1{
		GuildID: testGuildID,
	}

	fakeSvc := NewFakeService()
	fakeSvc.GetLeaderboardFunc = func(ctx context.Context, guildID sharedtypes.GuildID, seasonID string) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
		return results.SuccessResult[[]leaderboardtypes.LeaderboardEntry, error]([]leaderboardtypes.LeaderboardEntry{
			{UserID: "Tag 23 Placeholder", TagNumber: 23},
			{UserID: "legacy-unmapped-id", TagNumber: 24},
			{UserID: "839877196898238526", TagNumber: 1},
		}), nil
	}

	fakeUsers := NewFakeUserService()
	var gotUserIDs []sharedtypes.DiscordID
	fakeUsers.LookupProfilesFunc = func(ctx context.Context, userIDs []sharedtypes.DiscordID, guildID sharedtypes.GuildID) (results.OperationResult[*userservice.LookupProfilesResponse, error], error) {
		gotUserIDs = append([]sharedtypes.DiscordID(nil), userIDs...)
		return results.SuccessResult[*userservice.LookupProfilesResponse, error](&userservice.LookupProfilesResponse{
			Profiles: make(map[sharedtypes.DiscordID]*usertypes.UserProfile),
		}), nil
	}

	h := &LeaderboardHandlers{
		service:     fakeSvc,
		userService: fakeUsers,
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	if _, err := h.HandleGetLeaderboardRequest(context.Background(), testPayload); err != nil {
		t.Fatalf("HandleGetLeaderboardRequest() error = %v", err)
	}

	wantUserIDs := []sharedtypes.DiscordID{"23", "24", "839877196898238526", "1"}
	for _, want := range wantUserIDs {
		if !slices.Contains(gotUserIDs, want) {
			t.Fatalf("expected lookup IDs to contain %q, got %v", want, gotUserIDs)
		}
	}
	if slices.Contains(gotUserIDs, sharedtypes.DiscordID("Tag 23 Placeholder")) {
		t.Fatalf("expected placeholder label to be excluded from lookup IDs, got %v", gotUserIDs)
	}
	if slices.Contains(gotUserIDs, sharedtypes.DiscordID("legacy-unmapped-id")) {
		t.Fatalf("expected legacy unmapped ID to be excluded from lookup IDs, got %v", gotUserIDs)
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
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.SuccessResult[sharedtypes.TagNumber, error](5), nil
				}
			},
			wantTopic: sharedevents.LeaderboardTagLookupSucceededV1,
			wantFound: true,
		},
		{
			name: "Tag not found",
			setupFake: func(f *FakeService) {
				f.GetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.OperationResult[sharedtypes.TagNumber, error]{}, fmt.Errorf("not found")
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
			h := &LeaderboardHandlers{service: fakeSvc, userService: NewFakeUserService(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

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
				f.RoundGetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.SuccessResult[sharedtypes.TagNumber, error](10), nil
				}
			},
			wantErr:   false,
			wantTopic: sharedevents.RoundTagLookupFoundV1,
		},
		{
			name: "Round tag not found (sql.ErrNoRows)",
			setupFake: func(f *FakeService) {
				f.RoundGetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					err := sql.ErrNoRows
					return results.FailureResult[sharedtypes.TagNumber, error](err), nil
				}
			},
			wantErr:   false,
			wantTopic: sharedevents.RoundTagLookupNotFoundV1,
		},
		{
			name: "Service error",
			setupFake: func(f *FakeService) {
				f.RoundGetTagByUserIDFunc = func(ctx context.Context, g sharedtypes.GuildID, u sharedtypes.DiscordID) (results.OperationResult[sharedtypes.TagNumber, error], error) {
					return results.OperationResult[sharedtypes.TagNumber, error]{}, fmt.Errorf("internal error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSvc := NewFakeService()
			tt.setupFake(fakeSvc)
			h := &LeaderboardHandlers{service: fakeSvc, userService: NewFakeUserService(), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

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
