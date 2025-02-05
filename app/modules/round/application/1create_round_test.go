package roundservice

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot/app/eventbus/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
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
		shouldPublish bool
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
			shouldPublish: true,
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: "",
			shouldPublish: false,
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
			expectedEvent: "",
			shouldPublish: false,
		},
		{
			name: "Missing date",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundCreateRequestPayload{
					Title: "Valid Title",
					DateTime: roundtypes.RoundTimeInput{
						Time: time.Now().Format("15:04"),
					},
				},
			},
			expectedEvent: "",
			shouldPublish: false,
		},
		{
			name: "Missing time",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundCreateRequestPayload{
					Title: "Valid Title",
					DateTime: roundtypes.RoundTimeInput{
						Date: time.Now().Format("2006-01-02"),
					},
				},
			},
			expectedEvent: "",
			shouldPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			// Set up mock expectations
			if tt.shouldPublish {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(tt.expectedEvent), gomock.Any()).
					Times(1)
			}

			s := &RoundService{
				EventBus:       mockEventBus,
				logger:         logger,
				roundValidator: validator,
			}

			// Call the service function
			err := s.ValidateRoundRequest(tt.args.ctx, msg)
			if tt.shouldPublish {
				if err != nil {
					t.Errorf("ValidateRoundRequest() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("ValidateRoundRequest() expected error, got none")
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
		shouldPublish bool
	}{
		{
			name: "Valid date and time",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: time.Now().Format("2006-01-02"),
							Time: time.Now().Format("15:04"),
						},
					},
				},
			},
			expectedEvent: roundevents.RoundDateTimeParsed,
			shouldPublish: true,
		},
		{
			name: "Invalid payload",
			args: args{
				ctx:     context.Background(),
				payload: "invalid json",
			},
			expectedEvent: "",
			shouldPublish: false,
		},
		{
			name: "Invalid date format",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: "invalid-date",
							Time: time.Now().Format("15:04"),
						},
					},
				},
			},
			expectedEvent: "",
			shouldPublish: false,
		},
		{
			name: "Invalid time format",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: time.Now().Format("2006-01-02"),
							Time: "invalid-time",
						},
					},
				},
			},
			expectedEvent: "",
			shouldPublish: false,
		},
		{
			name: "Publish DateTimeParsed event fails",
			args: args{
				ctx: context.Background(),
				payload: roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Valid Title",
						DateTime: roundtypes.RoundTimeInput{
							Date: time.Now().Format("2006-01-02"),
							Time: time.Now().Format("15:04"),
						},
					},
				},
			},
			expectedEvent: roundevents.RoundDateTimeParsed,
			shouldPublish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare a mock message with the payload
			payloadBytes, _ := json.Marshal(tt.args.payload)
			msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, watermill.NewUUID())

			// Set up mock expectations
			if tt.shouldPublish {
				mockEventBus.EXPECT().
					Publish(gomock.Eq(tt.expectedEvent), gomock.Any()).
					Times(1)
			}

			s := &RoundService{
				EventBus: mockEventBus,
				logger:   logger,
			}

			// Call the service function
			err := s.ParseDateTime(tt.args.ctx, msg)
			if tt.shouldPublish {
				if err != nil {
					t.Errorf("ParseDateTime() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("ParseDateTime() expected error, got none")
				}
			}
		})
	}
}
