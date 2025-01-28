package leaderboardservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain/types"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_GetLeaderboardRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testCorrelationID := watermill.NewUUID()

	type fields struct {
		LeaderboardDB *leaderboarddb.MockLeaderboardDB
		EventBus      *eventbusmocks.MockEventBus
		logger        *slog.Logger
		eventUtil     eventutil.EventUtil
	}
	type args struct {
		ctx context.Context
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Get Leaderboard Request",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.GetLeaderboardRequestPayload{}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(gomock.Any()).
					Return(&leaderboarddbtypes.Leaderboard{
						LeaderboardData: map[int]string{1: "a", 2: "b"},
					}, nil).
					Times(1)
				f.EventBus.EXPECT().
					Publish(leaderboardevents.GetLeaderboardResponse, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.GetLeaderboardResponse {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.GetLeaderboardResponse, topic)
						}
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Get Active Leaderboard Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.GetLeaderboardRequestPayload{}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
			},
		},
		{
			name: "Publish Response Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.GetLeaderboardRequestPayload{}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(gomock.Any()).
					Return(&leaderboarddbtypes.Leaderboard{
						LeaderboardData: map[int]string{1: "a", 2: "b"},
					}, nil).
					Times(1)
				f.EventBus.EXPECT().
					Publish(leaderboardevents.GetLeaderboardResponse, gomock.Any()).
					Return(errors.New("publish error")).
					Times(1)
			},
		},
		{
			name: "Unmarshal Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: message.NewMessage(testCorrelationID, []byte("invalid-payload")),
			},
			wantErr: true,
			setup:   func(f fields, a args) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				LeaderboardDB: tt.fields.LeaderboardDB,
				EventBus:      tt.fields.EventBus,
				logger:        tt.fields.logger,
				eventUtil:     tt.fields.eventUtil,
			}
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}
			if err := s.GetLeaderboardRequest(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_GetTagByDiscordIDRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testDiscordID := "testDiscordID"
	testTagNumber := 123
	testCorrelationID := watermill.NewUUID()

	type fields struct {
		LeaderboardDB *leaderboarddb.MockLeaderboardDB
		EventBus      *eventbusmocks.MockEventBus
		logger        *slog.Logger
		eventUtil     eventutil.EventUtil
	}
	type args struct {
		ctx context.Context
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Get Tag By Discord ID Request",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.GetTagByDiscordIDRequestPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					GetTagByDiscordID(gomock.Any(), testDiscordID).
					Return(testTagNumber, nil).
					Times(1)
				f.EventBus.EXPECT().
					Publish(leaderboardevents.GetTagByDiscordIDResponse, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.GetTagByDiscordIDResponse {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.GetTagByDiscordIDResponse, topic)
						}
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Get Tag By Discord ID Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.GetTagByDiscordIDRequestPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					GetTagByDiscordID(gomock.Any(), testDiscordID).
					Return(0, errors.New("database error")).
					Times(1)
			},
		},
		{
			name: "Publish Response Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.GetTagByDiscordIDRequestPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					GetTagByDiscordID(gomock.Any(), testDiscordID).
					Return(testTagNumber, nil).
					Times(1)
				f.EventBus.EXPECT().
					Publish(leaderboardevents.GetTagByDiscordIDResponse, gomock.Any()).
					Return(errors.New("publish error")).
					Times(1)
			},
		},
		{
			name: "Unmarshal Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: message.NewMessage(testCorrelationID, []byte("invalid-payload")),
			},
			wantErr: true,
			setup:   func(f fields, a args) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LeaderboardService{
				LeaderboardDB: tt.fields.LeaderboardDB,
				EventBus:      tt.fields.EventBus,
				logger:        tt.fields.logger,
				eventUtil:     tt.fields.eventUtil,
			}
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}
			if err := s.GetTagByDiscordIDRequest(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.GetTagByDiscordIDRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
