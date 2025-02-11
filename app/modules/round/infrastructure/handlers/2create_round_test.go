package roundhandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

// --- Constants and Variables for Test Data ---
const (
	validatedCorrelationID = "test-correlation-id"
	validatedRoundID       = "test-round-id"
	validatedTitle         = "Test Round"
	validatedServiceError  = "service error"
)

var (
	validatedLocation          = "Test Location"
	validatedEventType         = "casual"
	validatedNow               = time.Now().UTC().Truncate(time.Second)
	validatedStartTime         = &validatedNow
	validatedEndTime           = &validatedNow
	validRoundValidatedPayload = roundevents.RoundValidatedPayload{
		RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
			Title:     validatedTitle,
			StartTime: validatedStartTime,
			Location:  &validatedLocation,
			EventType: &validatedEventType,
			EndTime:   validatedEndTime, // Now optional, but good to include in tests
		},
	}
)

func TestRoundHandlers_HandleRoundCreateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name        string
		payload     interface{}
		mockExpects func()
		wantErr     bool
		errMsg      string
	}{
		{
			name:    "Successful round create request",
			payload: validRoundValidatedPayload.RoundCreateRequestPayload, // Use pre-built payload
			wantErr: false,
			mockExpects: func() {
				mockRoundService.EXPECT().ValidateRoundRequest(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:    "Unmarshal error",
			payload: "invalid-payload", // Invalid JSON
			wantErr: true,
			errMsg:  "failed to unmarshal RoundCreateRequestPayload",
		},
		{
			name:    "Service layer error",
			payload: validRoundValidatedPayload.RoundCreateRequestPayload,
			wantErr: true,
			errMsg:  "failed to handle RoundCreateRequest event: " + validatedServiceError,
			mockExpects: func() {
				mockRoundService.EXPECT().ValidateRoundRequest(gomock.Any(), gomock.Any()).Return(fmt.Errorf(validatedServiceError)).Times(1)
			},
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
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, validatedCorrelationID)

			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := h.HandleRoundCreateRequest(msg) // Pass the message

			if tt.wantErr {
				if err == nil {
					t.Error("HandleRoundCreateRequest() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("HandleRoundCreateRequest() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("HandleRoundCreateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundValidated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name        string
		payload     interface{}
		mockExpects func()
		wantErr     bool
		errMsg      string
	}{
		{
			name:    "Successful round validated handling",
			payload: validRoundValidatedPayload,
			wantErr: false,
			mockExpects: func() {
				// Now expecting a call to StoreRound, NOT ParseDateTime
				mockRoundService.EXPECT().StoreRound(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name:    "Unmarshal error",
			payload: "invalid-payload", // Invalid JSON
			wantErr: true,
			errMsg:  "failed to unmarshal RoundValidatedPayload",
		},
		{
			name:    "Service layer error",
			payload: validRoundValidatedPayload, // Use constant
			wantErr: true,
			errMsg:  "failed to handle RoundValidated event: " + validatedServiceError,
			mockExpects: func() {
				// Now expecting StoreRound to return an error.
				mockRoundService.EXPECT().StoreRound(gomock.Any(), gomock.Any()).Return(fmt.Errorf(validatedServiceError)).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: mockRoundService, // Inject mock service
				logger:       logger,
			}

			// Create message with payload
			payloadBytes, _ := json.Marshal(tt.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, validatedCorrelationID)

			if tt.mockExpects != nil {
				tt.mockExpects()
			}

			err := h.HandleRoundValidated(msg)

			if tt.wantErr {
				if err == nil {
					t.Error("HandleRoundValidated() expected error, got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("HandleRoundValidated() error = %v, wantErrMsg containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("HandleRoundValidated() unexpected error: %v", err)
				}
			}
		})
	}
}
