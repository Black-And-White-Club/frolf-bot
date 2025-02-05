package roundhandlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservicemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservicemocks.NewMockService(ctrl)
	logger := slog.Default()

	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name        string
		args        args
		mockExpects func()
		wantErr     bool
	}{
		{
			name: "Successful round reminder processing",
			args: args{
				msg: createTestMessageReminder(roundevents.RoundReminderPayload{
					RoundID:      "some-round-id",
					ReminderType: "1h",
					RoundTitle:   "Test Round",
					StartTime:    time.Now().Add(1 * time.Hour),
					Location:     "Test Location",
				}),
			},
			mockExpects: func() {
				mockRoundService.EXPECT().
					ProcessRoundReminder(gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "Invalid payload",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid json")),
			},
			mockExpects: func() {},
			wantErr:     true,
		},
		{
			name: "Failed to process round reminder",
			args: args{
				msg: createTestMessageReminder(roundevents.RoundReminderPayload{
					RoundID:      "some-round-id",
					ReminderType: "1h",
					RoundTitle:   "Test Round",
					StartTime:    time.Now().Add(1 * time.Hour),
					Location:     "Test Location",
				}),
			},
			mockExpects: func() {
				mockRoundService.EXPECT().
					ProcessRoundReminder(gomock.Any()).
					Return(fmt.Errorf("processing error")).
					Times(1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockExpects()

			h := &RoundHandlers{
				RoundService: mockRoundService,
				logger:       logger,
			}

			if err := h.HandleRoundReminder(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundReminder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func createTestMessageReminder(payload roundevents.RoundReminderPayload) *message.Message {
	payloadBytes, _ := json.Marshal(payload)
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlationID", payload.RoundID)
	return msg
}
