package roundhandlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservicemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data ---
const (
	reminderHandlerRoundID       = "some-round-id"
	reminderHandlerCorrelationID = "some-correlation-id"
	reminderHandlerType          = "1h"
	reminderHandlerRoundTitle    = "Test Round"
	reminderProcessError         = "processing error"
)

var (
	reminderHandlerLocation  = "Test Location" // var for pointer
	reminderHandlerNow       = time.Now().UTC().Truncate(time.Second)
	reminderHandlerStartTime = &reminderHandlerNow

	validReminderHandlerPayload = roundevents.RoundReminderPayload{
		RoundID:      reminderHandlerRoundID,
		ReminderType: reminderHandlerType,
		RoundTitle:   reminderHandlerRoundTitle,
		StartTime:    reminderHandlerStartTime,
		Location:     &reminderHandlerLocation,
	}
)

func TestRoundHandlers_HandleRoundReminder(t *testing.T) {
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
			name:    "Successful round reminder processing",
			payload: validReminderHandlerPayload, // Use constant
			mockExpects: func() {
				mockRoundService.EXPECT().
					ProcessRoundReminder(gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name:        "Invalid payload",
			payload:     "invalid json", // Invalid JSON
			mockExpects: func() {},      // No service calls expected
			wantErr:     true,
			errMsg:      "failed to unmarshal RoundReminderPayload",
		},
		{
			name:    "Failed to process round reminder",
			payload: validReminderHandlerPayload, // Use constant
			mockExpects: func() {
				mockRoundService.EXPECT().
					ProcessRoundReminder(gomock.Any()).
					Return(fmt.Errorf(reminderProcessError)). // Simulate processing error
					Times(1)
			},
			wantErr: true,
			errMsg:  "failed to handle RoundReminder event: " + reminderProcessError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: mockRoundService, // Inject mock service
				logger:       logger,
			}

			// Create message directly using json.Marshal
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, reminderHandlerCorrelationID)

			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := h.HandleRoundReminder(msg) // Pass the message

			if tt.wantErr {
				if err == nil {
					t.Error("HandleRoundReminder() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("HandleRoundReminder() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("HandleRoundReminder() unexpected error: %v", err)
				}
			}
		})
	}
}
