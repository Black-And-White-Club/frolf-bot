package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	errorext "github.com/Black-And-White-Club/frolf-bot-shared/errors"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data (Update Round) ---

const (
	updateRoundID              = "some-round-id"
	updateCorrelationID        = "some-correlation-id"
	updateDBError       string = "db error"
)

var (
	updateNewTitle     string = "New Title"
	updateNewLocation  string = "New Location"
	updateOldLocation  string = "Old Location"
	updateOldEventType        = "Old Type"
	updateNow                 = time.Now().UTC().Truncate(time.Second)
	updateLater               = updateNow.Add(1 * time.Hour)
	updateOldStartTime        = &updateNow
	updateNewStartTime        = &updateLater

	// Pre-built payloads for common scenarios (Update Round)
	validUpdatePayload = roundevents.RoundUpdateRequestPayload{
		RoundID:   updateRoundID,
		Title:     &updateNewTitle,
		Location:  &updateNewLocation,
		StartTime: updateNewStartTime,
	}
	validFetchedPayload = roundevents.RoundFetchedPayload{
		Round: roundtypes.Round{
			ID:        updateRoundID,
			Title:     "Old Title",
			Location:  &updateOldLocation,
			EventType: &updateOldEventType,
			StartTime: updateOldStartTime,
			State:     roundtypes.RoundStateUpcoming,
		},
		RoundUpdateRequestPayload: validUpdatePayload,
	}
)

func TestRoundService_ValidateRoundUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl) // Not used in this test, but good practice
	mockErrorReporter := errorext.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
	s := &RoundService{
		EventBus:      mockEventBus,
		logger:        slog.Default(),
		ErrorReporter: mockErrorReporter,
		RoundDB:       mockRoundDB, // Include for completeness
	}
	tests := []struct {
		name          string
		payload       interface{}
		mockExpects   func()
		expectedEvent string // What event *should* be published
		wantErr       bool
		errMsg        string // Specific error message
	}{
		{
			name:          "Successful round update request validation",
			payload:       validUpdatePayload, // Use pre-built payload
			expectedEvent: roundevents.RoundUpdateValidated,
			wantErr:       false,
			mockExpects: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateValidated), gomock.Any()).
					Times(1).
					Return(nil) // Mock successful publishing
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: roundevents.RoundUpdateError,
			wantErr:       true,
			errMsg:        "invalid payload",
			mockExpects: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
		},
		{
			name: "Empty round ID",
			payload: roundevents.RoundUpdateRequestPayload{
				Title: &updateNewTitle,
			},
			expectedEvent: roundevents.RoundUpdateError,
			wantErr:       true,
			errMsg:        "round ID cannot be empty",
			mockExpects: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
		},
		{
			name: "No fields to update",
			payload: roundevents.RoundUpdateRequestPayload{
				RoundID: updateRoundID,
			},
			expectedEvent: roundevents.RoundUpdateError,
			wantErr:       true,
			errMsg:        "at least one field to update must be provided",
			mockExpects: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
		},
		{
			name:          "Publish RoundUpdateValidated event fails",
			payload:       validUpdatePayload,
			expectedEvent: roundevents.RoundUpdateValidated, // We still expect this event
			wantErr:       true,
			errMsg:        "failed to publish round.update.validated event",
			mockExpects: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateValidated), gomock.Any()).
					Return(fmt.Errorf("publish error")). // Simulate publish error
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, updateCorrelationID)

			// Set up mock expectations *before* calling the service function
			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := s.ValidateRoundUpdateRequest(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateRoundUpdateRequest() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateRoundUpdateRequest() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRoundUpdateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_UpdateRoundEntity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	mockErrorReporter := errorext.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
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
			name:          "Successful round entity update",
			payload:       validFetchedPayload, // Use pre-built payload
			expectedEvent: roundevents.RoundUpdated,
			wantErr:       false,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(updateRoundID)).
					Return(&validFetchedPayload.Round, nil). // Return initial round data
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(updateRoundID), gomock.Any()).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).
					Return(nil). // Mock success
					Times(1)
			},
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			expectedEvent: roundevents.RoundUpdateError,
			wantErr:       true,
			errMsg:        "invalid payload",
			mockDBSetup: func() {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
		},
		{
			name:    "Database error (GetRound)",
			payload: validFetchedPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(updateRoundID)).
					Return(nil, fmt.Errorf("failed to fetch existing round: %s", updateDBError)). // Simulate DB error
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
			expectedEvent: roundevents.RoundUpdateError,
			wantErr:       true,
			errMsg:        "failed to fetch existing round: " + updateDBError,
		},
		{
			name:    "Database error (UpdateRound)",
			payload: validFetchedPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(updateRoundID)).
					Return(&validFetchedPayload.Round, nil).
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(updateRoundID), gomock.Any()).
					Return(fmt.Errorf("failed to update round entity: %s", updateDBError)). // Simulate DB error
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdateError), gomock.Any()).
					Times(1).
					Return(nil) // Mock error publishing
			},
			expectedEvent: roundevents.RoundUpdateError,
			wantErr:       true,
			errMsg:        "failed to update round entity: " + updateDBError,
		},
		{
			name:    "Publish RoundUpdated event fails",
			payload: validFetchedPayload, // Use the valid payload
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(updateRoundID)).
					Return(&validFetchedPayload.Round, nil).
					Times(1)
				mockRoundDB.EXPECT().
					UpdateRound(gomock.Any(), gomock.Eq(updateRoundID), gomock.Any()).
					Return(nil). // Simulate successful DB update
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundUpdated), gomock.Any()).
					Return(fmt.Errorf("publish error")). // Simulate publish error
					Times(1)
			},
			expectedEvent: roundevents.RoundUpdated, // We still expect the event to be *attempted*
			wantErr:       true,
			errMsg:        "failed to publish round.updated event: failed to publish event round.updated: publish error", // Check for specific error

		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, updateCorrelationID)

			if tt.mockDBSetup != nil {
				tt.mockDBSetup()
			}

			err := s.UpdateRoundEntity(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdateRoundEntity() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("UpdateRoundEntity() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateRoundEntity() unexpected error: %v", err)
				}
			}
		})
	}
}
