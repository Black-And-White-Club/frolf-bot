package userhandlers

import (
	"errors"
	"fmt"
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

func TestUserHandlers_HandleUserPermissionsCheckRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

	testUserID := usertypes.DiscordID("123456789012345678") // Valid Discord ID
	testRole := usertypes.UserRoleEnum("admin")             // Use the enum type
	testRequesterID := "requester456"
	testCorrelationID := watermill.NewUUID()

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
			name: "Successful Permission Check Request",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(fmt.Sprintf(`{"discord_id":"%s", "role":"%s", "requester_id":"%s"}`, testUserID, testRole, testRequesterID))),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				mockUserService.EXPECT().
					CheckUserPermissionsInDB(gomock.Any(), testUserID, testRole, testRequesterID, testCorrelationID). // Expect the enum type
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
			name: "CheckUserPermissionsInDB Error",
			args: args{
				msg: message.NewMessage(watermill.NewUUID(), []byte(fmt.Sprintf(`{"discord_id":"%s", "role":"%s", "requester_id":"%s"}`, testUserID, testRole, testRequesterID))),
			},
			wantErr: true,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				mockUserService.EXPECT().
					CheckUserPermissionsInDB(gomock.Any(), testUserID, testRole, testRequesterID, testCorrelationID). // Expect the enum type
					Return(errors.New("database error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleUserPermissionsCheckRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleUserPermissionsCheckRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestUserHandlers_HandleUserPermissionsCheckFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

	testCorrelationID := watermill.NewUUID()

	t.Run("Successful Handling of Permissions Check Failed", func(t *testing.T) {
		msg := message.NewMessage(testCorrelationID, []byte(`{"user_id":"user123", "role":"admin", "requester_id":"requester456", "reason":"Some reason"}`))
		msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

		err := h.HandleUserPermissionsCheckFailed(msg)
		if err != nil {
			t.Errorf("HandleUserPermissionsCheckFailed() error = %v, wantErr %v", err, false)
		}
	})

	t.Run("Unmarshal Error", func(t *testing.T) {
		msg := message.NewMessage(testCorrelationID, []byte(`{invalid_json}`))
		msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

		err := h.HandleUserPermissionsCheckFailed(msg)
		if err == nil {
			t.Errorf("HandleUserPermissionsCheckFailed() expected error, got nil")
		}
	})
}
