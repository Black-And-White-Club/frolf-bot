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
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

const (
	validTitle                     = "Test Round"
	validRoundID     roundtypes.ID = 1
	duplicateRoundID               = "duplicate-round-id"
	correlationID                  = "some-correlation-id" // Use a constant correlation ID
	dbErrorMessage   string        = "some db error"       // Constant for the error message
)

var (
	validLocation    roundtypes.Location    = "Test Park"
	validDescription roundtypes.Description = "Test Description"
	validUserID      roundtypes.UserID      = "test-user-id"
	validChannelID                          = "test-channel-id"
	validTimezone    roundtypes.Timezone    = "America/New_York"

	now                         = time.Now().UTC().Truncate(time.Second) // Truncate for consistent comparisons
	validStartTime    time.Time = now.Add(2 * time.Hour)                 // Ensure start time is in the future
	validStartTimeStr           = validStartTime.Format(time.RFC3339)

	// Define valid CreateRoundRequestedPayload for reuse
	validCreateRequestedPayload = roundevents.CreateRoundRequestedPayload{
		Title:       validTitle,
		Description: validDescription,
		Location:    validLocation,
		StartTime:   validStartTimeStr,
		UserID:      validUserID,
		ChannelID:   validChannelID,
		Timezone:    validTimezone,
	}

	// Define valid RoundValidatedPayload for reuse
	validValidatedPayload = roundevents.RoundValidatedPayload{
		CreateRoundRequestedPayload: validCreateRequestedPayload,
	}

	// Define Round
	validRound = roundtypes.Round{
		ID:          validRoundID,
		Title:       roundtypes.Title(validTitle),
		Description: &validDescription,
		Location:    &validLocation,
		StartTime:   (*roundtypes.StartTime)(&validStartTime),
		CreatedBy:   roundtypes.UserID(validUserID),
		State:       roundtypes.RoundStateUpcoming,
	}

	// Define valid EntityCreatedPayload for reuse
	validEntityCreatedPayload = roundevents.RoundEntityCreatedPayload{
		Round:            validRound,
		DiscordChannelID: validChannelID,
		DiscordGuildID:   "",
	}
)

func TestRoundService_ValidateRoundRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	mockErrorReporter := errors.NewErrorReporter(mockEventBus, *slog.Default(), "serviceName", "environment")
	validator := roundutil.NewMockRoundValidator(ctrl)

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
		setupMocks    func()
		expectedEvent string
		shouldPublish bool
		wantErr       bool
		errMsg        string
	}{
		{
			name:    "Valid request",
			payload: validCreateRequestedPayload,
			setupMocks: func() {
				validator.EXPECT().
					ValidateRoundInput(gomock.Any()).
					Return([]string{}) // Return empty errors array for valid input

				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundValidated), gomock.Any()).
					Times(1).
					Return(nil) // Mock successful publishing
			},
			expectedEvent: roundevents.RoundValidated,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name:          "Invalid payload",
			payload:       "invalid json",
			setupMocks:    func() {}, // No mocks needed for this case
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "invalid payload",
		},
		{
			name: "Missing title",
			payload: roundevents.CreateRoundRequestedPayload{
				StartTime: validStartTimeStr,
				UserID:    validUserID,
				Location:  validLocation,
			},
			setupMocks: func() {
				validator.EXPECT().
					ValidateRoundInput(gomock.Any()).
					Return([]string{"Title is required"})

				// Set up expectations for the event bus
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundValidationFailed), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundValidationFailed,
			shouldPublish: true,
			wantErr:       false, // The error is handled by publishing a failure event
		},
		{
			name: "Missing start time",
			payload: roundevents.CreateRoundRequestedPayload{
				Title:    validTitle,
				UserID:   validUserID,
				Location: validLocation,
			},
			setupMocks:    func() {}, // No mocks needed for this case
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "missing required field: start_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

			err := s.ValidateRoundRequest(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidateRoundRequest() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
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

func TestRoundService_ProcessValidatedRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	mockTimeParser := roundutil.NewMockTimeParserInterface(ctrl)
	logger := slog.Default()

	s := &RoundService{
		EventBus: mockEventBus,
		RoundDB:  mockRoundDB,
		logger:   logger,
	}

	tests := []struct {
		name          string
		payload       any
		mockSetup     func()
		expectedEvent string
		shouldPublish bool
		wantErr       bool
		errMsg        string
	}{
		{
			name:    "Successful processing",
			payload: validValidatedPayload,
			mockSetup: func() {
				mockTimeParser.EXPECT().
					ParseUserTimeInput(gomock.Eq(validStartTimeStr), gomock.Eq(validTimezone), gomock.Any()).
					Return(validStartTime.Unix(), nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundEntityCreated), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundEntityCreated,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name:    "Time parsing error",
			payload: validValidatedPayload,
			mockSetup: func() {
				mockTimeParser.EXPECT().
					ParseUserTimeInput(gomock.Eq(validStartTimeStr), gomock.Eq(validTimezone), gomock.Any()).
					Return(int64(0), fmt.Errorf("failed to parse time")).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundValidationFailed), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundValidationFailed,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name:          "Invalid Payload",
			payload:       "invalid",
			mockSetup:     func() {},
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

			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			err := s.ProcessValidatedRound(context.Background(), msg, mockTimeParser)

			if tt.wantErr {
				if err == nil {
					t.Error("ProcessValidatedRound() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ProcessValidatedRound() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ProcessValidatedRound() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_StoreRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
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
			payload: validEntityCreatedPayload,
			mockDBSetup: func() {
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
			name: "Missing description",
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					ID:        validRoundID,
					Title:     roundtypes.Title(validTitle),
					Location:  &validLocation,
					StartTime: (*roundtypes.StartTime)(&validStartTime),
					CreatedBy: roundtypes.UserID(validUserID),
					State:     roundtypes.RoundStateUpcoming,
				},
				DiscordChannelID: validChannelID,
			},
			mockDBSetup:   func() {},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "description is required but was nil",
		},
		{
			name: "Missing location",
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					ID:          validRoundID,
					Title:       roundtypes.Title(validTitle),
					Description: &validDescription,
					StartTime:   (*roundtypes.StartTime)(&validStartTime),
					CreatedBy:   roundtypes.UserID(validUserID),
					State:       roundtypes.RoundStateUpcoming,
				},
				DiscordChannelID: validChannelID,
			},
			mockDBSetup:   func() {},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "location is required but was nil",
		},
		{
			name: "Missing start time",
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					ID:          validRoundID,
					Title:       roundtypes.Title(validTitle),
					Description: &validDescription,
					Location:    &validLocation,
					CreatedBy:   roundtypes.UserID(validUserID),
					State:       roundtypes.RoundStateUpcoming,
				},
				DiscordChannelID: validChannelID,
			},
			mockDBSetup:   func() {},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "startTime is required but was nil",
		},
		{
			name:    "Database error (CreateRound)",
			payload: validEntityCreatedPayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					CreateRound(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed to store round in database: %s", dbErrorMessage)).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundCreationFailed), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundCreationFailed,
			shouldPublish: true,
			wantErr:       false, // Error is handled by publishing a failure event
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

func TestRoundService_UpdateEventMessageID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	mockRoundDB := rounddb.NewMockRoundDBInterface(ctrl)
	logger := slog.Default()

	s := &RoundService{
		EventBus: mockEventBus,
		RoundDB:  mockRoundDB,
		logger:   logger,
	}

	validEventMessageID := "discord-message-id-123"
	validUpdatePayload := roundevents.RoundEventMessageIDUpdatedPayload{
		RoundID:        validRoundID,
		EventMessageID: validEventMessageID,
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
			name:    "Successful update",
			payload: validUpdatePayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					UpdateEventMessageID(gomock.Any(), gomock.Eq(validRoundID), gomock.Eq(validEventMessageID)).
					Return(nil).
					Times(1)
				mockEventBus.EXPECT().
					Publish(gomock.Eq(roundevents.RoundEventMessageIDUpdated), gomock.Any()).
					Times(1).
					Return(nil)
			},
			expectedEvent: roundevents.RoundEventMessageIDUpdated,
			shouldPublish: true,
			wantErr:       false,
		},
		{
			name: "Invalid RoundID",
			payload: roundevents.RoundEventMessageIDUpdatedPayload{
				RoundID:        0, // Invalid ID
				EventMessageID: validEventMessageID,
			},
			mockDBSetup:   func() {},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "invalid RoundID in payload",
		},
		{
			name:    "Round not found",
			payload: validUpdatePayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					UpdateEventMessageID(gomock.Any(), gomock.Eq(validRoundID), gomock.Eq(validEventMessageID)).
					Return(sql.ErrNoRows).
					Times(1)
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "round not found",
		},
		{
			name:    "Database error",
			payload: validUpdatePayload,
			mockDBSetup: func() {
				mockRoundDB.EXPECT().
					UpdateEventMessageID(gomock.Any(), gomock.Eq(validRoundID), gomock.Eq(validEventMessageID)).
					Return(fmt.Errorf("database error")).
					Times(1)
			},
			expectedEvent: "",
			shouldPublish: false,
			wantErr:       true,
			errMsg:        "failed to update Discord event ID",
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

			err := s.UpdateEventMessageID(context.Background(), msg)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdateEventMessageID() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("UpdateEventMessageID() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateEventMessageID() unexpected error: %v", err)
				}
			}
		})
	}
}
