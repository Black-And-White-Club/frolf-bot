package roundhandlers

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundDeleteRequest(t *testing.T) {
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
			name: "Successful round delete request handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundDeleteRequestPayload{
					RoundID: "some-round-id",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ValidateRoundDeleteRequest(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundDeleteRequestPayload{
					RoundID: "some-round-id",
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ValidateRoundDeleteRequest(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleRoundDeleteRequest(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundDeleteRequest() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleteValidated(t *testing.T) {
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
			name: "Successful round delete validated handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID: "some-round-id",
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().CheckRoundExists(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundDeleteValidatedPayload{
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RoundID: "some-round-id",
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().CheckRoundExists(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleRoundDeleteValidated(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundDeleteValidated() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundToDeleteFetched(t *testing.T) {
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
			name: "Successful round to delete fetched handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundToDeleteFetchedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						CreatedBy: "some-user-id",
					},
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RequestingUserDiscordID: "some-user-id",
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().CheckUserAuthorization(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundToDeleteFetchedPayload{
					Round: roundtypes.Round{
						ID:        "some-round-id",
						CreatedBy: "some-user-id",
					},
					RoundDeleteRequestPayload: roundevents.RoundDeleteRequestPayload{
						RequestingUserDiscordID: "some-user-id",
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().CheckUserAuthorization(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
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

			if err := h.HandleRoundToDeleteFetched(tt.args.msg); (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundToDeleteFetched() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}
