package leaderboardservice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardService_TagAvailabilityCheckRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

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
			name: "Successful Tag Availability Check - Available",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAvailabilityCheckRequestedPayload{
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
					Publish(leaderboardevents.TagAvailabilityCheckResponded, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.TagAvailabilityCheckResponded {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.TagAvailabilityCheckResponded, topic)
						}

						if len(msgs) != 1 {
							t.Fatalf("Expected 1 message, got %d", len(msgs))
						}

						msg := msgs[0]

						if msg.UUID == "" {
							t.Errorf("Expected message UUID to be set, but it was empty")
						}

						// Verify the correlation ID in the message metadata
						correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
						if correlationID != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, correlationID)
						}

						// You can add more assertions on the message payload if needed
						var payload leaderboardevents.TagAvailabilityCheckRespondedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("Failed to unmarshal message payload: %v", err)
						}

						if payload.TagNumber != testTagNumber {
							t.Errorf("Expected TagNumber %d, got %d", testTagNumber, payload.TagNumber)
						}

						if !payload.IsAvailable {
							t.Errorf("Expected IsAvailable to be true, got false")
						}

						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Successful Tag Availability Check - Not Available",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAvailabilityCheckRequestedPayload{
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				f.LeaderboardDB.EXPECT().
					CheckTagAvailability(gomock.Any(), testTagNumber).
					Return(false, nil).
					Times(1)
				f.EventBus.EXPECT().
					Publish(leaderboardevents.TagAvailabilityCheckResponded, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						// ... assertions ...
						return nil
					}).
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
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.TagAvailabilityCheckRequestedPayload{
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
		// ... (add more test cases for unmarshal error, publish error, etc.) ...
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
			if err := s.TagAvailabilityCheckRequested(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.TagAvailabilityCheckRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
