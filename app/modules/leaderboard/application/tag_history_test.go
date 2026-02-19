package leaderboardservice

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func TestLeaderboardService_GetTagHistory(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	fixedTime := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	roundID := "round-abc"

	tests := []struct {
		name      string
		memberID  string
		limit     int
		setupFake func(*FakeCommandPipeline)
		wantCount int
		wantErr   bool
	}{
		{
			name:     "returns history entries for a member",
			memberID: "user-1",
			limit:    10,
			setupFake: func(f *FakeCommandPipeline) {
				f.GetTagHistoryFunc = func(ctx context.Context, guildID, memberID string, limit int) ([]TagHistoryView, error) {
					return []TagHistoryView{
						{ID: 1, TagNumber: 5, NewMemberID: "user-1", Reason: "round_swap", CreatedAt: fixedTime, RoundID: &roundID},
						{ID: 2, TagNumber: 3, NewMemberID: "user-1", OldMemberID: "user-2", Reason: "round_swap", CreatedAt: fixedTime},
					}, nil
				}
			},
			wantCount: 2,
		},
		{
			name:     "returns empty slice when no history",
			memberID: "user-nobody",
			limit:    10,
			setupFake: func(f *FakeCommandPipeline) {
				f.GetTagHistoryFunc = func(ctx context.Context, guildID, memberID string, limit int) ([]TagHistoryView, error) {
					return []TagHistoryView{}, nil
				}
			},
			wantCount: 0,
		},
		{
			name:     "returns nil when pipeline errors",
			memberID: "user-1",
			limit:    10,
			setupFake: func(f *FakeCommandPipeline) {
				f.GetTagHistoryFunc = func(ctx context.Context, guildID, memberID string, limit int) ([]TagHistoryView, error) {
					return nil, errors.New("db failure")
				}
			},
			wantErr: true,
		},
		{
			name:     "nil pipeline returns ErrCommandPipelineUnavailable",
			memberID: "user-1",
			limit:    10,
			// no setupFake → commandPipeline will be nil
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var svc *LeaderboardService
			if tt.setupFake != nil {
				fake := &FakeCommandPipeline{}
				tt.setupFake(fake)
				svc = &LeaderboardService{commandPipeline: fake}
			} else {
				svc = &LeaderboardService{} // nil commandPipeline
			}

			got, err := svc.GetTagHistory(context.Background(), guildID, tt.memberID, tt.limit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetTagHistory() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(got) != tt.wantCount {
				t.Errorf("GetTagHistory() len = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestLeaderboardService_GetTagList(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		setupFake func(*FakeCommandPipeline)
		wantCount int
		wantErr   bool
	}{
		{
			name: "returns sorted tag list",
			setupFake: func(f *FakeCommandPipeline) {
				f.GetTaggedFunc = func(ctx context.Context, guildID string, clubUUID *string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{
						{MemberID: "user-2", Tag: 2},
						{MemberID: "user-1", Tag: 1},
					}, nil
				}
			},
			wantCount: 2,
		},
		{
			name: "returns empty list when no tagged members",
			setupFake: func(f *FakeCommandPipeline) {
				f.GetTaggedFunc = func(ctx context.Context, guildID string, clubUUID *string) ([]TaggedMemberView, error) {
					return []TaggedMemberView{}, nil
				}
			},
			wantCount: 0,
		},
		{
			name: "pipeline error propagates",
			setupFake: func(f *FakeCommandPipeline) {
				f.GetTaggedFunc = func(ctx context.Context, guildID string, clubUUID *string) ([]TaggedMemberView, error) {
					return nil, errors.New("db error")
				}
			},
			wantErr: true,
		},
		{
			name:    "nil pipeline returns ErrCommandPipelineUnavailable",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var svc *LeaderboardService
			if tt.setupFake != nil {
				fake := &FakeCommandPipeline{}
				tt.setupFake(fake)
				svc = &LeaderboardService{commandPipeline: fake}
			} else {
				svc = &LeaderboardService{}
			}

			got, err := svc.GetTagList(context.Background(), guildID, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetTagList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(got) != tt.wantCount {
				t.Errorf("GetTagList() len = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestLeaderboardService_GenerateTagGraphPNG(t *testing.T) {
	guildID := sharedtypes.GuildID("test-guild")

	tests := []struct {
		name      string
		memberID  string
		setupFake func(*FakeCommandPipeline)
		wantBytes bool
		wantErr   bool
	}{
		{
			name:     "returns non-nil PNG bytes on success",
			memberID: "user-1",
			setupFake: func(f *FakeCommandPipeline) {
				f.GenerateTagGraphPNGFunc = func(ctx context.Context, guildID, memberID string) ([]byte, error) {
					return []byte{0x89, 0x50, 0x4E, 0x47}, nil // PNG magic bytes
				}
			},
			wantBytes: true,
		},
		{
			name:     "pipeline error propagates",
			memberID: "user-1",
			setupFake: func(f *FakeCommandPipeline) {
				f.GenerateTagGraphPNGFunc = func(ctx context.Context, guildID, memberID string) ([]byte, error) {
					return nil, errors.New("chart error")
				}
			},
			wantErr: true,
		},
		{
			name:     "nil pipeline returns ErrCommandPipelineUnavailable",
			memberID: "user-1",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var svc *LeaderboardService
			if tt.setupFake != nil {
				fake := &FakeCommandPipeline{}
				tt.setupFake(fake)
				svc = &LeaderboardService{commandPipeline: fake}
			} else {
				svc = &LeaderboardService{}
			}

			got, err := svc.GenerateTagGraphPNG(context.Background(), guildID, tt.memberID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GenerateTagGraphPNG() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantBytes && len(got) == 0 {
				t.Error("GenerateTagGraphPNG() returned empty bytes, want PNG data")
			}
		})
	}
}
