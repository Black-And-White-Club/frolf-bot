package userhandlers

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/application/mocks"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleUserRoleUpdateRequest(t *testing.T) {
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
			name: "Successful Role Update Request",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678", "role":"admin", "requester_id":"requester456"}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					UpdateUserRole(gomock.Any(), args.msg, usertypes.DiscordID("123456789012345678"), "admin", "requester456").
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
			name: "UpdateUserRole Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678", "role":"admin", "requester_id":"requester456"}`)),
			},
			wantErr: true,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					UpdateUserRole(gomock.Any(), args.msg, usertypes.DiscordID("123456789012345678"), "admin", "requester456").
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

			err := h.HandleUserRoleUpdateRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserRoleUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserHandlers_HandleUserPermissionsCheckResponse(t *testing.T) {
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
			name: "Successful Permission Check Response - Has Permission",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678", "role":"admin", "has_permission":true}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					UpdateUserRoleInDatabase(gomock.Any(), args.msg, "123456789012345678", "admin").
					Return(nil).
					Times(1)
				mockUserService.EXPECT().
					PublishUserRoleUpdated(gomock.Any(), args.msg, "123456789012345678", "admin").
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Successful Permission Check Response - No Permission",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678", "role":"admin", "has_permission":false}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					PublishUserRoleUpdateFailed(gomock.Any(), args.msg, "123456789012345678", "admin", "User does not have required permission").
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
			name: "UpdateUserRoleInDatabase Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"123456789012345678", "role":"admin", "has_permission":true}`)),
			},
			wantErr: true, // Expecting an error from UpdateUserRoleInDatabase
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
				mockUserService.EXPECT().
					UpdateUserRoleInDatabase(gomock.Any(), args.msg, "123456789012345678", "admin").
					Return(errors.New("database error")).
					Times(1)
				mockUserService.EXPECT().
					PublishUserRoleUpdateFailed(gomock.Any(), args.msg, "123456789012345678", "admin", "database error").
					Return(nil).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleUserPermissionsCheckResponse(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserPermissionsCheckResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserHandlers_HandleUserRoleUpdateFailed(t *testing.T) {
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
			name: "Successful User Role Update Failed Handling",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(`{"discord_id":"user123", "role":"admin", "reason":"some reason"}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, "test-correlation-id")
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleUserRoleUpdateFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserRoleUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
