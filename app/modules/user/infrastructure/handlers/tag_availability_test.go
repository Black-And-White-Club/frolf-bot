package userhandlers

import (
	"encoding/json"
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

func TestUserHandlers_HandleTagAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678" // Valid Discord ID format
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
			name: "Successful Tag Available",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.TagAvailablePayload{
					DiscordID: usertypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				// Expect the service's CreateUser to be called with the correct arguments
				f.userService.EXPECT().CreateUser(a.msg.Context(), a.msg, usertypes.DiscordID(testDiscordID), &testTagNumber).Return(nil).Times(1)
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
			name: "Service Layer Error",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.TagAvailablePayload{
					DiscordID: usertypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)
				f.userService.EXPECT().CreateUser(a.msg.Context(), a.msg, usertypes.DiscordID(testDiscordID), &testTagNumber).Return(errors.New("service error")).Times(1)
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
			if err := h.HandleTagAvailable(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleTagAvailable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserHandlers_HandleTagUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mocks.NewMockService(ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testDiscordID := "123456789012345678" // Valid Discord ID format
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
			name: "Successful Tag Unavailable",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.TagUnavailablePayload{
					DiscordID: usertypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
					Reason:    "tag not available",
				}),
			},
			wantErr: false,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				// Expect the TagUnavailable method to be called with correct arguments
				f.userService.EXPECT().
					TagUnavailable(a.msg.Context(), a.msg, testTagNumber, usertypes.DiscordID(testDiscordID)).
					Return(nil).
					Times(1)
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
			setup: func(f fields, a args) {
				// No expectations for TagUnavailable, as unmarshaling should fail
			},
		},
		{
			name: "Service Layer Error",
			fields: fields{
				userService: mockUserService,
				logger:      logger,
			},
			args: args{
				msg: createTestMessageWithPayload(t, testCorrelationID, userevents.TagUnavailablePayload{
					DiscordID: usertypes.DiscordID(testDiscordID),
					TagNumber: testTagNumber,
					Reason:    "tag not available",
				}),
			},
			wantErr: true,
			setup: func(f fields, a args) {
				a.msg.Metadata.Set(middleware.CorrelationIDMetadataKey, testCorrelationID)

				// Expect the TagUnavailable method to return an error
				f.userService.EXPECT().
					TagUnavailable(a.msg.Context(), a.msg, testTagNumber, usertypes.DiscordID(testDiscordID)).
					Return(errors.New("service error")).
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
			if err := h.HandleTagUnavailable(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleTagUnavailable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create a test message with a payload and correlation ID in metadata
func createTestMessageWithPayload(t *testing.T, correlationID string, payload interface{}) *message.Message {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata = make(message.Metadata)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	return msg
}
