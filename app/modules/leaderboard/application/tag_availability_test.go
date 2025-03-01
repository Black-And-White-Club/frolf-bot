package leaderboardservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagAvailabilityCheckRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testDiscordID := leaderboardtypes.DiscordID("testDiscordID")
	testTagNumber := 123
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
			name: "Successful Tag Availability Check (Available)",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAvailabilityCheckRequestedPayload{
					DiscordID: testDiscordID,
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f *fields, a *args) {
				// Mock active leaderboard
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(a.ctx).
					Return(&leaderboarddbtypes.Leaderboard{
						LeaderboardData: map[int]string{},
						IsActive:        true,
					}, nil).
					Times(1)

				// Mock tag availability check
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(a.ctx, testTagNumber).
					Return(true, nil).
					Times(1)

				// Mock publishing TagAssignmentRequested
				f.EventBus.EXPECT().
					Publish(leaderboardevents.LeaderboardTagAssignmentRequested, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.LeaderboardTagAssignmentRequested {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.LeaderboardTagAssignmentRequested, topic)
						}
						msg := msgs[0]
						payload := leaderboardevents.TagAssignmentRequestedPayload{}
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Errorf("Failed to unmarshal message payload: %v", err)
						}
						if payload.DiscordID != testDiscordID {
							t.Errorf("Expected DiscordID %s, got %s", testDiscordID, payload.DiscordID)
						}
						if payload.TagNumber != testTagNumber {
							t.Errorf("Expected TagNumber %d, got %d", testTagNumber, payload.TagNumber)
						}
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Successful Tag Availability Check (Unavailable)",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAvailabilityCheckRequestedPayload{
					DiscordID: testDiscordID,
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f *fields, a *args) {
				// Mock active leaderboard
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(a.ctx).
					Return(&leaderboarddbtypes.Leaderboard{
						LeaderboardData: map[int]string{
							testTagNumber: string(testDiscordID),
						},
						IsActive: true,
					}, nil).
					Times(1)

				// Mock tag availability check
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(a.ctx, testTagNumber).
					Return(false, nil).
					Times(1)

				// Mock publishing TagUnavailable
				f.EventBus.EXPECT().
					Publish(leaderboardevents.TagUnavailable, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.TagUnavailable {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.TagUnavailable, topic)
						}
						msg := msgs[0]
						payload := leaderboardevents.TagUnavailablePayload{}
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Errorf("Failed to unmarshal message payload: %v", err)
						}
						if payload.DiscordID != testDiscordID {
							t.Errorf("Expected DiscordID %s, got %s", testDiscordID, payload.DiscordID)
						}
						if payload.TagNumber != testTagNumber {
							t.Errorf("Expected TagNumber %d, got %d", testTagNumber, payload.TagNumber)
						}
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Database Error GetActiveLeaderboard",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAvailabilityCheckRequestedPayload{
					DiscordID: testDiscordID,
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f *fields, a *args) {
				// Mock database error
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(a.ctx).
					Return(nil, errors.New("database error")).
					Times(1)
			},
		},
		{
			name: "Database Error CheckTagAvailability",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: testCtx,
				msg: createTestMessageWithPayload(testCorrelationID, &leaderboardevents.TagAvailabilityCheckRequestedPayload{
					DiscordID: testDiscordID,
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f *fields, a *args) {
				// Mock active leaderboard
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(a.ctx).
					Return(&leaderboarddbtypes.Leaderboard{
						LeaderboardData: make(map[int]string),
						IsActive:        true,
					}, nil).
					Times(1)

				// Mock database error on tag availability check
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(a.ctx, testTagNumber).
					Return(false, errors.New("database error")).
					Times(1)
			},
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
			if err := s.TagAvailabilityCheckRequested(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagAvailabilityCheckRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
