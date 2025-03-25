package leaderboardhandlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleGetLeaderboardRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

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
			name: "Successful Get Leaderboard Request",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.GetLeaderboardRequestPayload{}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().GetLeaderboardRequest(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Service Layer Error",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.GetLeaderboardRequestPayload{}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().GetLeaderboardRequest(gomock.Any(), a.msg).Return(errors.New("service error")).Times(1)
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
			if err := h.HandleGetLeaderboardRequest(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleGetLeaderboardRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleGetTagByUserIDRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testUserID := "testUserID"
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
			name: "Successful Get Tag By Discord ID Request",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.GetTagByUserIDRequestPayload{
					UserID: leaderboardtypes.UserID(testUserID),
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				// Ensure correlation ID is added
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				// Set mock expectations
				f.leaderboardService.EXPECT().
					GetTagByUserIDRequest(gomock.Any(), gomock.AssignableToTypeOf(&message.Message{})).
					DoAndReturn(func(ctx context.Context, msg *message.Message) error {
						// Verify metadata in the message
						if msg.Metadata.Get(middleware.CorrelationIDMetadataKey) != testCorrelationID {
							t.Errorf("Expected correlation ID %s, got %s", testCorrelationID, msg.Metadata.Get(middleware.CorrelationIDMetadataKey))
						}
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
				msg: message.NewMessage(testCorrelationID, []byte(`{invalid_json}`)), // Invalid JSON
			},
			wantErr: true,
			setup: func(f fields, a args) {
				// Ensure the service is NOT called
				f.leaderboardService.EXPECT().
					GetTagByUserIDRequest(gomock.Any(), gomock.Any()).
					Times(0)
			},
		},
		{
			name: "Service Layer Error",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.GetTagByUserIDRequestPayload{
					UserID: leaderboardtypes.UserID(testUserID),
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				// Add correlation ID to metadata
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				// Set expectation for a service layer error
				f.leaderboardService.EXPECT().
					GetTagByUserIDRequest(gomock.Any(), gomock.AssignableToTypeOf(&message.Message{})).
					Return(fmt.Errorf("service error")).
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
			if err := h.HandleGetTagByUserIDRequest(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleGetTagByUserIDRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
