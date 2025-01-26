package userhandlers

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user/application/mocks"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
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

	testDiscordID := usertypes.DiscordID("123456789012345678") // Valid Discord ID
	testTagNumber := 123
	testCorrelationID := watermill.NewUUID()

	type fields struct {
		userService *mocks.MockService
		logger      *slog.Logger
	}
	type args struct {
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func(f fields, a args)
	}{
		{
			name: "Successful Signup with Tag",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.UserSignupRequestPayload{
					DiscordID: usertypes.DiscordID(testDiscordID),
					TagNumber: &testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				// Expect CheckTagAvailability to be called because a tag is provided
				f.userService.EXPECT().CheckTagAvailability(a.msg.Context(), a.msg, testTagNumber, gomock.Any()).Return(nil).Times(1)
			},
		},
		{
			name: "Successful Signup without Tag",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.UserSignupRequestPayload{
					DiscordID: testDiscordID,
					TagNumber: nil, // No tag provided
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				// Expect CreateUser to be called because no tag is provided
				f.userService.EXPECT().CreateUser(a.msg.Context(), a.msg, testDiscordID, nil).Return(nil).Times(1)
			},
		},
		{
			name: "Unmarshal Error",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, "invalid-payload"),
			},
			wantErr: true,
			setup:   func(f fields, a args) {},
		},
		{
			name: "CheckTagAvailability Error",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.UserSignupRequestPayload{
					DiscordID: testDiscordID,
					TagNumber: &testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.userService.EXPECT().CheckTagAvailability(a.msg.Context(), a.msg, testTagNumber, gomock.Any()).Return(errors.New("error checking tag")).Times(1)
			},
		},
		{
			name: "CreateUser Error",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.UserSignupRequestPayload{
					DiscordID: usertypes.DiscordID(testDiscordID), // Explicitly use usertypes.DiscordID
					TagNumber: nil,                                // No tag provided
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.userService.EXPECT().
					CreateUser(a.msg.Context(), a.msg, usertypes.DiscordID(testDiscordID), nil). // Expect usertypes.DiscordID
					Return(errors.New("error creating user")).
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
			if err := h.HandleUserSignupRequest(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleUserSignupRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
