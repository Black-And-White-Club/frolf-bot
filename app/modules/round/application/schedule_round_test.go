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
	scheduleRoundID              = "some-uuid"
	scheduleCorrelationID        = "some-correlation-id"
	scheduleTitle         string = "Test Round"
	schedulePublishError  string = "publish error"
)

var (
	scheduleLocation     = "Test Location" // Now a var
	scheduleEventType    = "casual"        // Now a var
	scheduleNow          = time.Now().UTC().Truncate(time.Second)
	scheduleStartTime    = scheduleNow.Add(2 * time.Hour)
	validSchedulePayload = roundevents.RoundStoredPayload{
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
	mockRoundDB := rounddb.NewMockRoundDB(ctrl) //Not used here, but good practice to keep
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
			payload:       validSchedulePayload,       // Use constant
			expectedEvent: roundevents.RoundScheduled, // Expect final event
			wantErr:       false,
			mockExpects: func() {
				// Expect two calls for delayed messages and one for final confirmation
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(nil).Times(2)
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
			expectedEvent: "", // No final event if scheduling fails
			wantErr:       true,
			errMsg:        "failed to schedule 1h reminder: " + schedulePublishError,

			mockExpects: func() {
				// Expect only the first publish to fail
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(fmt.Errorf("failed to schedule 1h reminder: %s", schedulePublishError)).Times(1)
			},
		},
		{
			name:          "Failed to schedule round start",
			payload:       validSchedulePayload,
			expectedEvent: "", // No final event
			wantErr:       true,
			errMsg:        "failed to schedule round start: " + schedulePublishError,
			mockExpects: func() {
				// First publish succeeds, second fails
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(nil).Times(1)
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(fmt.Errorf("failed to schedule round start: %s", schedulePublishError)).Times(1)
			},
		},
		{
			name:          "Failed to publish final event",
			payload:       validSchedulePayload,
			expectedEvent: roundevents.RoundScheduled, //Still expect this topic
			wantErr:       true,
			errMsg:        "failed to publish " + roundevents.RoundScheduled + " event: " + schedulePublishError,

			mockExpects: func() {
				// Both delayed messages succeed, but final publish fails
				mockEventBus.EXPECT().Publish(roundevents.DelayedMessagesSubject, gomock.Any()).Return(nil).Times(2)
				mockEventBus.EXPECT().Publish(roundevents.RoundScheduled, gomock.Any()).Return(fmt.Errorf("failed to publish %s event: %s", roundevents.RoundScheduled, schedulePublishError)).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, scheduleCorrelationID)
			msg.Metadata.Set("event_type", roundevents.RoundCreateRequest) //Need this for the switch case

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
