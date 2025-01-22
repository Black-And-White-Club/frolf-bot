package userhandlers

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleCheckTagAvailabilityRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := userservice.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	type fields struct {
		userService *userservice.MockService
		logger      *slog.Logger
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*message.Message
		wantErr bool
		setup   func(f fields, args args)
	}{
		{
			name: "Successful tag availability check",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"tag_number":123}`)),
			},
			want:    nil,
			wantErr: false,
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")

				mockUserService.EXPECT().
					CheckTagAvailability(gomock.Any(), args.msg, 123).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Failed to unmarshal event",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"invalid_json"`)),
			},
			want:    nil,
			wantErr: true, // Expect an error
			setup: func(f fields, args args) {
				// No expectations needed as UnmarshalPayload will return an error
			},
		},
		{
			name: "Failed to check tag availability",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"tag_number":123}`)),
			},
			want:    nil,
			wantErr: true, // Expect an error
			setup: func(f fields, args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")

				mockUserService.EXPECT().
					CheckTagAvailability(gomock.Any(), args.msg, 123).
					Return(errors.New("some error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &UserHandlers{
				userService: tt.fields.userService,
				logger:      tt.fields.logger,
			}

			if tt.setup != nil {
				tt.setup(tt.fields, tt.args)
			}

			err := h.HandleCheckTagAvailabilityRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleCheckTagAvailabilityRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}
