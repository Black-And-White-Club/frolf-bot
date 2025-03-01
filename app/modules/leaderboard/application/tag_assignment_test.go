package leaderboardservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testDiscordID := leaderboardtypes.DiscordID("testDiscordID")
	testTagNumber := 123
	testUpdateID := "testUpdateID"
	testCorrelationID := watermill.NewUUID()
	type contextKey string
	const correlationIDKey contextKey = "correlationID"

	testCtx := context.WithValue(context.Background(), correlationIDKey, testCorrelationID)

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
		setup   func(f *fields, a *args)
	}{
		{
			name: "Successful Tag Assignment",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID:  testDiscordID,
					TagNumber:  testTagNumber,
					UpdateID:   testUpdateID,
					Source:     "user",
					UpdateType: "new_tag",
				}),
			},
			wantErr: false,
			setup: func(f *fields, a *args) {
				f.LeaderboardDB.EXPECT().
					AssignTag(a.ctx, testDiscordID, testTagNumber, gomock.Any(), testUpdateID).
					Return(nil).
					Times(1)

				f.EventBus.EXPECT().
					Publish(leaderboardevents.TagAssigned, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.TagAssigned {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.TagAssigned, topic)
						}
						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}
						msg := msgs[0]
						if msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, msg.Metadata.Get(middleware.CorrelationIDMetadataKey))
						}
						// Further assertions on the message payload can be added here
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "AssignTag Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID:  testDiscordID,
					TagNumber:  testTagNumber,
					UpdateID:   testUpdateID,
					Source:     "user",
					UpdateType: "new_tag",
				}),
			},
			wantErr: true,
			setup: func(f *fields, a *args) {
				f.LeaderboardDB.EXPECT().
					AssignTag(a.ctx, testDiscordID, testTagNumber, gomock.Any(), testUpdateID).
					Return(errors.New("database error")).
					Times(1)

				f.EventBus.EXPECT().
					Publish(leaderboardevents.LeaderboardTagAssignmentFailed, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.LeaderboardTagAssignmentFailed {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.LeaderboardTagAssignmentFailed, topic)
						}
						// Further assertions on the message payload can be added here
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Publish TagAssigned Error",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID:  testDiscordID,
					TagNumber:  testTagNumber,
					UpdateID:   testUpdateID,
					Source:     "user",
					UpdateType: "new_tag",
				}),
			},
			wantErr: true,
			setup: func(f *fields, a *args) {
				f.LeaderboardDB.EXPECT().
					AssignTag(a.ctx, testDiscordID, testTagNumber, gomock.Any(), testUpdateID).
					Return(nil).
					Times(1)

				f.EventBus.EXPECT().
					Publish(leaderboardevents.TagAssigned, gomock.Any()).
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
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, "invalid-payload"),
			},
			wantErr: true,
			setup:   func(f *fields, a *args) {},
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
				tt.setup(&tt.fields, &tt.args)
			}
			if err := s.TagAssignmentRequested(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagAssignmentRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
