package leaderboardhandlers

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestLeaderboardHandlers_HandleTagSwapRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testRequestorID := "requestorID"
	testTargetID := "targetID"
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
			name: "Successful Tag Swap Requested",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagSwapRequested(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagSwapRequestedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagSwapRequested(gomock.Any(), a.msg).Return(errors.New("service error")).Times(1)
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
			if err := h.HandleTagSwapRequested(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleTagSwapRequested() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardHandlers_HandleTagSwapInitiated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLeaderboardService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testRequestorID := "requestorID"
	testTargetID := "targetID"
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
			name: "Successful Tag Swap Initiated",
			fields: fields{
				leaderboardService: mockLeaderboardService,
				logger:             logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagSwapInitiatedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagSwapInitiated(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, testCorrelationID, leaderboardevents.TagSwapInitiatedPayload{
					RequestorID: testRequestorID,
					TargetID:    testTargetID,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.leaderboardService.EXPECT().TagSwapInitiated(gomock.Any(), a.msg).Return(errors.New("service error")).Times(1)
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
			if err := h.HandleTagSwapInitiated(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardHandlers.HandleTagSwapInitiated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
