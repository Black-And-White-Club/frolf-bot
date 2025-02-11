package userhandlers

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/user/application/mocks"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleGetUserRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		setup   func(args args)
	}{
		{
			name: "Successful GetUserRequest",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678"}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					GetUser(gomock.Any(), args.msg, usertypes.DiscordID("123456789012345678")).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Unmarshalling Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{invalid_json}`)),
			},
			wantErr: true,
			setup:   func(args args) {},
		},
		{
			name: "GetUser Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678"}`)),
			},
			wantErr: true,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					GetUser(gomock.Any(), args.msg, usertypes.DiscordID("123456789012345678")).
					Return(errors.New("service error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleGetUserRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetUserRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserHandlers_HandleGetUserRoleRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		setup   func(args args)
	}{
		{
			name: "Successful GetUserRoleRequest",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678"}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					GetUserRole(gomock.Any(), args.msg, usertypes.DiscordID("123456789012345678")).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Unmarshalling Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{invalid_json}`)),
			},
			wantErr: true,
			setup:   func(args args) {},
		},
		{
			name: "GetUserRole Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678"}`)),
			},
			wantErr: true,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					GetUserRole(gomock.Any(), args.msg, usertypes.DiscordID("123456789012345678")).
					Return(errors.New("service error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleGetUserRoleRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetUserRoleRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
