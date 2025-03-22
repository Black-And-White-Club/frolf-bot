package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data ---
const (
	scheduleRoundID              = 1
	scheduleCorrelationID        = "some-correlation-id"
	scheduleTitle                = "Test Round"
	schedulePublishError  string = "publish error"
)

var (
	scheduleLocation     roundtypes.Location  = "Test Location"
	scheduleEventType    roundtypes.EventType = "casual"
	scheduleNow                               = time.Now().UTC().Truncate(time.Second)
	scheduleStartTime                         = roundtypes.StartTime(scheduleNow.Add(2 * time.Hour))
	validSchedulePayload                      = roundevents.RoundStoredPayload{
		Round: roundtypes.Round{
			ID:        scheduleRoundID,
			Title:     scheduleTitle,
			Location:  &scheduleLocation, // Correctly taking the address
			EventType: &scheduleEventType,
			StartTime: &scheduleStartTime,
			State:     roundtypes.RoundStateUpcoming,
		},
	}
)

func TestRoundService_ScheduleRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	mockErrorReporter := errors.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
	logger := slog.Default()

	s := &RoundService{
		RoundDB:       mockRoundDB,
		EventBus:      mockEventBus,
		logger:        logger,
		ErrorReporter: mockErrorReporter,
	}
	tests := []struct {
		name          string
		payload       interface{}
		mockExpects   func()
		expectedEvent string
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "Successful scheduling of round events",
			payload:       validSchedulePayload,
			expectedEvent: roundevents.RoundScheduled,
			wantErr:       false,
			mockExpects: func() {
				// For reminder message
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).Times(1)

				// For ScheduleRoundProcessing - no return value
				mockEventBus.EXPECT().ScheduleRoundProcessing(
					gomock.Any(),
					roundtypes.ID(scheduleRoundID),
					gomock.Any(),
				).Times(1)

				// For final event
				mockEventBus.EXPECT().Publish(roundevents.RoundScheduled, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to unmarshal RoundStoredPayload",
		},
		{
			name:          "Failed to schedule 1-hour reminder",
			payload:       validSchedulePayload,
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to schedule 1h reminder: " + schedulePublishError,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed to schedule 1h reminder: %s", schedulePublishError)).Times(1)
			},
		},
		{
			name:          "Failed to publish final event",
			payload:       validSchedulePayload,
			expectedEvent: roundevents.RoundScheduled,
			wantErr:       true,
			errMsg:        "failed to publish " + roundevents.RoundScheduled + " event: " + schedulePublishError,
			mockExpects: func() {
				// For reminder message
				mockEventBus.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).Times(1)

				// For ScheduleRoundProcessing - no return value
				mockEventBus.EXPECT().ScheduleRoundProcessing(
					gomock.Any(),
					roundtypes.ID(scheduleRoundID),
					gomock.Any(),
				).Times(1)

				// For final event
				mockEventBus.EXPECT().Publish(roundevents.RoundScheduled, gomock.Any()).Return(fmt.Errorf("failed to publish %s event: %s", roundevents.RoundScheduled, schedulePublishError)).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, scheduleCorrelationID)
			msg.Metadata.Set("event_type", roundevents.RoundCreateRequest)

			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := s.ScheduleRoundEvents(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("ScheduleRoundEvents() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ScheduleRoundEvents() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ScheduleRoundEvents() unexpected error: %v", err)
				}
			}
		})
	}
}
