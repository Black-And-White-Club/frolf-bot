package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddbmocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data ---

const (
	storeUpdateRoundID       roundtypes.ID = 1
	storeUpdateCorrelationID               = "some-correlation-id"
	storeUpdateCancelError                 = "cancel error"
	storeUpdatePublishError                = "publish error"
	storeUpdateDBError                     = "db error"
)

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddbmocks.NewMockRoundDBInterface(ctrl)
	mockErrorReporter := errors.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
	logger := slog.Default()

	s := &RoundService{
		EventBus:      mockEventBus,
		RoundDB:       mockRoundDB,
		logger:        logger,
		ErrorReporter: mockErrorReporter,
	}

	// Create a sample round for test responses
	sampleRound := &roundtypes.Round{
		ID:        storeUpdateRoundID,
		Title:     "Test Round",
		StartTime: nil,
		Location:  nil,
	}

	tests := []struct {
		name        string
		payload     interface{}
		mockExpects func()
		wantErr     bool
		errMsg      string
	}{
		{
			name: "Successful update of scheduled round events",
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID: storeUpdateRoundID,
			},
			wantErr: false,
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), storeUpdateRoundID).
					Return(nil).
					Times(1)

				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), storeUpdateRoundID).
					Return(sampleRound, nil).
					Times(1)

				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateReschedule), gomock.Any()).
					Do(func(topic string, payload interface{}) {
						msg, ok := payload.(*message.Message)
						if !ok {
							t.Errorf("Expected *message.Message, got %T", payload)
							return
						}

						// Unmarshal and verify the payload inside the message
						var storedPayload roundevents.RoundStoredPayload
						if err := json.Unmarshal(msg.Payload, &storedPayload); err != nil {
							t.Errorf("Failed to unmarshal published payload: %v", err)
							return
						}

						if storedPayload.Round.ID != storeUpdateRoundID {
							t.Errorf("Expected round ID %v, got %v", storeUpdateRoundID, storedPayload.Round.ID)
						}
					}).
					Return(nil).
					Times(1)
			},
		},
		{
			name:    "Invalid payload",
			payload: "invalid json",
			wantErr: true,
			errMsg:  "invalid payload",
			mockExpects: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1)
			},
		},
		{
			name:    "Failed to cancel existing scheduled events - continues anyway",
			payload: roundevents.RoundScheduleUpdatePayload{RoundID: storeUpdateRoundID},
			wantErr: false, // Continues despite cancellation error
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), storeUpdateRoundID).
					Return(fmt.Errorf("%s", storeUpdateCancelError)).
					Times(1)

				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), storeUpdateRoundID).
					Return(sampleRound, nil).
					Times(1)

				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateReschedule), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name:    "Failed to fetch round from database",
			payload: roundevents.RoundScheduleUpdatePayload{RoundID: storeUpdateRoundID},
			wantErr: true,
			errMsg:  "failed to fetch round for rescheduling",
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), storeUpdateRoundID).
					Return(nil).
					Times(1)

				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), storeUpdateRoundID).
					Return(nil, fmt.Errorf("%s", storeUpdateDBError)).
					Times(1)

				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Do(func(topic string, payload interface{}) {
						msg, ok := payload.(*message.Message)
						if !ok {
							t.Errorf("Expected *message.Message, got %T", payload)
							return
						}

						var errorPayload roundevents.RoundUpdateErrorPayload
						if err := json.Unmarshal(msg.Payload, &errorPayload); err != nil {
							t.Errorf("Failed to unmarshal error payload: %v", err)
							return
						}

						if !strings.Contains(errorPayload.Error, "failed to fetch round for rescheduling") {
							t.Errorf("Unexpected error message: %s", errorPayload.Error)
						}
					}).
					Return(nil).
					Times(1)
			},
		},
		{
			name:    "Failed to publish round.update.reschedule event",
			payload: roundevents.RoundScheduleUpdatePayload{RoundID: storeUpdateRoundID},
			wantErr: true,
			errMsg:  "failed to publish reschedule event",
			mockExpects: func() {
				mockEventBus.EXPECT().
					CancelScheduledMessage(gomock.Any(), storeUpdateRoundID).
					Return(nil).
					Times(1)

				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), storeUpdateRoundID).
					Return(sampleRound, nil).
					Times(1)

				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateReschedule), gomock.Any()).
					Return(fmt.Errorf("%s", storeUpdatePublishError)).
					Times(1)

				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Do(func(topic string, payload interface{}) {
						msg, ok := payload.(*message.Message)
						if !ok {
							t.Errorf("Expected *message.Message, got %T", payload)
							return
						}

						var errorPayload roundevents.RoundUpdateErrorPayload
						if err := json.Unmarshal(msg.Payload, &errorPayload); err != nil {
							t.Errorf("Failed to unmarshal error payload: %v", err)
							return
						}

						if !strings.Contains(errorPayload.Error, "failed to publish event round.update.reschedule") {
							t.Errorf("Unexpected error message: %s", errorPayload.Error)
						}
					}).
					Return(nil).
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
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Unexpected error: got %v, want message containing %v", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
