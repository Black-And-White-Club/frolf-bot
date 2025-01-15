package roundhandlers

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleParticipantResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		msg     *message.Message
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "response": "accept"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					JoinRound(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "UnmarshalError",
			msg:     message.NewMessage(watermill.NewUUID(), []byte(`invalid json`)),
			wantErr: true,
		},
		{
			name: "JoinRoundError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "response": "accept"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					JoinRound(gomock.Any(), gomock.Any()).
					Return(errors.New("join round error"))
			},
			wantErr: true,
		},
		// Add more test cases for other scenarios...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			h := &RoundHandlers{
				RoundService: mockRoundService,
				logger:       logger,
			}
			if err := h.HandleParticipantResponse(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleParticipantResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name    string
		msg     *message.Message
		setup   func()
		wantErr bool
	}{
		{
			name: "Success - Regular Update",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "score": 10,
                "update_type": 0 
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					UpdateScore(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Success - Admin Update",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "score": 10,
                "update_type": 1 
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					UpdateScoreAdmin(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "UnmarshalError",
			msg:     message.NewMessage(watermill.NewUUID(), []byte(`invalid json`)),
			wantErr: true,
		},
		{
			name: "UpdateScoreError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "score": 10,
                "update_type": 0 
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					UpdateScore(gomock.Any(), gomock.Any()).
					Return(errors.New("update score error"))
			},
			wantErr: true,
		},
		{
			name: "UpdateScoreAdminError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "score": 10,
                "update_type": 1 
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					UpdateScoreAdmin(gomock.Any(), gomock.Any()).
					Return(errors.New("update score admin error"))
			},
			wantErr: true,
		},
		{
			name: "InvalidUpdateType",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "participant": "user-456",
                "score": 10,
                "update_type": 2 
            }`)),
			wantErr: true,
		},
		// Add more test cases for other scenarios...
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			h := &RoundHandlers{
				RoundService: mockRoundService,
				logger:       logger,
			}
			if err := h.HandleScoreUpdated(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
