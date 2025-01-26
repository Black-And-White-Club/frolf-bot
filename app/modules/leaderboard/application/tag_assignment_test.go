package leaderboardservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/types"
	leaderboarddbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagAssigned(t *testing.T) {
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
			name: "Successful Tag Assigned",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAssignedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(gomock.Any(), testTagNumber).
					Return(true, nil).
					Times(1)
				f.EventBus.EXPECT().
					Publish(leaderboardevents.TagAssignmentRequested, gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Tag Not Available",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAssignedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(gomock.Any(), testTagNumber).
					Return(false, nil).
					Times(1)
			},
		},
		{
			name: "Check Tag Availability Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAssignedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(gomock.Any(), testTagNumber).
					Return(false, errors.New("database error")).
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
			if err := s.TagAssigned(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_TagAssignmentRequested(t *testing.T) {
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
			name: "Successful Tag Assignment Requested",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil).Times(1)
				f.LeaderboardDB.EXPECT().DeactivateLeaderboard(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				f.LeaderboardDB.EXPECT().CreateLeaderboard(gomock.Any(), gomock.Any()).Return(int64(1), nil).Times(1)
				f.EventBus.EXPECT().Publish(leaderboardevents.TagAssignmentProcessed, gomock.Any()).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database error")).Times(1)
			},
		},
		{
			name: "Create Leaderboard Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{}, nil).Times(1)
				f.LeaderboardDB.EXPECT().DeactivateLeaderboard(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				f.LeaderboardDB.EXPECT().CreateLeaderboard(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("database error")).Times(1)
				// Expect publishTagAssignmentFailed to be called due to the error
				f.EventBus.EXPECT().
					Publish(leaderboardevents.TagAssignmentFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.TagAssignmentFailed {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.TagAssignmentFailed, topic)
						}
						// Add more assertions on the message if needed
						return nil
					}).
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
			if err := s.TagAssignmentRequested(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagAssignmentRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
