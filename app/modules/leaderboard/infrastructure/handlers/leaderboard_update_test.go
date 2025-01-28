package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/application/mocks"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleRoundFinalized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testRoundID := "testRoundID"
	testSortedTags := []string{"1:a", "2:b", "3:c"}
	testCorrelationID := watermill.NewUUID()

	type fields struct {
		leaderboardService *mocks.MockService
		logger             *slog.Logger
	}
	type args struct {
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
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.RoundFinalizedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				// Add correlation ID to metadata
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				// Expect the service function to validate the correlation ID from the message metadata
				f.leaderboardService.EXPECT().
					RoundFinalized(gomock.Any(), a.msg). // Pass the full message
					DoAndReturn(func(ctx context.Context, msg *message.Message) error {
						// Verify the correlation ID in the metadata
						if msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %v", testCorrelationID, msg.Metadata.Get(middleware.CorrelationIDMetadataKey))
						}
						return nil
					}).
					Times(1)
			},
		},
		// Other test cases remain unchanged...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				leaderboardService: tt.fields.leaderboardService,
				logger:             tt.fields.logger,
			}
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}
			if err := h.HandleRoundFinalized(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleLeaderboardUpdateRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testRoundID := "testRoundID"
	testSortedTags := []string{"1:a", "2:b", "3:c"}
	testCorrelationID := watermill.NewUUID()

	type fields struct {
		leaderboardService *mocks.MockService
		logger             *slog.Logger
	}
	type args struct {
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
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().
					LeaderboardUpdateRequested(gomock.Any(), a.msg).
					DoAndReturn(func(ctx context.Context, msg *message.Message) error {
						return nil
					}).
					Times(1)
			},
		},
		{
			name: "Unmarshal Error",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, "invalid-payload"),
			},
			wantErr: true,
			setup:   func(f fields, a args) {},
		},
		{
			name: "Service Layer Error",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.LeaderboardUpdateRequestedPayload{
					RoundID:               testRoundID,
					SortedParticipantTags: testSortedTags,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().
					LeaderboardUpdateRequested(gomock.Any(), a.msg).
					Return(errors.New("service error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &LeaderboardHandlers{
				leaderboardService: tt.fields.leaderboardService,
				logger:             tt.fields.logger,
			}
			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}
			if err := h.HandleLeaderboardUpdateRequested(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleLeaderboardUpdateRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create a test message with a payload and correlation ID in metadata
func createTestMessageWithPayload(t *testing.T, correlationID string, payload interface{}) *message.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata = make(message.Metadata)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID) // Ensure correlation ID is set
	return msg
}
