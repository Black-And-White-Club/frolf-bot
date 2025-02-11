package roundhandlers

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundStored(t *testing.T) {
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
			name: "Successful round stored handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID: "some-round-id",
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ScheduleRoundEvents(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID: "some-round-id",
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ScheduleRoundEvents(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleRoundStored(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundStored() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundScheduled(t *testing.T) {
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
			name: "Successful round scheduled handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundScheduledPayload{
					RoundID: "some-round-id",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().PublishRoundCreated(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundScheduledPayload{
					RoundID: "some-round-id",
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().PublishRoundCreated(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleRoundScheduled(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundScheduled() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleUpdateDiscordEventID(t *testing.T) {
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
			name: "Successful Discord event ID update",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundEventCreatedPayload{
					RoundID:        "some-round-id",
					DiscordEventID: "new-discord-event-id",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().UpdateDiscordEventID(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundEventCreatedPayload{
					RoundID:        "some-round-id",
					DiscordEventID: "new-discord-event-id",
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().UpdateDiscordEventID(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleUpdateDiscordEventID(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleUpdateDiscordEventID() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}
