package leaderboardservice

import (
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardService(t *testing.T) {
	tests := []struct {
		name          string
		leaderboardDB *leaderboarddb.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}{
		{
			name:          "Create New Leaderboard Service",
			leaderboardDB: leaderboarddb.NewMockLeaderboardDB(gomock.NewController(t)),
			eventBus:      eventbusmock.NewMockEventBus(gomock.NewController(t)),
			logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewLeaderboardService(tt.leaderboardDB, tt.eventBus, tt.logger)
			if got == nil {
				t.Error("NewLeaderboardService() returned nil")
				return // Return early if got is nil
			}
			if got.LeaderboardDB != tt.leaderboardDB {
				t.Error("LeaderboardDB not set correctly")
			}
			if got.EventBus != tt.eventBus {
				t.Error("EventBus not set correctly")
			}
			if got.logger != tt.logger {
				t.Error("Logger not set correctly")
			}
		})
	}
}

func TestLeaderboardService_sortScores(t *testing.T) {
	tests := []struct {
		name    string
		scores  []events.Score
		want    []events.Score
		wantErr bool
	}{
		{
			name: "Sort Scores",
			scores: []events.Score{
				{DiscordID: "123", Score: -2, TagNumber: "1"},
				{DiscordID: "456", Score: 1, TagNumber: "2"},
				{DiscordID: "789", Score: -2, TagNumber: "3"},
			},
			want: []events.Score{
				{DiscordID: "789", Score: -2, TagNumber: "3"},
				{DiscordID: "123", Score: -2, TagNumber: "1"},
				{DiscordID: "456", Score: 1, TagNumber: "2"},
			},
			wantErr: false,
		},
		{
			name: "Error Converting Tag Number",
			scores: []events.Score{
				{DiscordID: "123", Score: -2, TagNumber: "invalid"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
			mockEventBus := eventbusmock.NewMockEventBus(ctrl)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			s := NewLeaderboardService(mockLeaderboardDB, mockEventBus, logger)

			got, err := s.sortScores(tt.scores)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.sortScores() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LeaderboardService.sortScores() = %v, want %v", got, tt.want)
			}
		})
	}
}
