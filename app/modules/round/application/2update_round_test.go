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
	storeUpdateRoundID       = "some-round-id"
	storeUpdateCorrelationID = "some-correlation-id"
	storeUpdateDBError       = "db error"
	storeUpdatePublishError  = "publish error"
	storeUpdateCancelError   = "cancel error"
)

var (
	storeUpdateTitle     = "Updated Title"
	storeUpdateLocation  = "Updated Location"
	storeUpdateEventType = "Updated Type" // Use var for pointer
	storeUpdateTimeNow   = time.Now().UTC().Truncate(time.Second)
	storeUpdateStartTime = &storeUpdateTimeNow

	// Pre-built payload
	validStoredUpdatePayload = roundevents.RoundEntityUpdatedPayload{
		Round: roundtypes.Round{
			ID:        storeUpdateRoundID,
			Title:     storeUpdateTitle,
			Location:  &storeUpdateLocation,  // Now takes the address
			EventType: &storeUpdateEventType, // Now takes the address
			StartTime: storeUpdateStartTime,
			State:     roundtypes.RoundStateUpcoming,
		},
	}

	validRoundUpdatedPayload = roundevents.RoundUpdatedPayload{
		RoundID: storeUpdateRoundID,
	}
)

func TestRoundService_StoreRoundUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
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
		mockDBSetup   func()
		expectedEvent string
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "Successful round update storage",
			payload:       validStoredUpdatePayload, // Use pre-built payload
			expectedEvent: roundevents.RoundUpdated,
			wantErr:       false,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(storeUpdateRoundID), gomock.Any()).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).
					Return(nil). // Mock successful publishing
					Times(1)
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: "round.update.error",
			wantErr:       true,
			errMsg:        "invalid payload",
			mockDBSetup: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq("round.update.error"), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
		},
		{
			name:    "Database error",
			payload: validStoredUpdatePayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(storeUpdateRoundID), gomock.Any()).
					Return(fmt.Errorf(storeUpdateDBError)). // Simulate DB error
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq("round.update.error"), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
			expectedEvent: "round.update.error",
			wantErr:       true,
			errMsg:        storeUpdateDBError, // Check for the specific error
		},
		{
			name:    "Publish RoundUpdated event fails",
			payload: validStoredUpdatePayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(storeUpdateRoundID), gomock.Any()).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).
					Return(fmt.Errorf(storeUpdatePublishError)). // Simulate publish error
					Times(1)
			},
			expectedEvent: roundevents.RoundUpdated, // Event *should* be published
			wantErr:       true,
			errMsg:        storeUpdatePublishError, // Check specific publish error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, storeUpdateCorrelationID)

			if tt.mockDBSetup != nil {
				tt.mockDBSetup()
			}

			err := s.StoreRoundUpdate(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("StoreRoundUpdate() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("StoreRoundUpdate() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("StoreRoundUpdate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockErrorReporter := errors.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
	logger := slog.Default()

	s := &RoundService{
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
			name:          "Successful update of scheduled round events",
			payload:       validRoundUpdatedPayload,
			expectedEvent: roundevents.RoundScheduleUpdate,
			wantErr:       false,
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), gomock.Eq(storeUpdateRoundID)).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScheduleUpdate), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "invalid payload",
		},
		{
			name:          "Failed to cancel existing scheduled events",
			payload:       validRoundUpdatedPayload,
			expectedEvent: "",
			wantErr:       true,
			errMsg:        "failed to cancel existing scheduled events: " + storeUpdateCancelError,
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), gomock.Eq(storeUpdateRoundID)).
					Return(fmt.Errorf(storeUpdateCancelError)). // Simulate cancel error
					Times(1)
			},
		},
		{
			name:          "Failed to publish round.schedule.update event",
			payload:       validRoundUpdatedPayload,
			expectedEvent: roundevents.RoundScheduleUpdate,
			wantErr:       true,
			errMsg:        "failed to publish event",
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), gomock.Eq(storeUpdateRoundID)).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundScheduleUpdate), gomock.Any()).
					Return(fmt.Errorf(storeUpdatePublishError)).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, storeUpdateCorrelationID)

			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := s.UpdateScheduledRoundEvents(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdateScheduledRoundEvents() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("UpdateScheduledRoundEvents() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateScheduledRoundEvents() unexpected error: %v", err)
				}
			}
		})
	}
}
