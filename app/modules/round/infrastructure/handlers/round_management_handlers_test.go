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

func TestRoundHandlers_HandleCreateRound(t *testing.T) {
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
				"title": "Test Round",
				"location": "Test Location",
				"date_time": {
					"date": "2024-01-15",
					"time": "10:00"
				},
				"participants": [
					{
						"discord_id": "user-123",
						"tag_number": 1,
						"response": "ACCEPT"
					}
				]
			}`)),
			setup: func() {
				mockRoundService.EXPECT().
					CreateRound(gomock.Any(), gomock.Any()).
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
			name: "CreateRoundError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
				"title": "Test Round",
				"location": "Test Location",
				"date_time": {
					"date": "2024-01-15",
					"time": "10:00"
				},
				"participants": [
					{
						"discord_id": "user-123",
						"tag_number": 1,
						"response": "ACCEPT"
					}
				]
			}`)),
			setup: func() {
				mockRoundService.EXPECT().
					CreateRound(gomock.Any(), gomock.Any()).
					Return(errors.New("create round error"))
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
			if err := h.HandleCreateRound(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleCreateRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundHandlers_HandleUpdateRound(t *testing.T) {
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
                "title": "Updated Title"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					UpdateRound(gomock.Any(), gomock.Any()).
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
			name: "UpdateRoundError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123",
                "title": "Updated Title"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					UpdateRound(gomock.Any(), gomock.Any()).
					Return(errors.New("update round error"))
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
			if err := h.HandleUpdateRound(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleUpdateRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundHandlers_HandleDeleteRound(t *testing.T) {
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
                "round_id": "round-123"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					DeleteRound(gomock.Any(), gomock.Any()).
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
			name: "DeleteRoundError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					DeleteRound(gomock.Any(), gomock.Any()).
					Return(errors.New("delete round error"))
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
			if err := h.HandleDeleteRound(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleDeleteRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundHandlers_HandleStartRound(t *testing.T) {
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
                "round_id": "round-123"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					StartRound(gomock.Any(), gomock.Any()).
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
			name: "StartRoundError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					StartRound(gomock.Any(), gomock.Any()).
					Return(errors.New("start round error"))
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
			if err := h.HandleStartRound(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleStartRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
