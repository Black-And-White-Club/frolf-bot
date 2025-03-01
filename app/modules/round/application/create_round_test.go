package roundservice

import (
	"context"
	"database/sql"
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
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

const (
	validTitle       string = "Test Round"
	validRoundID            = "some-round-id"
	duplicateRoundID        = "duplicate-round-id"
	correlationID           = "some-correlation-id" // Use a constant correlation ID
	dbErrorMessage   string = "some db error"       // Constant for the error message
)

var (
	validLocation    = "Test Park"
	validDescription = "Test Description"

	now            = time.Now().UTC().Truncate(time.Second) // Truncate for consistent comparisons
	validStartTime = now.Add(2 * time.Hour)                 // Ensure start time is in the future

	// Define a valid RoundCreateRequestPayload for reuse.
	validCreatePayload = roundevents.RoundCreateRequestPayload{
		Title:       validTitle,
		StartTime:   &validStartTime,
		Location:    &validLocation,
		Description: &validDescription,
	}
	// Define Round
	validRound = roundtypes.Round{
		ID:        validRoundID,
		Title:     validTitle,
		Location:  &validLocation,
		StartTime: &validStartTime,
		State:     roundtypes.RoundStateUpcoming,
	}
	validStoredPayload = roundevents.RoundEntityCreatedPayload{Round: validRound}
)

func TestRoundService_ValidateRoundRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	mockErrorReporter := errors.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
	validator := roundutil.NewRoundValidator()

	s := &RoundService{
		RoundDB:        mockRoundDB,
		EventBus:       mockEventBus,
		logger:         slog.Default(),
		roundValidator: validator,
		ErrorReporter:  mockErrorReporter,
	}
	tests := []struct {
		name          string
		payload       interface{}
		expectedEvent string
		shouldPublish bool
		wantErr       bool
		errMsg        string // More specific error checking
	}{
		{
			name:          "Valid request",
			payload:       validCreatePayload, // Use the constant
			expectedEvent: roundevents.RoundValidated,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json", // Invalid JSON
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
		},
		{
			name: "Missing title",
			payload: roundevents.RoundCreateRequestPayload{ // No title
				StartTime: &validStartTime,
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "validation errors: [title cannot be empty]",
		},
		{
			name: "Missing start time",
			payload: roundevents.RoundCreateRequestPayload{ // No StartTime
				Title: validTitle,
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "missing required field: start_time",
		},
		{
			name: "Missing end time",
			payload: roundevents.RoundCreateRequestPayload{
				Title:     "Valid Title",
				StartTime: &validStartTime,
			},
			expectedEvent: roundevents.RoundValidated,
			shouldPublish: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID) // Use constant

			if tt.shouldPublish {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(tt.expectedEvent), gomock.Any()).
					Times(1).
					Return(nil) // Mock successful publishing
			}

			err := s.ValidateRoundRequest(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateRoundRequest() expected error, got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ValidateRoundRequest() error = %v, wantErrMsg %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRoundRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_StoreRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDB(ctrl)
	logger := slog.Default()

	s := &RoundService{
		EventBus: mockEventBus,
		RoundDB:  mockRoundDB,
		logger:   logger,
	}

	tests := []struct {
		name          string
		payload       any
		mockDBSetup   func()
		expectedEvent string
		shouldPublish bool
		wantErr       bool
		errMsg        string
	}{
		{
			name:    "Successful round storage",
			payload: validStoredPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(validRoundID)).
					Return(nil, sql.ErrNoRows).
					Times(1)
				mockRoundDB.EXPECT().
					CreateRound(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundStored), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundStored,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name:    "Duplicate round",
			payload: validStoredPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(validRoundID)).
					Return(&roundtypes.Round{}, nil). // Simulate round already exists
					Times(1)
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "round already exists",
		},
		{
			name:    "Database error (GetRound)",
			payload: validStoredPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(validRoundID)).
					Return(nil, fmt.Errorf("failed to check for existing round: %s", dbErrorMessage)).
					Times(1)
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "failed to check for existing round: " + dbErrorMessage,
		},
		{
			name:    "Database error (CreateRound)",
			payload: validStoredPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					GetRound(gomock.Any(), gomock.Eq(validRoundID)).
					Return(nil, sql.ErrNoRows).
					Times(1)
				mockRoundDB.EXPECT().
					CreateRound(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed to store round in database: %s", dbErrorMessage)).
					Times(1)
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "failed to store round in database: " + dbErrorMessage,
		},
		{
			name:          "Invalid Payload",
			payload:       "invalid",
			mockDBSetup:   func() {},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "invalid payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

			if tt.mockDBSetup != nil {
				tt.mockDBSetup()
			}

			err := s.StoreRound(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("StoreRound() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("StoreRound() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("StoreRound() unexpected error: %v", err)
				}
			}
		})
	}
}
