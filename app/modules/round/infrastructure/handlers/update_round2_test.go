package roundhandlers

import (
	"encoding/json"
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

var (
	RoundID roundtypes.ID = 1
)

func TestRoundHandlers_HandleRoundEntityUpdated(t *testing.T) {
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
			name: "Successful round entity updated handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundEntityUpdatedPayload{
					Round: roundtypes.Round{
						ID:    RoundID,
						Title: "Test Round",
						State: roundtypes.RoundStateUpcoming,
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().StoreRoundUpdate(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundEntityUpdatedPayload{
					Round: roundtypes.Round{
						ID:    RoundID,
						Title: "Test Round",
						State: roundtypes.RoundStateUpcoming,
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().StoreRoundUpdate(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleRoundEntityUpdated(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundEntityUpdated() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundScheduleUpdate(t *testing.T) {
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
		name        string
		fields      fields
		args        args
		wantErr     bool
		mockExpects func()
	}{
		{
			name: "Successful handling of RoundScheduleUpdate event",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundScheduleUpdatePayload{
						RoundID: RoundID,
					})
					return payload
				}()),
			},
			wantErr: false,
			mockExpects: func() {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Failed to unmarshal RoundScheduleUpdatePayload",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid json")),
			},
			wantErr:     true,
			mockExpects: func() {},
		},
		{
			name: "Failed to update scheduled round events",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), func() []byte {
					payload, _ := json.Marshal(roundevents.RoundScheduleUpdatePayload{
						RoundID: RoundID,
					})
					return payload
				}()),
			},
			wantErr: true,
			mockExpects: func() {
				mockRoundService.EXPECT().UpdateScheduledRoundEvents(gomock.Any(), gomock.Any()).Return(fmt.Errorf("update error")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &RoundHandlers{
				RoundService: tt.fields.RoundService,
				logger:       tt.fields.logger,
			}
			tt.mockExpects()
			if err := h.HandleRoundScheduleUpdate(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("RoundHandlers.HandleRoundScheduleUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
