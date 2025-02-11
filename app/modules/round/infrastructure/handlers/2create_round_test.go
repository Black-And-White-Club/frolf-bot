package roundhandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundCreateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservice.NewMockService(ctrl)
	logger := slog.Default()

	type fields struct {
		RoundService *roundservice.MockService
		logger       *slog.Logger
	}

	type args struct {
		msg *message.Message
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func(f fields, a args)
	}{
		{
			name: "Successful round create request",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundCreateRequestPayload{
					Title: "Test Round",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")

				f.RoundService.EXPECT().ValidateRoundRequest(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), "invalid-payload"),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				// No expectations on the service layer as unmarshalling should fail first
			},
		},
		{
			name: "Service layer error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundCreateRequestPayload{
					Title: "Test Round",
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")

				f.RoundService.EXPECT().ValidateRoundRequest(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: tt.fields.RoundService,
				logger:       tt.fields.logger,
			}

			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields, tt.args)
			}

			if err := h.HandleRoundCreateRequest(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundCreateRequest() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundValidated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		RoundService *roundservice.MockService
		logger       *slog.Logger
	}

	type args struct {
		msg *message.Message
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func(f fields, a args)
	}{
		{
			name: "Successful round validated handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Test Round",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ParseDateTime(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), "invalid-payload"),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				// No expectations on the service layer as unmarshalling should fail first
			},
		},
		{
			name: "Service layer error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundValidatedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Test Round",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ParseDateTime(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: tt.fields.RoundService,
				logger:       tt.fields.logger,
			}

			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields, tt.args)
			}

			if err := h.HandleRoundValidated(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundValidated() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDateTimeParsed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoundService := roundservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		RoundService *roundservice.MockService
		logger       *slog.Logger
	}

	type args struct {
		msg *message.Message
	}

	tests := []struct {
		name          string
		fields        fields
		args          args
		expectedEvent string
		expectErr     bool
		mockExpects   func(f fields, a args)
	}{
		{
			name: "Successful round date time parsed handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundDateTimeParsedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Test Round",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
					StartTime: func(t time.Time) *time.Time { return &t }(time.Now()),
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().StoreRound(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), "invalid-payload"),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				// No expectations on the service layer as unmarshalling should fail first
			},
		},
		{
			name: "Service layer error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundDateTimeParsedPayload{
					RoundCreateRequestPayload: roundevents.RoundCreateRequestPayload{
						Title: "Test Round",
						DateTime: roundtypes.RoundTimeInput{
							Date: "2023-12-25",
							Time: "10:00",
						},
					},
					StartTime: func(t time.Time) *time.Time { return &t }(time.Now()),
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().StoreRound(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: tt.fields.RoundService,
				logger:       tt.fields.logger,
			}

			if tt.mockExpects != nil {
				tt.mockExpects(tt.fields, tt.args)
			}

			if err := h.HandleRoundDateTimeParsed(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundDateTimeParsed() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

// Helper function to create a test message with a payload
func createTestMessageWithPayload(t *testing.T, correlationID string, payload interface{}) *message.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata = make(message.Metadata)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	return msg
}
