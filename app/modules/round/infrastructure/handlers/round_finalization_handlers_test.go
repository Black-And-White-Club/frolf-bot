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

func TestRoundHandlers_HandleFinalizeRound(t *testing.T) {
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
					FinalizeRound(gomock.Any(), gomock.Any()).
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
			name: "FinalizeRoundError",
			msg: message.NewMessage(watermill.NewUUID(), []byte(`{
                "round_id": "round-123"
            }`)),
			setup: func() {
				mockRoundService.EXPECT().
					FinalizeRound(gomock.Any(), gomock.Any()).
					Return(errors.New("finalize round error"))
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
			if err := h.HandleFinalizeRound(context.Background(), tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleFinalizeRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
