package leaderboardservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmock "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	events "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardmocks "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_AssignTag(t *testing.T) {
	type fields struct {
		leaderboardDB *leaderboardmocks.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}
	type args struct {
		ctx   context.Context
		event events.TagAssignedEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Successful Assign",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().AssignTag(gomock.Any(), "123", int(123)).Return(nil)
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
				event: events.TagAssignedEvent{
					DiscordID: "123",
					TagNumber: 123,
				},
			},
			wantErr: false,
		},
		{
			name: "Error Assigning Tag",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().AssignTag(gomock.Any(), "123", int(123)).Return(errors.New("database error"))
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx: context.Background(),
				event: events.TagAssignedEvent{
					DiscordID: "123",
					TagNumber: 123,
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
			if err := s.AssignTag(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.AssignTag() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_SwapTags(t *testing.T) {
	type fields struct {
		leaderboardDB *leaderboardmocks.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}
	type args struct {
		ctx         context.Context
		requestorID string
		targetID    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Successful Swap",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().SwapTags(gomock.Any(), "123", "456").Return(nil)
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:         context.Background(),
				requestorID: "123",
				targetID:    "456",
			},
			wantErr: false,
		},
		{
			name: "Error Swapping Tags",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().SwapTags(gomock.Any(), "123", "456").Return(errors.New("database error"))
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:         context.Background(),
				requestorID: "123",
				targetID:    "456",
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
			if err := s.SwapTags(tt.args.ctx, tt.args.requestorID, tt.args.targetID); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.SwapTags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_CheckTagAvailability(t *testing.T) {
	type fields struct {
		leaderboardDB *leaderboardmocks.MockLeaderboardDB
		eventBus      *eventbusmock.MockEventBus
		logger        *slog.Logger
	}
	type args struct {
		ctx       context.Context
		tagNumber int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Tag Available",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().CheckTagAvailability(gomock.Any(), 123).Return(true, nil)
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:       context.Background(),
				tagNumber: 123,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "Tag Not Available",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().CheckTagAvailability(gomock.Any(), 456).Return(false, nil)
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:       context.Background(),
				tagNumber: 456,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "Error Checking Availability",
			fields: fields{
				leaderboardDB: func() *leaderboardmocks.MockLeaderboardDB {
					ctrl := gomock.NewController(t)
					mock := leaderboardmocks.NewMockLeaderboardDB(ctrl)
					mock.EXPECT().CheckTagAvailability(gomock.Any(), 789).Return(false, errors.New("database error"))
					return mock
				}(),
				eventBus: eventbusmock.NewMockEventBus(gomock.NewController(t)),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			},
			args: args{
				ctx:       context.Background(),
				tagNumber: 789,
			},
			want:    false,
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
			got, err := s.CheckTagAvailability(tt.args.ctx, tt.args.tagNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.CheckTagAvailability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LeaderboardService.CheckTagAvailability() = %v, want %v", got, tt.want)
			}
		})
	}
}
