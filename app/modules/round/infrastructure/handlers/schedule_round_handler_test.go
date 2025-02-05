package roundhandlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservicemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleScheduleRoundEvents(t *testing.T) {
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
			name: "Successful round scheduling",
			args: args{
				msg: createTestMessage(roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Test Round",
						Location:  "Test Location",
						StartTime: time.Now().Add(2 * time.Hour),
					},
				}),
			},
			mockExpects: func() {
				mockRoundService.EXPECT().
					ScheduleRoundEvents(gomock.Any(), gomock.Any()).
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
			name: "Failed to schedule round events",
			args: args{
				msg: createTestMessage(roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						Title:     "Test Round",
						Location:  "Test Location",
						StartTime: time.Now().Add(2 * time.Hour),
					},
				}),
			},
			mockExpects: func() {
				mockRoundService.EXPECT().
					ScheduleRoundEvents(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("scheduling error")).
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

			if err := h.HandleScheduleRoundEvents(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleScheduleRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func createTestMessage(payload roundevents.RoundStoredPayload) *message.Message {
	payloadBytes, _ := json.Marshal(payload)
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlationID", payload.Round.ID)
	return msg
}
