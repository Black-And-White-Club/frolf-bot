package leaderboardservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_UpdateLeaderboard(t *testing.T) {
	type fields struct {
		leaderboardDB *leaderboarddb.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}
	type args struct {
		ctx   context.Context
		event events.LeaderboardUpdateEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Successful Update",
			fields: fields{
				leaderboardDB: func() *leaderboarddb.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
					mockLeaderboardDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any()).Return(nil)
					return mockLeaderboardDB
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
				event: events.LeaderboardUpdateEvent{
					Scores: []events.Score{
						{DiscordID: "123", Score: -2, TagNumber: "1"},
						{DiscordID: "456", Score: 1, TagNumber: "2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Error Sorting Scores",
			fields: fields{
				leaderboardDB: leaderboarddb.NewMockLeaderboardDB(gomock.NewController(t)),
				eventBus:      eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
				event: events.LeaderboardUpdateEvent{
					Scores: []events.Score{
						{DiscordID: "123", Score: -2, TagNumber: "invalid"},
					},
				},
			},
			wantErr: true, // Expecting an error because of the invalid TagNumber
		},
		{
			name: "Error Updating Leaderboard",
			fields: fields{
				leaderboardDB: func() *leaderboarddb.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
					mockLeaderboardDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
					return mockLeaderboardDB
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
				event: events.LeaderboardUpdateEvent{
					Scores: []events.Score{
						{DiscordID: "123", Score: -2, TagNumber: "1"},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				LeaderboardDB: tt.fields.leaderboardDB,
				EventBus:      tt.fields.eventBus,
				logger:        tt.fields.logger,
			}
			if err := s.UpdateLeaderboard(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.UpdateLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	type fields struct {
		leaderboardDB *leaderboarddb.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []events.LeaderboardEntry
		wantErr bool
	}{
		{
			name: "Successful Get",
			fields: fields{
				leaderboardDB: func() *leaderboarddb.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboarddb.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().GetLeaderboard(gomock.Any()).Return(&leaderboardtypes.Leaderboard{
						LeaderboardData: map[int]string{
							1: "123",
							2: "456",
						},
						// ... other fields if needed
					}, nil)
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
			},
			want: []events.LeaderboardEntry{
				{TagNumber: "1", DiscordID: "123"},
				{TagNumber: "2", DiscordID: "456"},
			},
			wantErr: false,
		},
		{
			name: "Error Getting Leaderboard",
			fields: fields{
				leaderboardDB: func() *leaderboarddb.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboarddb.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().GetLeaderboard(gomock.Any()).Return(nil, errors.New("database error"))
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				LeaderboardDB: tt.fields.leaderboardDB,
				EventBus:      tt.fields.eventBus,
				logger:        tt.fields.logger,
			}
			got, err := s.GetLeaderboard(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.GetLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LeaderboardService.GetLeaderboard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardService_GetTagByDiscordID(t *testing.T) {
	type fields struct {
		leaderboardDB *leaderboarddb.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}
	type args struct {
		ctx       context.Context
		discordID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "Successful Get",
			fields: fields{
				leaderboardDB: func() *leaderboarddb.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboarddb.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().GetTagByDiscordID(gomock.Any(), "123").Return(42, nil)
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:       context.Background(),
				discordID: "123",
			},
			want:    42,
			wantErr: false,
		},
		{
			name: "Error Getting Tag",
			fields: fields{
				leaderboardDB: func() *leaderboarddb.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboarddb.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().GetTagByDiscordID(gomock.Any(), "456").Return(0, errors.New("database error"))
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:       context.Background(),
				discordID: "456",
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				LeaderboardDB: tt.fields.leaderboardDB,
				EventBus:      tt.fields.eventBus,
				logger:        tt.fields.logger,
			}
			got, err := s.GetTagByDiscordID(tt.args.ctx, tt.args.discordID)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.GetTagByDiscordID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LeaderboardService.GetTagByDiscordID() = %v, want %v", got, tt.want)
			}
		})
	}
}
