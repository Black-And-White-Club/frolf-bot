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

func TestUserHandlers_HandleUserSignupRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

	testDiscordID := usertypes.DiscordID("12345")
	testTag := 123
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
			name: "Successful Signup Request",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{"discord_id":"12345","tag_number":123}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				mockUserService.EXPECT().
					CreateUser(gomock.Any(), args.msg, testDiscordID, &testTag).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "Unmarshalling Error",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{invalid_json}`)),
			},
			wantErr: true, // Expect an error
			setup:   func(args args) {},
		},
		{
			name: "CreateUser Error",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{"discord_id": "12345", "tag_number": 123}`)),
			},
			wantErr: true, // Expect an error
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				mockUserService.EXPECT().
					CreateUser(gomock.Any(), args.msg, testDiscordID, &testTag).
					Return(errors.New("some error")).
					Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleUserSignupRequest(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserHandlers_HandleUserCreated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

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
			name: "Successful User Created Event",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{"discord_id":"12345","tag_number":123}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
			},
		},
		{
			name: "Unmarshal Error",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{invalid_json}`)),
			},
			wantErr: true, // Expect an error
			setup:   func(args args) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleUserCreated(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserHandlers_HandleUserCreationFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := &UserHandlers{
		userService: mockUserService,
		logger:      logger,
	}

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
			name: "Successful User Creation Failed Event",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{"reason":"some reason"}`)),
			},
			wantErr: false,
			setup: func(args args) {
				args.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
			},
		},
		{
			name: "Unmarshal Error",
			args: args{
				msg: message.NewMessage(testCorrelationID, []byte(`{invalid_json}`)),
			},
			wantErr: true, // Expect an error
			setup:   func(args args) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.args)
			}

			err := h.HandleUserCreationFailed(tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
