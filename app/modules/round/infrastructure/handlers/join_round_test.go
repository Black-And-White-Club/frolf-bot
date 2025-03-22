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

func TestRoundHandlers_HandleRoundParticipantJoinRequest(t *testing.T) {
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
			name: "Successful participant join request handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundtypes.ID(1),
					UserID:   "some-discord-id",
					Response: "some-response",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().CheckParticipantStatus(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid-payload")), // Use a simple invalid payload
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundtypes.ID(1),
					UserID:   "some-discord-id",
					Response: "some-response",
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().CheckParticipantStatus(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
			},
		},
		{
			name: "Late Join Detected",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinRequestPayload{
					RoundID:    roundtypes.ID(1),
					UserID:     "some-discord-id",
					Response:   "some-response",
					JoinedLate: func(b bool) *bool { return &b }(true), // Create a pointer to true
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().CheckParticipantStatus(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Tag Number Provided",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinRequestPayload{
					RoundID:   roundtypes.ID(1),
					UserID:    "some-discord-id",
					Response:  "some-response",
					TagNumber: func(i int) *int { return &i }(123),
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().CheckParticipantStatus(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "No Optional Fields",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundtypes.ID(1),
					UserID:   "some-discord-id",
					Response: "some-response",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().CheckParticipantStatus(gomock.Any(), a.msg).Return(nil).Times(1)
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

			err := h.HandleRoundParticipantJoinRequest(tt.args.msg)
			if (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoinRequest() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundParticipantJoinValidated(t *testing.T) {
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
			name: "Successful participant join validated handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinValidatedPayload{
					ParticipantJoinRequestPayload: roundevents.ParticipantJoinRequestPayload{
						RoundID:  1,
						UserID:   "some-discord-id",
						Response: "some-response",
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				// Set up expectation for the *new* message that will be sent.
				f.RoundService.EXPECT().RequestTagNumber(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid-payload")),
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinValidatedPayload{
					ParticipantJoinRequestPayload: roundevents.ParticipantJoinRequestPayload{
						RoundID:  1,
						UserID:   "some-discord-id",
						Response: "some-response",
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().RequestTagNumber(gomock.Any(), gomock.Any()).Return(fmt.Errorf("service error")).Times(1)
			},
		},
		{
			name: "Tag Number and JoinedLate",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinValidatedPayload{
					ParticipantJoinRequestPayload: roundevents.ParticipantJoinRequestPayload{
						RoundID:    1,
						UserID:     "some-discord-id",
						Response:   "some-response",
						TagNumber:  func(i int) *int { return &i }(123),
						JoinedLate: func(b bool) *bool { return &b }(true),
					},
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().RequestTagNumber(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "RequestTagNumber fails",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.ParticipantJoinValidatedPayload{
					ParticipantJoinRequestPayload: roundevents.ParticipantJoinRequestPayload{
						RoundID:  1,
						UserID:   "some-discord-id",
						Response: "some-response",
					},
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().RequestTagNumber(gomock.Any(), gomock.Any()).Return(fmt.Errorf("tag number error")).Times(1)
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

			err := h.HandleRoundParticipantJoinValidated(tt.args.msg)
			if (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundParticipantJoinValidated() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundTagNumberFound(t *testing.T) {
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
			name: "Successful round tag number found handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberFoundPayload{
					RoundID:   1,
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().ParticipantTagFound(gomock.Any(), a.msg).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal error",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte("invalid-payload")),
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberFoundPayload{
					RoundID:   1,
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().ParticipantTagFound(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
			},
		},
		{
			name: "Invalid RoundID",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberFoundPayload{
					RoundID:   1, // Use invalid round ID type
					UserID:    "some-discord-id",
					TagNumber: intPtr(1234),
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				//  unmarshalling error
			},
		},
		{
			name: "Zero TagNumber",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberFoundPayload{
					RoundID:   1,
					UserID:    "some-discord-id",
					TagNumber: intPtr(0), //  0 tag number.
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().ParticipantTagFound(gomock.Any(), a.msg).Return(nil).Times(1)
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

			err := h.HandleRoundTagNumberFound(tt.args.msg)
			if (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundTagNumberFound() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundTagNumberNotFound(t *testing.T) {
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
			name: "Successful round tag number not found handling",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberNotFoundPayload{
					UserID: "some-discord-id",
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ParticipantTagNotFound(gomock.Any(), a.msg).Return(nil).Times(1)
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
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberNotFoundPayload{
					UserID: "some-discord-id",
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				f.RoundService.EXPECT().ParticipantTagNotFound(gomock.Any(), a.msg).Return(fmt.Errorf("service error")).Times(1)
			},
		},
		{
			name: "Missing UserID",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberNotFoundPayload{ // Missing UserID
				}),
			},
			expectErr: true,
			mockExpects: func(f fields, a args) {
				//  unmarshalling error is expected.
			},
		},
		{
			name: "Empty UserID",
			fields: fields{
				RoundService: mockRoundService,
				logger:       logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, watermill.NewUUID(), roundevents.RoundTagNumberNotFoundPayload{
					UserID: "", // Empty UserID
				}),
			},
			expectErr: false,
			mockExpects: func(f fields, a args) {
				f.RoundService.EXPECT().ParticipantTagNotFound(gomock.Any(), a.msg).Return(nil).Times(1)
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

			err := h.HandleRoundTagNumberNotFound(tt.args.msg)
			if (err != nil) != tt.expectErr {
				t.Errorf("RoundHandlers.HandleRoundTagNumberNotFound() error = %v, wantErr %v", err, tt.expectErr)
			}
		})
	}
}
