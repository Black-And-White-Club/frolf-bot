package leaderboardservice

import (
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"go.uber.org/mock/gomock"
)

func TestNewLeaderboardService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type args struct {
		db       *leaderboarddb.MockLeaderboardDB
		eventBus *eventbusmocks.MockEventBus
		logger   *slog.Logger
	}
	tests := []struct {
		name string
		args args
		want *LeaderboardService
	}{
		{
			name: "Successful Creation",
			args: args{
				db:       mockLeaderboardDB,
				eventBus: mockEventBus,
				logger:   logger,
			},
			want: &LeaderboardService{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventutil.NewEventUtil(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewLeaderboardService(tt.args.db, tt.args.eventBus, tt.args.logger)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLeaderboardService() = %v, want %v", got, tt.want)
			}
		})
	}
}
