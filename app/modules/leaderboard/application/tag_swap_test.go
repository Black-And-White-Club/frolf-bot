package leaderboardservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboarddbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagSwapRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testRequestorID := "requestorID"
	testTargetID := "targetID"
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
			name: "Successful Tag Swap Requested",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: map[int]string{1: testRequestorID, 2: testTargetID},
				}, nil).Times(1)
				f.EventBus.EXPECT().Publish(leaderboardevents.TagSwapInitiated, gomock.Any()).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(nil, errors.New("database error")).Times(1)
			},
		},
		{
			name: "User Not Found",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: map[int]string{1: "someOtherID", 2: "anotherID"},
				}, nil).Times(1)
			},
		},
		{
			name: "Publish Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
					LeaderboardData: map[int]string{1: testRequestorID, 2: testTargetID},
				}, nil).Times(1)
				f.EventBus.EXPECT().Publish(leaderboardevents.TagSwapInitiated, gomock.Any()).Return(errors.New("publish error")).Times(1)
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
			if err := s.TagSwapRequested(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagSwapRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_TagSwapInitiated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testRequestorID := "requestorID"
	testTargetID := "targetID"
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
			name: "Successful Tag Swap Initiated",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapInitiatedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().SwapTags(gomock.Any(), testRequestorID, testTargetID).Return(nil).Times(1)
				f.EventBus.EXPECT().Publish(leaderboardevents.TagSwapProcessed, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Swap Tags DB Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapInitiatedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().SwapTags(gomock.Any(), testRequestorID, testTargetID).Return(errors.New("database error")).Times(1)
				f.EventBus.EXPECT().Publish(leaderboardevents.TagSwapFailed, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish Processed Event Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagSwapInitiatedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: false, // Expecting no error at the service level since we only log the publish error
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().SwapTags(gomock.Any(), testRequestorID, testTargetID).Return(nil).Times(1)
				f.EventBus.EXPECT().Publish(leaderboardevents.TagSwapProcessed, gomock.Any()).Return(errors.New("publish error")).Times(1)
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
			if err := s.TagSwapInitiated(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagSwapInitiated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
