package leaderboardhandlers

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/application/mocks"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "testDiscordID"
	testTagNumber := 123
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
			name: "Successful Tag Assigned",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagAssignedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagAssigned(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagAssignedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagAssigned(gomock.Any(), a.msg).Return(errors.New("service error")).Times(1)
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
			if err := h.HandleTagAssigned(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleTagAssigned() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagAssignmentRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "testDiscordID"
	testTagNumber := 123
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
			name: "Successful Tag Assignment Requested",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagAssignmentRequested(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagAssignmentRequestedPayload{
					DiscordID: leaderboardtypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagAssignmentRequested(gomock.Any(), a.msg).Return(errors.New("service error")).Times(1)
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
			if err := h.HandleTagAssignmentRequested(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleTagAssignmentRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
