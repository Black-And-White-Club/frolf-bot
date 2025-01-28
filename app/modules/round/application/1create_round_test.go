package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	eventbusmocks "github.com/Black-And-White-Club/tcr-bot/app/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	roundutil "github.com/Black-And-White-Club/tcr-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateRoundRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()
	validator := roundutil.NewRoundValidator() // Use the real validator

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
	}{
		{
			name: "Valid request",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundCreateRequestPayload{
					Title: "Valid Title",
					DateTime: roundtypes.RoundTimeInput{
						Date: time.Now().Format("2006-01-02"),
						Time: time.Now().Format("15:04"),
					},
				},
			},
			expectedEvent: roundevents.RoundValidated,
			expectErr:     false,
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
		},
		{
			name: "Missing title",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundCreateRequestPayload{
					DateTime: roundtypes.RoundTimeInput{
						Date: time.Now().Format("2006-01-02"),
						Time: time.Now().Format("15:04"),
					},
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
		},
		{
			name: "Missing date",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundCreateRequestPayload{
					Title: "Title",
					DateTime: roundtypes.RoundTimeInput{
						Time: time.Now().Format("15:04"),
					},
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
		},
		{
			name: "Missing time",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundCreateRequestPayload{
					Title: "Title",
					DateTime: roundtypes.RoundTimeInput{
						Date: time.Now().Format("2006-01-02"),
					},
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			// Set expectations for publishing events
			if tt.expectedEvent != "" && !tt.expectErr {
				mockEventBus.EXPECT().Publish(gomock.Eq(tt.expectedEvent), gomock.Any()).Return(nil).Times(1)
			} else if tt.expectErr {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			}

			// Create a RoundService instance with minimal dependencies
			s := &RoundService{
				EventBus:       mockEventBus,
				logger:         logger,
				eventUtil:      eventutil.NewEventUtil(),
				roundValidator: validator,
			}

			// Call the service function
			err := s.ValidateRoundRequest(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ValidateRoundRequest() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateRoundRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_ParseDateTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func()
	}{
		{
			name: "Valid date and time",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
				},
			},
			expectedEvent: roundevents.RoundDateTimeParsed,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDateTimeParsed), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid date format",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "25-12-2023",
							Time: "10:00",
						},
					},
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid time format",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00 PM",
						},
					},
				},
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish DateTimeParsed event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundDateTimeParsed), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())
			tt.mockExpects()

			s := &RoundService{
				EventBus:       mockEventBus,
				logger:         logger,
				eventUtil:      eventutil.NewEventUtil(),
				roundValidator: roundutil.NewRoundValidator(),
			}

			// Call the service function
			err := s.ParseDateTime(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("ParseDateTime() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("ParseDateTime() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRoundService_CreateRoundEntity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbusmocks.NewMockEventBus(ctrl)
	logger := slog.Default()

	type args struct {
		ctx     context.Context
		payload interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func()
	}{
		{
			name: "Valid round entity creation",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDateTimeParsedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
					StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
				},
			},
			expectedEvent: roundevents.RoundEntityCreated,
			expectErr:     false,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundEntityCreated), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: roundevents.RoundError,
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundError), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Publish RoundEntityCreated event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundDateTimeParsedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
					StartTime: time.Date(2023, 12, 25, 10, 0, 0, 0, time.UTC),
				},
			},
			expectedEvent: "",
			expectErr:     true,
			mockExpects: func() {
				mockEventBus.EXPECT().Publish(gomock.Eq(roundevents.RoundEntityCreated), gomock.Any()).Return(fmt.Errorf("publish error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			tt.mockExpects()

			s := &RoundService{
				EventBus:  mockEventBus,
				logger:    logger,
				eventUtil: eventutil.NewEventUtil(),
			}

			// Call the service function
			err := s.CreateRoundEntity(tt.args.ctx, msg)
			if tt.expectErr {
				if err == nil {
					t.Error("CreateRoundEntity() expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("CreateRoundEntity() unexpected error: %v", err)
				}
			}
		})
	}
}
