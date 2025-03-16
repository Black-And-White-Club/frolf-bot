package roundhandlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	roundservicemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data ---
const (
	scheduleHandlerRoundID       = "some-round-id"
	scheduleHandlerCorrelationID = "some-correlation-id"
	scheduleHandlerTitle         = "Test Round"
	scheduleHandlerScheduleError = "scheduling error"
)

var (
	scheduleHandlerLocation     = "Test Location"
	scheduleHandlerEventType    = "casual"
	scheduleHandlerNow          = time.Now().UTC().Truncate(time.Second)
	scheduleHandlerStartTime    = &scheduleHandlerNow
	validScheduleHandlerPayload = roundevents.RoundStoredPayload{
		Round: roundtypes.Round{
			RoundID:   scheduleHandlerRoundID,
			Title:     scheduleHandlerTitle,
			Location:  &scheduleHandlerLocation,
			EventType: &scheduleHandlerEventType,
			StartTime: scheduleHandlerStartTime,
			State:     roundtypes.RoundStateUpcoming,
		},
	}
)

func TestRoundHandlers_HandleScheduleRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservicemocks.NewMockService(ctrl)
	logger := slog.Default()

	tests := []struct {
		name        string
		payload     interface{}
		mockExpects func()
		wantErr     bool
		errMsg      string
	}{
		{
			name:    "Successful round scheduling",
			payload: validScheduleHandlerPayload, // Use pre-built payload
			mockExpects: func() {
				mockRoundService.EXPECT().
					ScheduleRoundEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name:    "Invalid payload",
			payload: "invalid json", // Invalid JSON
			wantErr: true,
			errMsg:  "failed to unmarshal RoundStoredPayload",
		},
		{
			name:    "Failed to schedule round events",
			payload: validScheduleHandlerPayload, // Use pre-built payload
			mockExpects: func() {
				mockRoundService.EXPECT().
					ScheduleRoundEvents(gomock.Any(), gomock.Any()).
					Return(errors.New(scheduleHandlerScheduleError)). // Simulate scheduling error
					Times(1)
			},
			wantErr: true,
			errMsg:  "failed to handle RoundStored event: " + scheduleHandlerScheduleError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: mockRoundService,
				logger:       logger,
			}

			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, scheduleHandlerCorrelationID)
			msg.Metadata.Set("event_type", roundevents.RoundCreateRequest)

			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := h.HandleScheduleRoundEvents(msg)

			if tt.wantErr {
				if err == nil {
					t.Error("HandleScheduleRoundEvents() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("HandleScheduleRoundEvents() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("HandleScheduleRoundEvents() unexpected error: %v", err)
				}
			}
		})
	}
}
