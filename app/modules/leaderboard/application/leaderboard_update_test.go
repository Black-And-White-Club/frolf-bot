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
	leaderboarddbtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

type contextKey string

const correlationIDKey contextKey = "correlationID"

func TestLeaderboardService_RoundFinalized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testRoundID := "testRoundID"
	testSortedTags := []string{"1:a", "2:b", "3:c"}
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
			name: "Successful Round Finalized",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.RoundFinalizedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				// Expect the EventBus to publish a LeaderboardUpdateRequested event
				f.EventBus.EXPECT().
					Publish(leaderboardevents.LeaderboardUpdateRequested, gomock.Any()).
					DoAndReturn(func(topic string, msgs ...*message.Message) error {
						if topic != leaderboardevents.LeaderboardUpdateRequested {
							t.Errorf("Expected topic %s, got %s", leaderboardevents.LeaderboardUpdateRequested, topic)
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
						var payload leaderboardevents.LeaderboardUpdateRequestedPayload
						err := json.Unmarshal(msg.Payload, &payload)
						if err != nil {
							t.Fatalf("Failed to unmarshal message payload: %v", err)
						}

						if payload.RoundID != testRoundID {
							t.Errorf("Expected RoundID %s, got %s", testRoundID, payload.RoundID)
						}

						if len(payload.SortedParticipantTags) != len(testSortedTags) {
							t.Errorf("Expected SortedParticipantTags length %d, got %d", len(testSortedTags), len(payload.SortedParticipantTags))
						}

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
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.RoundFinalizedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				f.EventBus.EXPECT().
					Publish(leaderboardevents.LeaderboardUpdateRequested, gomock.Any()).
					Return(errors.New("publish error")).
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
				tt.setup(tt.fields, tt.args)
			}
			if err := s.RoundFinalized(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_LeaderboardUpdateRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eventUtil := eventutil.NewEventUtil()

	testRoundID := "testRoundID"
	testSortedTags := []string{"1:a", "2:b", "3:c"}
	testLeaderboardData := map[int]string{1: "a", 2: "b", 3: "c"}
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
			name: "Successful Leaderboard Update Requested",
			fields: fields{
				LeaderboardDB: mockLeaderboardDB,
				EventBus:      mockEventBus,
				logger:        logger,
				eventUtil:     eventUtil,
			},
			args: args{
				ctx: context.WithValue(context.Background(), correlationIDKey, testCorrelationID),
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				// Mock the database calls
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(gomock.Any()).
					Return(&leaderboarddbtypes.Leaderboard{ID: 1, LeaderboardData: testLeaderboardData}, nil).
					Times(1)
				f.LeaderboardDB.EXPECT().
					CreateLeaderboard(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, leaderboard *leaderboarddbtypes.Leaderboard) (int64, error) {
						// Simulate creating a new leaderboard
						return 1, nil
					}).
					Times(1)
				f.LeaderboardDB.EXPECT().
					DeactivateLeaderboard(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

				// Expect the EventBus to publish a LeaderboardUpdated event
				f.EventBus.EXPECT().
					Publish(leaderboardevents.LeaderboardUpdated, gomock.Any()).
					Return(nil). // Assuming successful publish
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
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				// Mock the GetActiveLeaderboard call to return an error
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
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
				msg: createTestMessageWithPayload(testCorrelationID, leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				// Mock the GetActiveLeaderboard call to return an error
				f.LeaderboardDB.EXPECT().
					GetActiveLeaderboard(gomock.Any()).
					Return(&leaderboarddbtypes.Leaderboard{ID: 1, LeaderboardData: testLeaderboardData}, nil).
					Times(1)
				f.LeaderboardDB.EXPECT().
					CreateLeaderboard(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database error")).
					Times(1)
				// Expect the EventBus to publish a LeaderboardUpdateFailed event
				f.EventBus.EXPECT().
					Publish(leaderboardevents.LeaderboardUpdateFailed, gomock.Any()).
					Return(nil). // Assuming successful publish
					Times(1)
			},
		},
		// ... (add more test cases for other scenarios) ...
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
			if err := s.LeaderboardUpdateRequested(tt.args.ctx, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.HandleLeaderboardUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create a test message with a payload
func createTestMessageWithPayload(correlationID string, payload interface{}) *message.Message {
	payloadBytes, _ := json.Marshal(payload)
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	// Set metadata after message creation
	msg.Metadata = make(message.Metadata)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	return msg
}
